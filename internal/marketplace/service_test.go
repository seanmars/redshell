package marketplace

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestCacheDirName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"github.com::owner@repo", "github.com--owner@repo"},
		{"gitlab.com::group/subgroup@proj", "gitlab.com--group-subgroup@proj"},
		{"host::a:b", "host--a-b"},
		{`host::a\b`, "host--a-b"},
		{`host::a*b?c"d<e>f|g`, "host--a-b-c-d-e-f-g"},
		{"plain", "plain"},
	}
	for _, c := range cases {
		got := CacheDirName(c.in)
		if got != c.want {
			t.Errorf("CacheDirName(%q) = %q, want %q", c.in, got, c.want)
		}
		// Idempotency: running it twice must produce the same result.
		if got2 := CacheDirName(got); got2 != got {
			t.Errorf("CacheDirName not idempotent: %q -> %q", got, got2)
		}
	}
}

// makeBareRepo creates a bare git repo populated with a working clone that has
// the given files committed. Returns the bare repo path.
func makeBareRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	bare := filepath.Join(root, "remote.git")
	work := filepath.Join(root, "work")

	mustGit(t, "", "init", "--bare", bare)
	mustGit(t, "", "init", work)
	mustGit(t, work, "config", "user.email", "test@example.com")
	mustGit(t, work, "config", "user.name", "test")
	mustGit(t, work, "config", "commit.gpgsign", "false")

	for relPath, content := range files {
		full := filepath.Join(work, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", relPath, err)
		}
	}

	mustGit(t, work, "add", "-A")
	mustGit(t, work, "commit", "-m", "initial")
	mustGit(t, work, "branch", "-M", "main")
	mustGit(t, work, "remote", "add", "origin", bare)
	mustGit(t, work, "push", "-u", "origin", "main")
	return bare
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, string(out))
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	return NewServiceWithCacheRoot(filepath.Join(root, "marketplace.json"), filepath.Join(root, ".cache"))
}

func TestService_AddCreatesCache(t *testing.T) {
	bare := makeBareRepo(t, map[string]string{
		".claude-plugin/marketplace.json": `{"name":"Test","plugins":[]}`,
	})
	svc := newTestService(t)

	m, err := svc.addNormalized("file://" + filepath.ToSlash(bare))
	if err != nil {
		t.Fatalf("addNormalized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(svc.CacheDir(m.ID), ".git")); err != nil {
		t.Errorf("expected .git in cache dir: %v", err)
	}
	if got := m.Name["claude"]; got != "Test" {
		t.Errorf("claude display name = %q, want %q", got, "Test")
	}
}

func TestService_AddCloneFailureCleansCache(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	svc := newTestService(t)
	bogus := "file:///nonexistent/path/that/should/fail.git"

	if _, err := svc.addNormalized(bogus); err == nil {
		t.Fatal("expected addNormalized to fail against bogus URL")
	}
	id := GenerateID(bogus)
	if _, err := os.Stat(svc.CacheDir(id)); !os.IsNotExist(err) {
		t.Errorf("expected cache dir to be cleaned up after clone failure, stat err: %v", err)
	}
}

func TestService_RemoveDeletesCache(t *testing.T) {
	bare := makeBareRepo(t, map[string]string{
		".claude-plugin/marketplace.json": `{"name":"Test","plugins":[]}`,
	})
	svc := newTestService(t)

	m, err := svc.addNormalized("file://" + filepath.ToSlash(bare))
	if err != nil {
		t.Fatalf("addNormalized: %v", err)
	}
	cacheDir := svc.CacheDir(m.ID)
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("cache dir missing after Add: %v", err)
	}
	if err := svc.Remove(m.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Errorf("expected cache dir gone after Remove, stat err: %v", err)
	}
}

func TestService_RefreshUpdatesCache(t *testing.T) {
	bare := makeBareRepo(t, map[string]string{
		".claude-plugin/marketplace.json": `{"name":"v1","plugins":[]}`,
	})
	svc := newTestService(t)
	m, err := svc.addNormalized("file://" + filepath.ToSlash(bare))
	if err != nil {
		t.Fatalf("addNormalized: %v", err)
	}

	// Mutate the bare repo by pushing a new commit from a fresh working clone.
	work := t.TempDir()
	mustGit(t, "", "clone", bare, filepath.Join(work, "work"))
	workDir := filepath.Join(work, "work")
	mustGit(t, workDir, "config", "user.email", "test@example.com")
	mustGit(t, workDir, "config", "user.name", "test")
	mustGit(t, workDir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(workDir, ".claude-plugin", "marketplace.json"), []byte(`{"name":"v2","plugins":[]}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mustGit(t, workDir, "add", "-A")
	mustGit(t, workDir, "commit", "-m", "v2")
	mustGit(t, workDir, "push", "origin", "main")

	if err := svc.Refresh(m.ID); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	cached, err := os.ReadFile(filepath.Join(svc.CacheDir(m.ID), ".claude-plugin", "marketplace.json"))
	if err != nil {
		t.Fatalf("read cached manifest: %v", err)
	}
	if string(cached) != `{"name":"v2","plugins":[]}` {
		t.Errorf("expected v2 manifest after Refresh, got %q", string(cached))
	}
}

func TestService_RefreshConcurrentSameID(t *testing.T) {
	bare := makeBareRepo(t, map[string]string{
		".claude-plugin/marketplace.json": `{"name":"Test","plugins":[]}`,
	})
	svc := newTestService(t)
	m, err := svc.addNormalized("file://" + filepath.ToSlash(bare))
	if err != nil {
		t.Fatalf("addNormalized: %v", err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = svc.Refresh(m.ID)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent Refresh #%d: %v", i, err)
		}
	}
	lockPath := filepath.Join(svc.CacheDir(m.ID), ".git", "index.lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("expected no .git/index.lock after concurrent Refresh, got %v", err)
	}
}

func TestNormalizeGitURL(t *testing.T) {
	okCases := []struct {
		in, want string
	}{
		{"https://github.com/owner/repo", "https://github.com/owner/repo"},
		{"https://github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"http://example.com/owner/repo", "http://example.com/owner/repo"},
		{"git@github.com:owner/repo.git", "https://github.com/owner/repo"},
		{"git@gitlab.com:group/subgroup/repo", "https://gitlab.com/group/subgroup/repo"},
		{"github.com/owner/repo", "https://github.com/owner/repo"},
		{"  https://github.com/owner/repo  ", "https://github.com/owner/repo"},
	}
	for _, c := range okCases {
		got, err := normalizeGitURL(c.in)
		if err != nil {
			t.Errorf("normalizeGitURL(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("normalizeGitURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	rejectCases := []string{
		"file:///tmp/x.git",
		"ssh://git@github.com/owner/repo",
		"https://github.com/owner",
		"https://github.com",
		"git@github.com",
	}
	for _, in := range rejectCases {
		if _, err := normalizeGitURL(in); err == nil {
			t.Errorf("normalizeGitURL(%q) expected error, got nil", in)
		}
	}
}
