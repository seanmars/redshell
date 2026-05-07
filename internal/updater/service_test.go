package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"redshell/internal/preferences"
)

type fakeProvider struct {
	name    string
	release Release
	err     error
	calls   int
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) LatestRelease(_ context.Context) (Release, error) {
	f.calls++
	if f.err != nil {
		return Release{}, f.err
	}
	return f.release, nil
}

type capturedEvent struct {
	name string
	data any
}

type eventRecorder struct {
	mu     sync.Mutex
	events []capturedEvent
}

func (r *eventRecorder) emit(name string, data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, capturedEvent{name: name, data: data})
}

func (r *eventRecorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.events))
	for _, e := range r.events {
		out = append(out, e.name)
	}
	return out
}

func (r *eventRecorder) findFirst(name string) (any, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if e.name == name {
			return e.data, true
		}
	}
	return nil, false
}

func newTestPrefs(t *testing.T) *preferences.Service {
	t.Helper()
	root := t.TempDir()
	return preferences.NewServiceWithPath(filepath.Join(root, ".redshell", "preferences.json"))
}

func newTestService(t *testing.T, providers map[string]Provider, runningVersion string) (*Service, *eventRecorder, *preferences.Service) {
	t.Helper()
	prefs := newTestPrefs(t)
	rec := &eventRecorder{}
	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "redshell.exe")
	if err := os.WriteFile(exePath, []byte("running-bytes"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	svc, err := NewServiceWithProviders(prefs, runningVersion, exePath, providers, Options{
		Now:   func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) },
		Swap:  func(currentPath, newPath string) error { return os.Rename(newPath, currentPath) },
		Spawn: func(string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	svc.emit = rec.emit
	svc.quitApp = func() {}
	return svc, rec, prefs
}

func TestService_RunCheckEmitsAvailableForNewerRelease(t *testing.T) {
	rel := Release{
		Version:      "v0.5.0",
		AssetURL:     "https://example.com/r.exe",
		AssetName:    "redshell-windows-amd64.exe",
		AssetSize:    100,
		ChecksumsURL: "https://example.com/checksums.txt",
	}
	svc, rec, _ := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, "v0.4.0")

	svc.RunCheck(context.Background(), "manual")
	got := rec.names()
	if !contains(got, "updater:check-started") {
		t.Fatalf("expected check-started, events=%v", got)
	}
	if !contains(got, "updater:available") {
		t.Fatalf("expected available, events=%v", got)
	}
}

func TestService_RunCheckEmitsUpToDateWhenSameOrOlder(t *testing.T) {
	rel := Release{Version: "v0.4.0", AssetName: "redshell-windows-amd64.exe"}
	svc, rec, _ := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, "v0.4.0")

	svc.RunCheck(context.Background(), "manual")
	if !contains(rec.names(), "updater:up-to-date") {
		t.Fatalf("expected up-to-date, got %v", rec.names())
	}
	if contains(rec.names(), "updater:available") {
		t.Fatalf("should not emit available when running version is current, got %v", rec.names())
	}
}

func TestService_RunCheckSuppressesAvailableForSkipVersion(t *testing.T) {
	rel := Release{Version: "v0.5.0", AssetName: "redshell-windows-amd64.exe"}
	svc, rec, prefs := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, "v0.4.0")
	if err := prefs.SetAutoUpdateSkipVersion("v0.5.0"); err != nil {
		t.Fatalf("SetAutoUpdateSkipVersion: %v", err)
	}

	svc.RunCheck(context.Background(), "manual")
	if contains(rec.names(), "updater:available") {
		t.Fatalf("skipped version must not emit available, events=%v", rec.names())
	}
	state := svc.GetState()
	if state.LatestAvailable == nil || state.LatestAvailable.Version != "v0.5.0" {
		t.Fatal("skipped version should still be cached as latestAvailable for the UI")
	}
}

func TestService_RunCheckEmitsErrorOnProviderFailure(t *testing.T) {
	svc, rec, _ := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", err: errors.New("boom")},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", err: errors.New("boom")},
	}, "v0.4.0")

	svc.RunCheck(context.Background(), "manual")
	if !contains(rec.names(), "updater:error") {
		t.Fatalf("expected error event, got %v", rec.names())
	}
}

func TestService_PeekBothSourcesQueriesBothInParallel(t *testing.T) {
	gh := &fakeProvider{name: "github", release: Release{Version: "v0.5.0", AssetName: "redshell-windows-amd64.exe"}}
	gl := &fakeProvider{name: "gitlab", release: Release{Version: "v0.4.8", AssetName: "redshell-windows-amd64.exe"}}
	svc, _, _ := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: gh,
		preferences.AutoUpdateSourceGitLab: gl,
	}, "v0.4.0")

	res := svc.PeekBothSources(context.Background())
	if res.GitHub == nil || res.GitHub.Version != "v0.5.0" {
		t.Fatalf("github peek: got %#v", res.GitHub)
	}
	if res.GitLab == nil || res.GitLab.Version != "v0.4.8" {
		t.Fatalf("gitlab peek: got %#v", res.GitLab)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors, got %#v", res.Errors)
	}
}

