package updater

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"redshell/internal/preferences"
)

const (
	httpDefaultTimeout      = 30 * time.Second
	checksumsMaxBytes       = 1 << 20 // 1 MiB ceiling on checksums file size
	defaultIntervalFallback = 6 * time.Hour
)

// EventEmitter delivers an updater:* runtime event to the frontend. Provided
// by the Wails app wrapper; left as a no-op when the service runs without a
// UI (e.g. unit tests).
type EventEmitter func(name string, data any)

// SwapFunc replaces a running executable with a verified new one. Defaults
// to the package-level Swap; tests inject a fake.
type SwapFunc func(currentPath, newPath string) error

// SpawnFunc starts a detached child process at exePath. Defaults to
// exec.Command(exePath).Start(); tests inject a fake.
type SpawnFunc func(exePath string) error

// InstallerSpawnFunc launches the NSIS installer with elevation. Defaults
// to SpawnInstaller (which uses ShellExecute "runas" on Windows). Tests
// inject a fake to avoid actually invoking ShellExecute.
type InstallerSpawnFunc func(installerPath string, args []string) error

// Options are non-required dependencies of the Service. Any zero field is
// filled with a production default at construction.
type Options struct {
	HTTPClient     *http.Client
	Now            func() time.Time
	Swap           SwapFunc
	Spawn          SpawnFunc
	InstallerSpawn InstallerSpawnFunc
}

// Service orchestrates the auto-update flow: periodic checks, install on
// demand, integration with preferences observers and the Wails close-intercept.
type Service struct {
	prefs          *preferences.Service
	runningVersion string
	exePath        string
	httpClient     *http.Client
	providers      map[string]Provider
	swap           SwapFunc
	spawn          SpawnFunc
	installerSpawn InstallerSpawnFunc
	now            func() time.Time

	emit       EventEmitter
	quitApp    func()
	cancelLoop context.CancelFunc
	loopDone   chan struct{}

	inProgress atomic.Bool

	mu         sync.Mutex
	lastResult *Release

	manualCheckCh  chan struct{}
	prefsChangedCh chan struct{}
	snapshot       autoUpdateSnapshot
}

type autoUpdateSnapshot struct {
	source        string
	intervalHours int
	enabled       bool
}

// NewService is the production constructor. It builds GitHub and GitLab
// providers from the persisted preferences. Use NewServiceWithProviders in
// tests when you need to inject httptest-backed providers.
func NewService(prefs *preferences.Service, runningVersion, exePath string, opts Options) (*Service, error) {
	if prefs == nil {
		return nil, errors.New("updater: prefs is required")
	}
	if exePath == "" {
		return nil, errors.New("updater: exePath is required")
	}
	au, err := prefs.GetAutoUpdate()
	if err != nil {
		return nil, fmt.Errorf("updater: read prefs: %w", err)
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: httpDefaultTimeout}
	}
	providers, err := buildProviders(au, httpClient)
	if err != nil {
		return nil, err
	}
	return newServiceWithProviders(prefs, runningVersion, exePath, providers, opts, httpClient), nil
}

// NewServiceWithProviders accepts pre-built providers and is used by tests
// to wire httptest.Server endpoints into the service.
func NewServiceWithProviders(prefs *preferences.Service, runningVersion, exePath string, providers map[string]Provider, opts Options) (*Service, error) {
	if prefs == nil {
		return nil, errors.New("updater: prefs is required")
	}
	if exePath == "" {
		return nil, errors.New("updater: exePath is required")
	}
	if providers == nil {
		return nil, errors.New("updater: providers is required")
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: httpDefaultTimeout}
	}
	return newServiceWithProviders(prefs, runningVersion, exePath, providers, opts, httpClient), nil
}

