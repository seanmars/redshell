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

	"redshell/internal/preferences"
)

// installerCallRecord captures a SpawnInstaller call so the test can assert
// the path/args we passed match what we expected.
type installerCallRecord struct {
	mu    sync.Mutex
	calls []struct {
		path string
		args []string
	}
	err error
}

func (r *installerCallRecord) spawn(path string, args []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, struct {
		path string
		args []string
	}{path: path, args: append([]string(nil), args...)})
	return r.err
}

func (r *installerCallRecord) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func newInstallerTestService(t *testing.T, rec *installerCallRecord, rel Release, srvClient *http.Client) (*Service, *eventRecorder, string, *bool) {
	t.Helper()
	prefs := newTestPrefs(t)
	er := &eventRecorder{}
	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "redshell.exe")
	if err := os.WriteFile(exePath, []byte("running-bytes"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	// Installer pathway downloads to a process-wide path in os.TempDir().
	// Every test must scrub that file (and its .partial sibling) on exit
	// so leftovers don't bleed across tests or across `go test` runs.
	// Also scrub the legacy .new name in case a previous broken build of
	// the test suite (or product) left one behind.
	t.Cleanup(func() {
		path := installerDownloadPath()
		dir := filepath.Dir(path)
		_ = os.Remove(path)
		_ = os.Remove(path + ".partial")
		_ = os.Remove(filepath.Join(dir, "redshell-installer.new"))
		_ = os.Remove(filepath.Join(dir, "redshell-installer.new.partial"))
	})
	quitCalled := false
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github", release: rel},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab", release: rel},
	}, Options{
		HTTPClient: srvClient,
		Swap:       func(string, string) error { t.Fatal("swap must not be called on installer pathway"); return nil },
		Spawn: func(string, []string) error {
			t.Fatal("portable spawn must not be called on installer pathway")
			return nil
		},
		InstallerSpawn: rec.spawn,
	})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	svc.emit = er.emit
	svc.quitApp = func() { quitCalled = true }
	return svc, er, exePath, &quitCalled
}

// installerHTTPServer stands up an httptest server that serves a fake
// installer binary plus a checksums file with the matching SHA-256 entry.
func installerHTTPServer(t *testing.T, body []byte, checksumName string) (string, string, *http.Client, func()) {
	t.Helper()
	hashSum := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hashSum[:])
	checksumsBody := hashHex + "  " + checksumName + "\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/installer.exe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	})
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksumsBody))
	})
	srv := httptest.NewServer(mux)
	return srv.URL + "/installer.exe", srv.URL + "/checksums.txt", srv.Client(), srv.Close
}