func TestService_PeekBothSourcesSurfacesPerSourceErrors(t *testing.T) {
	gh := &fakeProvider{name: "github", release: Release{Version: "v0.5.0", AssetName: "redshell-windows-amd64.exe"}}
	gl := &fakeProvider{name: "gitlab", err: errors.New("network is unreachable")}
	svc, _, _ := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: gh,
		preferences.AutoUpdateSourceGitLab: gl,
	}, "v0.4.0")

	res := svc.PeekBothSources(context.Background())
	if res.GitHub == nil {
		t.Fatal("github peek should succeed")
	}
	if res.GitLab != nil {
		t.Fatal("gitlab peek should be nil on failure")
	}
	if msg := res.Errors[preferences.AutoUpdateSourceGitLab]; msg == "" {
		t.Fatal("expected gitlab error to be recorded")
	}
}

func TestService_InstallAvailableHappyPath(t *testing.T) {
	body := []byte("fake new binary payload")
	hashSum := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hashSum[:])
	checksumsBody := hashHex + "  redshell-windows-amd64.exe\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/r.exe", func(w http.ResponseWriter, r *http.Request) { w.Write(body) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksumsBody))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := Release{
		Version:      "v0.5.0",
		AssetURL:     srv.URL + "/r.exe",
		AssetName:    "redshell-windows-amd64.exe",
		AssetSize:    int64(len(body)),
		ChecksumsURL: srv.URL + "/checksums.txt",
	}

	prefs := newTestPrefs(t)
	rec := &eventRecorder{}
	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "redshell.exe")
	if err := os.WriteFile(exePath, []byte("running-v0.4.0"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}

	var swapCalled, spawnCalled, quitCalled bool
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, Options{
		HTTPClient: srv.Client(),
		Swap: func(currentPath, newPath string) error {
			swapCalled = true
			return os.Rename(newPath, currentPath)
		},
		Spawn: func(string) error { spawnCalled = true; return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	svc.emit = rec.emit
	svc.quitApp = func() { quitCalled = true }

	svc.RunCheck(context.Background(), "manual")
	if !contains(rec.names(), "updater:available") {
		t.Fatalf("setup: expected available event, got %v", rec.names())
	}

	if err := svc.InstallAvailable(context.Background()); err != nil {
		t.Fatalf("InstallAvailable: %v", err)
	}
	if !swapCalled {
		t.Fatal("swap should be called")
	}
	if !spawnCalled {
		t.Fatal("spawn should be called")
	}
	if !quitCalled {
		t.Fatal("quit should be called")
	}
	if !svc.InProgress() {
		t.Fatal("InProgress should remain true after install for close-intercept short-circuit")
	}
	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("read exe after install: %v", err)
	}
	if string(got) != string(body) {
		t.Fatal("running exe should have been replaced by the downloaded payload")
	}
	if !contains(rec.names(), "updater:installed") {
		t.Fatalf("expected installed event, got %v", rec.names())
	}
}

func TestService_InstallAvailableRejectsChecksumMismatch(t *testing.T) {
	body := []byte("fake new binary payload")
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
	checksumsBody := wrongHash + "  redshell-windows-amd64.exe\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/r.exe", func(w http.ResponseWriter, r *http.Request) { w.Write(body) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksumsBody))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := Release{
		Version:      "v0.5.0",
		AssetURL:     srv.URL + "/r.exe",
		AssetName:    "redshell-windows-amd64.exe",
		AssetSize:    int64(len(body)),
		ChecksumsURL: srv.URL + "/checksums.txt",
	}

	prefs := newTestPrefs(t)
	rec := &eventRecorder{}
	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "redshell.exe")
	if err := os.WriteFile(exePath, []byte("running"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, Options{
		HTTPClient: srv.Client(),
		Swap:       func(string, string) error { t.Fatal("swap must not be called on checksum mismatch"); return nil },
		Spawn:      func(string) error { t.Fatal("spawn must not be called on checksum mismatch"); return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	svc.emit = rec.emit
	svc.quitApp = func() { t.Fatal("quit must not be called on checksum mismatch") }

	svc.RunCheck(context.Background(), "manual")
	err = svc.InstallAvailable(context.Background())
	if err == nil {
		t.Fatal("expected checksum mismatch to error")
	}
	if !contains(rec.names(), "updater:error") {
		t.Fatalf("expected error event, got %v", rec.names())
	}
	if _, statErr := os.Stat(exePath + ".new"); !os.IsNotExist(statErr) {
		t.Fatalf("rejected .new should be removed: %v", statErr)
	}
	if svc.InProgress() {
		t.Fatal("InProgress must remain false on rejected install")
	}
}

func TestService_InstallAvailableRejectsMissingChecksumsEntry(t *testing.T) {
	body := []byte("payload")
	mux := http.NewServeMux()
	mux.HandleFunc("/r.exe", func(w http.ResponseWriter, r *http.Request) { w.Write(body) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  some-other-file.exe\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := Release{
		Version: "v0.5.0", AssetURL: srv.URL + "/r.exe", AssetName: "redshell-windows-amd64.exe",
		AssetSize: int64(len(body)), ChecksumsURL: srv.URL + "/checksums.txt",
	}
	prefs := newTestPrefs(t)
	rec := &eventRecorder{}
	exePath := filepath.Join(t.TempDir(), "redshell.exe")
	if err := os.WriteFile(exePath, []byte("running"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, Options{HTTPClient: srv.Client(), Swap: func(string, string) error { return nil }, Spawn: func(string) error { return nil }})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	svc.emit = rec.emit
	svc.quitApp = func() {}
	svc.RunCheck(context.Background(), "manual")
	if err := svc.InstallAvailable(context.Background()); err == nil {
		t.Fatal("expected missing-asset checksum to error")
	}
}

func TestService_InstallAvailableErrorsWhenNoCachedRelease(t *testing.T) {
	svc, _, _ := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github"},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab"},
	}, "v0.4.0")
	if err := svc.InstallAvailable(context.Background()); err == nil {
		t.Fatal("expected error when no release cached")
	}
}

func TestService_StartEmitsManualRequiredOnNonWritableDir(t *testing.T) {
	prefs := newTestPrefs(t)
	rec := &eventRecorder{}
	exePath := filepath.Join(t.TempDir(), "non-existent-subdir", "redshell.exe")
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github"},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab"},
	}, Options{
		Swap: func(string, string) error { return nil }, Spawn: func(string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}

	if err := svc.Start(context.Background(), rec.emit, func() {}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !contains(rec.names(), "updater:manual-required") {
		t.Fatalf("expected manual-required event, got %v", rec.names())
	}
	if svc.cancelLoop != nil {
		t.Fatal("ticker goroutine must not start when install dir is not writable")
	}
}

func TestService_GetStateReflectsPrefsAndCache(t *testing.T) {
	rel := Release{Version: "v0.5.0", AssetName: "redshell-windows-amd64.exe"}
	svc, _, prefs := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, "v0.4.0")
	if err := prefs.SetAutoUpdateSource(preferences.AutoUpdateSourceGitLab); err != nil {
		t.Fatalf("SetAutoUpdateSource: %v", err)
	}
	svc.RunCheck(context.Background(), "manual")

	state := svc.GetState()
	if state.Source != preferences.AutoUpdateSourceGitLab {
		t.Fatalf("Source: got %q want gitlab", state.Source)
	}
	if state.RunningVersion != "v0.4.0" {
		t.Fatalf("RunningVersion: got %q", state.RunningVersion)
	}
	if state.LatestAvailable == nil || state.LatestAvailable.Version != "v0.5.0" {
		t.Fatal("expected latestAvailable cached")
	}
}

func TestService_NextIntervalDefaultsTo6Hours(t *testing.T) {
	svc, _, _ := newTestService(t, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github"},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab"},
	}, "v0.4.0")
	got := svc.nextInterval()
	if got != 6*time.Hour {
		t.Fatalf("default interval: got %v want 6h", got)
	}
}