func newServiceWithProviders(prefs *preferences.Service, runningVersion, exePath string, providers map[string]Provider, opts Options, httpClient *http.Client) *Service {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	swap := opts.Swap
	if swap == nil {
		swap = Swap
	}
	spawn := opts.Spawn
	if spawn == nil {
		spawn = defaultSpawn
	}
	installerSpawn := opts.InstallerSpawn
	if installerSpawn == nil {
		installerSpawn = SpawnInstaller
	}
	au, _ := prefs.GetAutoUpdate()
	return &Service{
		prefs:          prefs,
		runningVersion: runningVersion,
		exePath:        exePath,
		httpClient:     httpClient,
		providers:      providers,
		swap:           swap,
		spawn:          spawn,
		installerSpawn: installerSpawn,
		now:            now,
		manualCheckCh:  make(chan struct{}, 1),
		prefsChangedCh: make(chan struct{}, 1),
		snapshot: autoUpdateSnapshot{
			source:        au.Source,
			intervalHours: au.IntervalHours,
			enabled:       au.Enabled,
		},
	}
}

func buildProviders(au preferences.AutoUpdate, httpClient *http.Client) (map[string]Provider, error) {
	assetName := AssetNameFor(runtime.GOOS, runtime.GOARCH)
	// Installer asset name is best-effort — only Windows AMD64 has one
	// today. On unsupported OS/arch we leave it empty so providers don't
	// look for a non-existent asset.
	installerName, _ := InstallerAssetNameFor(runtime.GOOS, runtime.GOARCH)
	gh, err := NewGitHubProvider(au.GithubRepo, assetName, httpClient)
	if err != nil {
		return nil, fmt.Errorf("build github provider: %w", err)
	}
	gh.SetInstallerAssetName(installerName)
	gl, err := NewGitLabProvider(au.GitlabHost, au.GitlabProject, assetName, httpClient)
	if err != nil {
		return nil, fmt.Errorf("build gitlab provider: %w", err)
	}
	gl.SetInstallerAssetName(installerName)
	return map[string]Provider{
		preferences.AutoUpdateSourceGitHub: gh,
		preferences.AutoUpdateSourceGitLab: gl,
	}, nil
}