func TestService_StartSkipsManualRequiredForInstallerBuild(t *testing.T) {
	withBuildKind(t, "installer")

	prefs := newTestPrefs(t)
	rec := &eventRecorder{}
	// Use a non-writable directory (one that doesn't exist) — for portable
	// builds this would fire updater:manual-required. For installer builds
	// the probe must be skipped.
	exePath := filepath.Join(t.TempDir(), "non-existent-subdir", "redshell.exe")
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github"},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab"},
	}, Options{
		Swap:           func(string, string) error { return nil },
		Spawn:          func(string, []string) error { return nil },
		InstallerSpawn: func(string, []string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	if err := svc.Start(context.Background(), rec.emit, func() {}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if contains(rec.names(), "updater:manual-required") {
		t.Fatalf("installer build must not emit manual-required even when dir is non-writable, events=%v", rec.names())
	}
	if svc.cancelLoop == nil {
		t.Fatal("installer build should still register the run loop")
	}
	// Tear down the loop so the test exits cleanly.
	svc.Stop()
}

func TestService_GetStateReportsBuildKindAndScopesManualRequired(t *testing.T) {
	withBuildKind(t, "installer")

	prefs := newTestPrefs(t)
	exePath := filepath.Join(t.TempDir(), "missing-dir", "redshell.exe")
	svc, err := NewServiceWithProviders(prefs, "v0.4.0", exePath, map[string]Provider{
		preferences.AutoUpdateSourceGitHub: &fakeProvider{name: "github"},
		preferences.AutoUpdateSourceGitLab: &fakeProvider{name: "gitlab"},
	}, Options{Swap: func(string, string) error { return nil }, Spawn: func(string, []string) error { return nil }, InstallerSpawn: func(string, []string) error { return nil }})
	if err != nil {
		t.Fatalf("NewServiceWithProviders: %v", err)
	}
	st := svc.GetState()
	if st.BuildKind != "installer" {
		t.Fatalf("BuildKind: got %q want installer", st.BuildKind)
	}
	if st.ManualRequired {
		t.Fatal("ManualRequired must be false for installer builds even in non-writable dirs")
	}
}

func TestService_InstallAvailable_InstallerHappyPath(t *testing.T) {
	withBuildKind(t, "installer")

	body := []byte("fake installer payload")
	installerURL, checksumsURL, client, closeSrv := installerHTTPServer(t, body, "RedShell-amd64-installer.exe")
	defer closeSrv()

	rel := Release{
		Version:            "v0.5.0",
		AssetURL:           "unused-portable-url",
		AssetName:          "redshell-windows-amd64.exe",
		ChecksumsURL:       checksumsURL,
		InstallerAssetURL:  installerURL,
		InstallerAssetName: "RedShell-amd64-installer.exe",
		InstallerAssetSize: int64(len(body)),
	}

	rec := &installerCallRecord{}
	svc, er, _, quitCalled := newInstallerTestService(t, rec, rel, client)

	svc.RunCheck(context.Background(), "manual")
	if err := svc.InstallAvailable(context.Background()); err != nil {
		t.Fatalf("InstallAvailable: %v", err)
	}

	if rec.callCount() != 1 {
		t.Fatalf("installerSpawn calls: got %d want 1", rec.callCount())
	}
	rec.mu.Lock()
	got := rec.calls[0]
	rec.mu.Unlock()
	// Installer is downloaded to %TEMP% (NOT next to the exe) so the
	// non-admin running process can write to it before UAC elevation.
	wantPath := installerDownloadPath()
	if got.path != wantPath {
		t.Fatalf("installerSpawn path: got %q want %q", got.path, wantPath)
	}
	if len(got.args) != 1 || got.args[0] != "/S" {
		t.Fatalf("installerSpawn args: got %v want [/S]", got.args)
	}
	if !*quitCalled {
		t.Fatal("quitApp must be called after successful installer spawn")
	}
	if !svc.InProgress() {
		t.Fatal("InProgress must remain true after spawn for close-intercept short-circuit")
	}
	if !contains(er.names(), "updater:installed") {
		t.Fatalf("expected installed event, got %v", er.names())
	}
	// Sanity: portable swap path was wired to t.Fatal, so we know it wasn't called.
}

func TestService_InstallAvailable_InstallerMissingAsset(t *testing.T) {
	withBuildKind(t, "installer")

	rel := Release{
		Version:      "v0.5.0",
		AssetName:    "redshell-windows-amd64.exe",
		ChecksumsURL: "https://example.invalid/checksums.txt",
		// No InstallerAssetURL / Name - simulating a release that didn't ship an installer.
	}

	rec := &installerCallRecord{}
	svc, er, _, quitCalled := newInstallerTestService(t, rec, rel, http.DefaultClient)

	svc.RunCheck(context.Background(), "manual")
	if err := svc.InstallAvailable(context.Background()); err == nil {
		t.Fatal("expected error for missing installer asset")
	} else if !errors.Is(err, ErrInstallerNotFound) {
		t.Fatalf("expected ErrInstallerNotFound, got %v", err)
	}
	if rec.callCount() != 0 {
		t.Fatal("installerSpawn must not be called when the asset is missing")
	}
	if *quitCalled {
		t.Fatal("quitApp must not be called when install errors out before spawn")
	}
	// Confirm the error event has the right stage
	data, ok := er.findFirst("updater:error")
	if !ok {
		t.Fatalf("expected updater:error event, got %v", er.names())
	}
	m, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("error payload not map: %#v", data)
	}
	if m["stage"] != "installer-download" {
		t.Fatalf("error stage: got %v want installer-download", m["stage"])
	}
}

func TestService_InstallAvailable_InstallerUACDeclined(t *testing.T) {
	withBuildKind(t, "installer")

	body := []byte("payload")
	installerURL, checksumsURL, client, closeSrv := installerHTTPServer(t, body, "RedShell-amd64-installer.exe")
	defer closeSrv()

	rel := Release{
		Version:            "v0.5.0",
		AssetName:          "redshell-windows-amd64.exe",
		ChecksumsURL:       checksumsURL,
		InstallerAssetURL:  installerURL,
		InstallerAssetName: "RedShell-amd64-installer.exe",
		InstallerAssetSize: int64(len(body)),
	}

	rec := &installerCallRecord{err: ErrUACDeclined}
	svc, er, _, quitCalled := newInstallerTestService(t, rec, rel, client)

	svc.RunCheck(context.Background(), "manual")
	err := svc.InstallAvailable(context.Background())
	if err == nil {
		t.Fatal("expected UAC-declined to surface as error")
	}
	if !errors.Is(err, ErrUACDeclined) {
		t.Fatalf("expected ErrUACDeclined, got %v", err)
	}
	if svc.InProgress() {
		t.Fatal("inProgress must be cleared when UAC is declined so close-intercept resumes normal behavior")
	}
	if *quitCalled {
		t.Fatal("quitApp must not be called when UAC is declined")
	}
	data, ok := er.findFirst("updater:error")
	if !ok {
		t.Fatalf("expected updater:error event, got %v", er.names())
	}
	m, _ := data.(map[string]any)
	if m["stage"] != "installer-spawn" {
		t.Fatalf("error stage: got %v want installer-spawn", m["stage"])
	}
	if m["message"] != "user cancelled elevation" {
		t.Fatalf("error message: got %v want user cancelled elevation", m["message"])
	}
}

func TestService_InstallAvailable_InstallerChecksumMismatch(t *testing.T) {
	withBuildKind(t, "installer")

	body := []byte("real payload")
	// Serve a checksums file with the WRONG hash for the installer name.
	mux := http.NewServeMux()
	mux.HandleFunc("/installer.exe", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0000000000000000000000000000000000000000000000000000000000000000  RedShell-amd64-installer.exe\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := Release{
		Version:            "v0.5.0",
		AssetName:          "redshell-windows-amd64.exe",
		ChecksumsURL:       srv.URL + "/checksums.txt",
		InstallerAssetURL:  srv.URL + "/installer.exe",
		InstallerAssetName: "RedShell-amd64-installer.exe",
		InstallerAssetSize: int64(len(body)),
	}

	rec := &installerCallRecord{}
	svc, er, _, quitCalled := newInstallerTestService(t, rec, rel, srv.Client())

	svc.RunCheck(context.Background(), "manual")
	if err := svc.InstallAvailable(context.Background()); err == nil {
		t.Fatal("expected checksum mismatch to error")
	}
	if rec.callCount() != 0 {
		t.Fatal("installerSpawn must not be called on checksum mismatch")
	}
	if *quitCalled {
		t.Fatal("quitApp must not be called on checksum mismatch")
	}
	if svc.InProgress() {
		t.Fatal("inProgress must remain false on checksum mismatch")
	}
	if !contains(er.names(), "updater:error") {
		t.Fatalf("expected error event, got %v", er.names())
	}
}