func TestService_MaybeFireStartupCheckHonorsLastCheckedAt(t *testing.T) {
	rel := Release{Version: "v0.5.0", AssetName: "redshell-windows-amd64.exe"}
	prefs := newTestPrefs(t)
	provider := &fakeProvider{name: "github", release: rel}
	rec := &eventRecorder{}
	exePath := filepath.Join(t.TempDir(), "redshell.exe")
	if err := os.WriteFile(exePath, []byte("running"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: provider,
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab"},
	}, Options{Now: func() time.Time { return now }, Swap: func(string, string) error { return nil }, Spawn: func(string) error { return nil }})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	svc.emit = rec.emit
	svc.quitApp = func() {}

	// 1 hour ago + 6h interval = should NOT fire
	if err := prefs.SetAutoUpdateLastCheckedAt(now.Add(-1 * time.Hour)); err != nil {
		t.Fatalf("SetAutoUpdateLastCheckedAt: %v", err)
	}
	svc.maybeFireStartupCheck(context.Background())
	if provider.calls != 0 {
		t.Fatalf("recent lastCheckedAt should suppress startup check, calls=%d", provider.calls)
	}

	// 7 hours ago + 6h interval = should fire
	if err := prefs.SetAutoUpdateLastCheckedAt(now.Add(-7 * time.Hour)); err != nil {
		t.Fatalf("SetAutoUpdateLastCheckedAt: %v", err)
	}
	svc.maybeFireStartupCheck(context.Background())
	if provider.calls != 1 {
		t.Fatalf("stale lastCheckedAt should fire startup check, calls=%d", provider.calls)
	}

	// disabled = should not fire even when stale
	if err := prefs.SetAutoUpdateEnabled(false); err != nil {
		t.Fatalf("SetAutoUpdateEnabled: %v", err)
	}
	if err := prefs.SetAutoUpdateLastCheckedAt(now.Add(-100 * time.Hour)); err != nil {
		t.Fatalf("SetAutoUpdateLastCheckedAt: %v", err)
	}
	svc.maybeFireStartupCheck(context.Background())
	if provider.calls != 1 {
		t.Fatalf("disabled should suppress startup check, calls=%d", provider.calls)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