func defaultSpawn(exePath string) error {
	cmd := exec.Command(exePath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// Start runs the cleanup pass, install-dir writability check, registers
// preference observers, and spawns the ticker goroutine. emit and quitApp
// must be supplied by the Wails app wrapper.
func (s *Service) Start(ctx context.Context, emit EventEmitter, quitApp func()) error {
	if emit == nil {
		emit = func(string, any) {}
	}
	if quitApp == nil {
		quitApp = func() {}
	}
	s.emit = emit
	s.quitApp = quitApp

	if err := CleanupStale(s.exePath); err != nil {
		s.emit("updater:error", map[string]any{
			"stage":   "cleanup",
			"message": err.Error(),
		})
	}

	// The writability probe only applies to portable builds. Installer
	// builds (BuildKind == "installer") use the elevated silent-install
	// pathway, which does NOT need the install directory to be writable
	// by the current user — UAC takes care of elevation at install time.
	if IsPortable() && !IsWritable(filepath.Dir(s.exePath)) {
		s.emit("updater:manual-required", map[string]any{
			"reason":  "install directory is not writable by the current user",
			"exePath": s.exePath,
		})
		return nil
	}

	s.prefs.OnChange(s.onPrefsChange)

	loopCtx, cancel := context.WithCancel(ctx)
	s.cancelLoop = cancel
	s.loopDone = make(chan struct{})
	go s.runLoop(loopCtx)
	return nil
}

// Stop tears down the ticker goroutine. Safe to call multiple times.
func (s *Service) Stop() {
	if s.cancelLoop != nil {
		s.cancelLoop()
	}
	if s.loopDone != nil {
		<-s.loopDone
	}
}

// InProgress reports whether the rename swap has begun. The Wails close
// intercept consults this to bypass the close-behavior prompt.
func (s *Service) InProgress() bool {
	return s.inProgress.Load()
}

// CheckNow signals the run loop to fire an immediate check. Returns
// without blocking if a check is already pending.
func (s *Service) CheckNow() {
	select {
	case s.manualCheckCh <- struct{}{}:
	default:
	}
}

// SkipVersion persists the version as one to suppress notifications for.
func (s *Service) SkipVersion(version string) error {
	return s.prefs.SetAutoUpdateSkipVersion(version)
}

// Unskip clears the persisted skip-version.
func (s *Service) Unskip() error {
	return s.prefs.SetAutoUpdateSkipVersion("")
}

// PeekResult is a one-shot read of both providers. Used by the Settings tab
// to render side-by-side latest-version status without changing active source.
type PeekResult struct {
	GitHub *Release          `json:"github,omitempty"`
	GitLab *Release          `json:"gitlab,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
}

// PeekBothSources queries both providers in parallel without modifying
// active source state or lastCheckedAt.
func (s *Service) PeekBothSources(ctx context.Context) PeekResult {
	res := PeekResult{Errors: make(map[string]string)}
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for name, p := range s.providers {
		wg.Add(1)
		go func(name string, p Provider) {
			defer wg.Done()
			rel, err := p.LatestRelease(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.Errors[name] = err.Error()
				return
			}
			cp := rel
			switch name {
			case preferences.AutoUpdateSourceGitHub:
				res.GitHub = &cp
			case preferences.AutoUpdateSourceGitLab:
				res.GitLab = &cp
			}
		}(name, p)
	}
	wg.Wait()
	return res
}

// State is the snapshot the frontend reads when rendering the Updates tab.
type State struct {
	Enabled         bool     `json:"enabled"`
	Source          string   `json:"source"`
	IntervalHours   int      `json:"intervalHours"`
	RunningVersion  string   `json:"runningVersion"`
	LastCheckedAt   string   `json:"lastCheckedAt"`
	LatestAvailable *Release `json:"latestAvailable,omitempty"`
	SkipVersion     string   `json:"skipVersion"`
	InProgress      bool     `json:"inProgress"`
	ManualRequired  bool     `json:"manualRequired"`
	// BuildKind is "portable" or "installer". The frontend uses it to gate
	// install-pathway-specific copy (e.g. the UAC-prompt hint).
	BuildKind string `json:"buildKind"`
}

// GetState returns the current service snapshot for UI consumption.
func (s *Service) GetState() State {
	au, _ := s.prefs.GetAutoUpdate()
	s.mu.Lock()
	last := s.lastResult
	s.mu.Unlock()
	var lastCopy *Release
	if last != nil {
		cp := *last
		lastCopy = &cp
	}
	return State{
		Enabled:         au.Enabled,
		Source:          au.Source,
		IntervalHours:   au.IntervalHours,
		RunningVersion:  s.runningVersion,
		LastCheckedAt:   au.LastCheckedAt,
		LatestAvailable: lastCopy,
		SkipVersion:     au.SkipVersion,
		InProgress:      s.inProgress.Load(),
		// ManualRequired only applies to portable builds. An installer build
		// in Program Files is expected to fail the writability probe; that
		// is normal, not a manual-required signal.
		ManualRequired: IsPortable() && !IsWritable(filepath.Dir(s.exePath)),
		BuildKind:      BuildKind,
	}
}

// InstallAvailable installs the most recent available release. Returns an
// error if no release is currently cached as available.
func (s *Service) InstallAvailable(ctx context.Context) error {
	s.mu.Lock()
	rel := s.lastResult
	s.mu.Unlock()
	if rel == nil {
		return errors.New("no release available to install")
	}
	return s.install(ctx, *rel)
}

func (s *Service) onPrefsChange(_ preferences.Preferences) {
	select {
	case s.prefsChangedCh <- struct{}{}:
	default:
	}
}

func (s *Service) runLoop(ctx context.Context) {
	defer close(s.loopDone)
	s.maybeFireStartupCheck(ctx)
	timer := time.NewTimer(s.nextInterval())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			au, _ := s.prefs.GetAutoUpdate()
			if au.Enabled {
				s.runCheck(ctx, "ticker")
			}
			timer.Reset(s.nextInterval())
		case <-s.manualCheckCh:
			s.runCheck(ctx, "manual")
			drainAndReset(timer, s.nextInterval())
		case <-s.prefsChangedCh:
			au, _ := s.prefs.GetAutoUpdate()
			sourceChanged := au.Source != s.snapshot.source
			s.snapshot = autoUpdateSnapshot{
				source:        au.Source,
				intervalHours: au.IntervalHours,
				enabled:       au.Enabled,
			}
			drainAndReset(timer, s.nextInterval())
			if sourceChanged && au.Enabled {
				s.runCheck(ctx, "source-change")
			}
		}
	}
}

func drainAndReset(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

func (s *Service) maybeFireStartupCheck(ctx context.Context) {
	au, _ := s.prefs.GetAutoUpdate()
	if !au.Enabled {
		return
	}
	if au.LastCheckedAt == "" {
		s.runCheck(ctx, "startup")
		return
	}
	last, err := time.Parse(time.RFC3339, au.LastCheckedAt)
	if err != nil {
		s.runCheck(ctx, "startup")
		return
	}
	if s.now().Sub(last) >= time.Duration(au.IntervalHours)*time.Hour {
		s.runCheck(ctx, "startup")
	}
}

func (s *Service) nextInterval() time.Duration {
	au, _ := s.prefs.GetAutoUpdate()
	if au.IntervalHours <= 0 {
		return defaultIntervalFallback
	}
	return time.Duration(au.IntervalHours) * time.Hour
}

// RunCheck is exposed for tests; production code reaches it via the loop.
func (s *Service) RunCheck(ctx context.Context, trigger string) {
	s.runCheck(ctx, trigger)
}

func (s *Service) runCheck(ctx context.Context, trigger string) {
	au, _ := s.prefs.GetAutoUpdate()
	s.emit("updater:check-started", map[string]any{
		"source":  au.Source,
		"trigger": trigger,
	})

	p, ok := s.providers[au.Source]
	if !ok {
		s.emit("updater:error", map[string]any{
			"stage":   "check",
			"source":  au.Source,
			"message": fmt.Sprintf("unknown source %q", au.Source),
		})
		return
	}

	rel, err := p.LatestRelease(ctx)
	_ = s.prefs.SetAutoUpdateLastCheckedAt(s.now())
	if err != nil {
		s.emit("updater:error", map[string]any{
			"stage":   "check",
			"source":  au.Source,
			"message": err.Error(),
		})
		return
	}

	cmp := Compare(rel.Version, s.runningVersion)
	if cmp <= 0 {
		s.mu.Lock()
		s.lastResult = nil
		s.mu.Unlock()
		s.emit("updater:up-to-date", map[string]any{
			"source":         au.Source,
			"latestVersion":  rel.Version,
			"runningVersion": s.runningVersion,
		})
		return
	}

	s.mu.Lock()
	cp := rel
	s.lastResult = &cp
	s.mu.Unlock()

	if rel.Version == au.SkipVersion {
		// Cache the result so PeekBothSources / GetState can surface
		// the "skipped" indicator, but suppress the available toast.
		return
	}

	s.emit("updater:available", rel)
}

func (s *Service) install(ctx context.Context, rel Release) error {
	if s.inProgress.Load() {
		return errors.New("update already in progress")
	}
	if IsInstaller() {
		return s.installInstaller(ctx, rel)
	}

	// Portable pathway requires the portable binary asset. Releases since
	// the installer-only switch may omit it; surface that explicitly so the
	// user knows they need to manually download the installer once to move
	// off the portable build.
	if rel.AssetURL == "" || rel.AssetName == "" {
		err := fmt.Errorf("%w: portable asset not in release %s; this release ships installer-only — download the latest installer manually to migrate", ErrAssetNotFound, rel.Version)
		s.emit("updater:error", map[string]any{
			"stage":   "download",
			"message": err.Error(),
		})
		return err
	}

	newPath := s.exePath + ".new"
	progressFn := func(d, total int64) {
		s.emit("updater:download-progress", map[string]any{
			"bytesDownloaded": d,
			"totalBytes":      total,
		})
	}

	type binResult struct {
		sha string
		err error
	}
	type csResult struct {
		body []byte
		err  error
	}
	binCh := make(chan binResult, 1)
	csCh := make(chan csResult, 1)

	dlCtx, cancelDL := context.WithCancel(ctx)
	defer cancelDL()

	go func() {
		sha, err := Download(dlCtx, s.httpClient, rel.AssetURL, newPath, rel.AssetSize, progressFn)
		binCh <- binResult{sha: sha, err: err}
	}()
	go func() {
		body, err := downloadBytes(dlCtx, s.httpClient, rel.ChecksumsURL)
		csCh <- csResult{body: body, err: err}
	}()

	bin := <-binCh
	cs := <-csCh

	if bin.err != nil {
		_ = os.Remove(newPath)
		_ = os.Remove(newPath + ".partial")
		s.emit("updater:error", map[string]any{
			"stage":   "download",
			"message": bin.err.Error(),
		})
		return bin.err
	}
	if cs.err != nil {
		_ = os.Remove(newPath)
		s.emit("updater:error", map[string]any{
			"stage":   "download",
			"message": "checksums: " + cs.err.Error(),
		})
		return cs.err
	}

	checksums, err := ParseChecksums(bytes.NewReader(cs.body))
	if err != nil {
		_ = os.Remove(newPath)
		s.emit("updater:error", map[string]any{
			"stage":   "verify",
			"message": "checksums file unavailable: " + err.Error(),
		})
		return err
	}
	expected, ok := checksums[rel.AssetName]
	if !ok {
		_ = os.Remove(newPath)
		err := fmt.Errorf("asset %q not listed in checksums", rel.AssetName)
		s.emit("updater:error", map[string]any{
			"stage":   "verify",
			"message": err.Error(),
		})
		return err
	}
	if expected != bin.sha {
		_ = os.Remove(newPath)
		err := fmt.Errorf("checksum mismatch for %s: expected %s, got %s", rel.AssetName, expected, bin.sha)
		s.emit("updater:error", map[string]any{
			"stage":   "verify",
			"message": err.Error(),
		})
		return err
	}

	s.inProgress.Store(true)

	if err := s.swap(s.exePath, newPath); err != nil {
		s.inProgress.Store(false)
		_ = os.Remove(newPath)
		s.emit("updater:error", map[string]any{
			"stage":   "rename",
			"message": err.Error(),
		})
		return err
	}

	if err := s.spawn(s.exePath); err != nil {
		s.emit("updater:error", map[string]any{
			"stage":   "spawn",
			"message": err.Error(),
		})
		return err
	}

	s.emit("updater:installed", map[string]any{"version": rel.Version})
	s.quitApp()
	return nil
}

// installInstaller is the installer-build pathway. It downloads the NSIS
// installer asset, verifies its SHA-256 against checksums.txt, then spawns
// it under a UAC-elevated `runas` shell verb with the silent install flag.
// The order of operations is deliberate (see openspec change
// enable-installer-build-updater design Decision 3 / 3b):
//
//  1. download + verify  (any failure: emit error, no quit)
//  2. inProgress = true  (so the close intercept short-circuits while UAC shows)
//  3. installerSpawn     (blocks until UAC accepted/declined)
//  4. on decline/error: clear inProgress, emit error, return
//  5. emit installed
//  6. quitApp            (releases file lock so the elevated installer can
//     overwrite RedShell.exe; NSIS Sleep at the top of the install Section
//     absorbs the brief race)
func (s *Service) installInstaller(ctx context.Context, rel Release) error {
	if rel.InstallerAssetURL == "" || rel.InstallerAssetName == "" {
		err := fmt.Errorf("%w (release %s)", ErrInstallerNotFound, rel.Version)
		s.emit("updater:error", map[string]any{
			"stage":   "installer-download",
			"message": err.Error(),
		})
		return err
	}

	// Download to %TEMP% rather than next to the exe. For installer builds
	// the install directory is Program Files (or similar admin-only path)
	// — the running app at medium integrity cannot write there. UAC
	// elevation happens AFTER download, so the destination must be a
	// user-writable scratch location. The elevated installer child runs as
	// the same user and can still read the file.
	installerPath := installerDownloadPath()
	progressFn := func(d, total int64) {
		s.emit("updater:download-progress", map[string]any{
			"bytesDownloaded": d,
			"totalBytes":      total,
		})
	}

	type binResult struct {
		sha string
		err error
	}
	type csResult struct {
		body []byte
		err  error
	}
	binCh := make(chan binResult, 1)
	csCh := make(chan csResult, 1)

	dlCtx, cancelDL := context.WithCancel(ctx)
	defer cancelDL()

	go func() {
		sha, err := Download(dlCtx, s.httpClient, rel.InstallerAssetURL, installerPath, rel.InstallerAssetSize, progressFn)
		binCh <- binResult{sha: sha, err: err}
	}()
	go func() {
		body, err := downloadBytes(dlCtx, s.httpClient, rel.ChecksumsURL)
		csCh <- csResult{body: body, err: err}
	}()

	bin := <-binCh
	cs := <-csCh

	if bin.err != nil {
		_ = os.Remove(installerPath)
		_ = os.Remove(installerPath + ".partial")
		s.emit("updater:error", map[string]any{
			"stage":   "installer-download",
			"message": bin.err.Error(),
		})
		return bin.err
	}
	if cs.err != nil {
		_ = os.Remove(installerPath)
		s.emit("updater:error", map[string]any{
			"stage":   "installer-download",
			"message": "checksums: " + cs.err.Error(),
		})
		return cs.err
	}

	checksums, err := ParseChecksums(bytes.NewReader(cs.body))
	if err != nil {
		_ = os.Remove(installerPath)
		s.emit("updater:error", map[string]any{
			"stage":   "verify",
			"message": "checksums file unavailable: " + err.Error(),
		})
		return err
	}
	expected, ok := checksums[rel.InstallerAssetName]
	if !ok {
		_ = os.Remove(installerPath)
		err := fmt.Errorf("installer asset %q not listed in checksums", rel.InstallerAssetName)
		s.emit("updater:error", map[string]any{
			"stage":   "verify",
			"message": err.Error(),
		})
		return err
	}
	if expected != bin.sha {
		_ = os.Remove(installerPath)
		err := fmt.Errorf("checksum mismatch for %s: expected %s, got %s", rel.InstallerAssetName, expected, bin.sha)
		s.emit("updater:error", map[string]any{
			"stage":   "verify",
			"message": err.Error(),
		})
		return err
	}

	// Set inProgress BEFORE spawn — the UAC dialog blocks on the same
	// goroutine and we want the close intercept to short-circuit during
	// that window in case the user closes the app while the prompt is up.
	s.inProgress.Store(true)
	if err := s.installerSpawn(installerPath, []string{"/S"}); err != nil {
		s.inProgress.Store(false)
		message := err.Error()
		if errors.Is(err, ErrUACDeclined) {
			message = "user cancelled elevation"
		}
		s.emit("updater:error", map[string]any{
			"stage":   "installer-spawn",
			"message": message,
		})
		return err
	}

	s.emit("updater:installed", map[string]any{"version": rel.Version})
	s.quitApp()
	return nil
}

func downloadBytes(ctx context.Context, httpClient *http.Client, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, checksumsMaxBytes))
}
