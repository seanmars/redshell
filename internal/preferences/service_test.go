package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, ".redshell", "preferences.json")
	return NewServiceWithPath(path), path
}

func TestService_GetReturnsDefaultsWhenFileMissing(t *testing.T) {
	svc, path := newService(t)

	prefs, err := svc.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if prefs.CloseBehavior != CloseBehaviorUnset {
		t.Fatalf("expected unset, got %q", prefs.CloseBehavior)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected file not to be created on read")
	}
}

func TestService_SetCloseBehaviorRoundTrip(t *testing.T) {
	svc, path := newService(t)

	if err := svc.SetCloseBehavior(CloseBehaviorMinimizeToTray); err != nil {
		t.Fatalf("SetCloseBehavior: %v", err)
	}

	got, err := svc.GetCloseBehavior()
	if err != nil {
		t.Fatalf("GetCloseBehavior: %v", err)
	}
	if got != CloseBehaviorMinimizeToTray {
		t.Fatalf("expected minimize-to-tray, got %q", got)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var stored Preferences
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if stored.CloseBehavior != CloseBehaviorMinimizeToTray {
		t.Fatalf("expected stored minimize-to-tray, got %q", stored.CloseBehavior)
	}
}

func TestService_GetReturnsErrorOnMalformedJSON(t *testing.T) {
	svc, path := newService(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := svc.Get(); err == nil {
		t.Fatal("expected malformed JSON to surface an error")
	}
}

func TestService_SetCloseBehaviorRejectsInvalidValue(t *testing.T) {
	svc, path := newService(t)

	if err := svc.SetCloseBehavior("nope"); err == nil {
		t.Fatal("expected invalid value to fail")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected file not to be created on rejected set")
	}
}

func TestService_OnChangeFiresOnRealChange(t *testing.T) {
	svc, _ := newService(t)

	var calls atomic.Int32
	var lastValue atomic.Value
	svc.OnChange(func(p Preferences) {
		calls.Add(1)
		lastValue.Store(p.CloseBehavior)
	})

	if err := svc.SetCloseBehavior(CloseBehaviorExit); err != nil {
		t.Fatalf("SetCloseBehavior: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 callback, got %d", got)
	}
	if got, _ := lastValue.Load().(string); got != CloseBehaviorExit {
		t.Fatalf("expected callback with exit, got %q", got)
	}
}

func TestService_OnChangeSilentOnNoOpSet(t *testing.T) {
	svc, _ := newService(t)

	if err := svc.SetCloseBehavior(CloseBehaviorExit); err != nil {
		t.Fatalf("SetCloseBehavior: %v", err)
	}

	var calls atomic.Int32
	svc.OnChange(func(Preferences) { calls.Add(1) })

	if err := svc.SetCloseBehavior(CloseBehaviorExit); err != nil {
		t.Fatalf("SetCloseBehavior repeat: %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected 0 callbacks for no-op set, got %d", got)
	}
}

func TestService_GetReturnsErrorForInvalidStoredValue(t *testing.T) {
	svc, path := newService(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"closeBehavior":"weird"}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := svc.Get(); err == nil {
		t.Fatal("expected invalid stored value to error")
	}
}

func TestService_GetReturnsDefaultAutoUpdateWhenBlockMissing(t *testing.T) {
	svc, _ := newService(t)
	got, err := svc.GetAutoUpdate()
	if err != nil {
		t.Fatalf("GetAutoUpdate: %v", err)
	}
	want := defaultAutoUpdate()
	if got != want {
		t.Fatalf("default autoUpdate mismatch: got %#v want %#v", got, want)
	}
}

func TestService_PartialAutoUpdateBlockFillsDefaults(t *testing.T) {
	svc, path := newService(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	body := `{"closeBehavior":"exit","autoUpdate":{"enabled":false}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := svc.GetAutoUpdate()
	if err != nil {
		t.Fatalf("GetAutoUpdate: %v", err)
	}
	if got.Enabled {
		t.Fatal("expected Enabled=false to be preserved")
	}
	if got.IntervalHours != defaultAutoUpdateInterval {
		t.Fatalf("expected default intervalHours=%d, got %d", defaultAutoUpdateInterval, got.IntervalHours)
	}
	if got.Source != AutoUpdateSourceGitHub {
		t.Fatalf("expected default source github, got %q", got.Source)
	}
	if got.GithubRepo != defaultAutoUpdateGithubRepo {
		t.Fatalf("expected default githubRepo, got %q", got.GithubRepo)
	}
}

func TestService_SetAutoUpdateIntervalRejectsInvalid(t *testing.T) {
	svc, _ := newService(t)
	for _, bad := range []int{0, 2, 3, 7, 169, -1} {
		if err := svc.SetAutoUpdateInterval(bad); err == nil {
			t.Fatalf("expected interval %d to be rejected", bad)
		}
	}
	for _, ok := range []int{1, 6, 12, 24, 168} {
		if err := svc.SetAutoUpdateInterval(ok); err != nil {
			t.Fatalf("expected interval %d to be accepted: %v", ok, err)
		}
	}
}

func TestService_SetAutoUpdateSourceRejectsInvalid(t *testing.T) {
	svc, _ := newService(t)
	for _, bad := range []string{"", "GitHub", "git-hub", "bitbucket"} {
		if err := svc.SetAutoUpdateSource(bad); err == nil {
			t.Fatalf("expected source %q to be rejected", bad)
		}
	}
	if err := svc.SetAutoUpdateSource(AutoUpdateSourceGitLab); err != nil {
		t.Fatalf("SetAutoUpdateSource(gitlab): %v", err)
	}
}

func TestService_SetAutoUpdateGitlabHostRequiresHTTPS(t *testing.T) {
	svc, _ := newService(t)
	for _, bad := range []string{"", "gitlab.com", "http://gitlab.com", "://"} {
		if err := svc.SetAutoUpdateGitlabHost(bad); err == nil {
			t.Fatalf("expected gitlabHost %q to be rejected", bad)
		}
	}
	if err := svc.SetAutoUpdateGitlabHost("https://gitlab.example.org"); err != nil {
		t.Fatalf("valid host rejected: %v", err)
	}
}

func TestService_SetAutoUpdateRepoSlugRequiresSlash(t *testing.T) {
	svc, _ := newService(t)
	for _, bad := range []string{"", "noslash", " owner/repo", "owner /repo"} {
		if err := svc.SetAutoUpdateGithubRepo(bad); err == nil {
			t.Fatalf("expected githubRepo %q to be rejected", bad)
		}
	}
	if err := svc.SetAutoUpdateGithubRepo("seanmars/redshell"); err != nil {
		t.Fatalf("valid githubRepo rejected: %v", err)
	}
}

func TestService_OnChangeFiresOnObservableAutoUpdateFields(t *testing.T) {
	cases := []struct {
		name string
		op   func(*Service) error
	}{
		{"enabled", func(s *Service) error { return s.SetAutoUpdateEnabled(false) }},
		{"interval", func(s *Service) error { return s.SetAutoUpdateInterval(24) }},
		{"source", func(s *Service) error { return s.SetAutoUpdateSource(AutoUpdateSourceGitLab) }},
		{"skipVersion", func(s *Service) error { return s.SetAutoUpdateSkipVersion("v0.5.0") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, _ := newService(t)
			var calls atomic.Int32
			svc.OnChange(func(Preferences) { calls.Add(1) })
			if err := c.op(svc); err != nil {
				t.Fatalf("op: %v", err)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("expected 1 callback for %s, got %d", c.name, got)
			}
		})
	}
}

func TestService_OnChangeSilentOnLastCheckedAt(t *testing.T) {
	svc, _ := newService(t)
	var calls atomic.Int32
	svc.OnChange(func(Preferences) { calls.Add(1) })
	if err := svc.SetAutoUpdateLastCheckedAt(time.Now()); err != nil {
		t.Fatalf("SetAutoUpdateLastCheckedAt: %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected 0 callbacks for lastCheckedAt write, got %d", got)
	}
}

func TestService_OnChangeSilentOnAdvancedRepoFields(t *testing.T) {
	svc, _ := newService(t)
	var calls atomic.Int32
	svc.OnChange(func(Preferences) { calls.Add(1) })
	if err := svc.SetAutoUpdateGithubRepo("other/redshell"); err != nil {
		t.Fatalf("SetAutoUpdateGithubRepo: %v", err)
	}
	if err := svc.SetAutoUpdateGitlabHost("https://gitlab.example.org"); err != nil {
		t.Fatalf("SetAutoUpdateGitlabHost: %v", err)
	}
	if err := svc.SetAutoUpdateGitlabProject("group/redshell"); err != nil {
		t.Fatalf("SetAutoUpdateGitlabProject: %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected 0 callbacks for advanced repo writes, got %d", got)
	}
}

func TestService_OnChangeSilentOnNoOpAutoUpdateSet(t *testing.T) {
	svc, _ := newService(t)
	if err := svc.SetAutoUpdateInterval(24); err != nil {
		t.Fatalf("SetAutoUpdateInterval: %v", err)
	}
	var calls atomic.Int32
	svc.OnChange(func(Preferences) { calls.Add(1) })
	if err := svc.SetAutoUpdateInterval(24); err != nil {
		t.Fatalf("repeat SetAutoUpdateInterval: %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected 0 callbacks for no-op set, got %d", got)
	}
}

func TestService_SetAutoUpdateBulkValidatesAndPersists(t *testing.T) {
	svc, path := newService(t)
	bad := AutoUpdate{
		Enabled:       true,
		IntervalHours: 99,
		Source:        AutoUpdateSourceGitHub,
		GithubRepo:    "seanmars/redshell",
		GitlabHost:    "https://gitlab.com",
		GitlabProject: "seanmars/redshell",
	}
	if err := svc.SetAutoUpdate(bad); err == nil {
		t.Fatal("expected invalid bulk set to fail")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected file not to be created on rejected bulk set")
	}

	good := defaultAutoUpdate()
	good.Enabled = false
	good.Source = AutoUpdateSourceGitLab
	if err := svc.SetAutoUpdate(good); err != nil {
		t.Fatalf("SetAutoUpdate: %v", err)
	}
	got, err := svc.GetAutoUpdate()
	if err != nil {
		t.Fatalf("GetAutoUpdate: %v", err)
	}
	if got != good {
		t.Fatalf("autoUpdate not persisted: got %#v want %#v", got, good)
	}
}

func TestService_SetAutoUpdateLastCheckedAtFormat(t *testing.T) {
	svc, _ := newService(t)
	now := time.Date(2026, 5, 7, 13, 30, 0, 0, time.UTC)
	if err := svc.SetAutoUpdateLastCheckedAt(now); err != nil {
		t.Fatalf("SetAutoUpdateLastCheckedAt: %v", err)
	}
	got, err := svc.GetAutoUpdate()
	if err != nil {
		t.Fatalf("GetAutoUpdate: %v", err)
	}
	want := "2026-05-07T13:30:00Z"
	if got.LastCheckedAt != want {
		t.Fatalf("expected %q, got %q", want, got.LastCheckedAt)
	}
}

func TestService_RejectsInvalidStoredAutoUpdate(t *testing.T) {
	svc, path := newService(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	body := `{"closeBehavior":"exit","autoUpdate":{"intervalHours":3,"source":"github","githubRepo":"seanmars/redshell","gitlabHost":"https://gitlab.com","gitlabProject":"seanmars/redshell"}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := svc.Get(); err == nil {
		t.Fatal("expected invalid stored intervalHours to error")
	}
}
