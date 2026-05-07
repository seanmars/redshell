package updater

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

const testGithubAssetName = "redshell-windows-amd64.exe"

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func newGithubServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GitHubProvider) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p, err := NewGitHubProvider("seanmars/redshell", testGithubAssetName, srv.Client())
	if err != nil {
		t.Fatalf("NewGitHubProvider: %v", err)
	}
	p.APIBase = srv.URL
	return srv, p
}

func TestGitHubProvider_NewRejectsInvalidSlug(t *testing.T) {
	for _, bad := range []string{"", "noslash", "/repo", "owner/", "/"} {
		if _, err := NewGitHubProvider(bad, testGithubAssetName, nil); err == nil {
			t.Fatalf("expected slug %q to be rejected", bad)
		}
	}
}

func TestGitHubProvider_NewRejectsEmptyAssetName(t *testing.T) {
	if _, err := NewGitHubProvider("seanmars/redshell", "", nil); err == nil {
		t.Fatal("expected empty assetName to be rejected")
	}
}

func TestGitHubProvider_LatestReleaseHappyPath(t *testing.T) {
	body := loadFixture(t, "github_latest.json")
	var captured struct {
		Path   string
		Accept string
	}
	_, p := newGithubServer(t, func(w http.ResponseWriter, r *http.Request) {
		captured.Path = r.URL.Path
		captured.Accept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"abc123"`)
		w.Write(body)
	})

	rel, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if want := "/repos/seanmars/redshell/releases/latest"; captured.Path != want {
		t.Fatalf("path mismatch: got %q want %q", captured.Path, want)
	}
	if captured.Accept != "application/vnd.github+json" {
		t.Fatalf("expected GitHub Accept header, got %q", captured.Accept)
	}
	if rel.Version != "v0.5.0" {
		t.Fatalf("Version: got %q want v0.5.0", rel.Version)
	}
	if rel.AssetName != testGithubAssetName {
		t.Fatalf("AssetName: got %q want %q", rel.AssetName, testGithubAssetName)
	}
	if rel.AssetSize != 12345678 {
		t.Fatalf("AssetSize: got %d want 12345678", rel.AssetSize)
	}
	const wantAssetURL = "https://api.github.com/repos/seanmars/redshell/releases/assets/1"
	const wantChecksumsURL = "https://api.github.com/repos/seanmars/redshell/releases/assets/3"
	if rel.AssetURL != wantAssetURL {
		t.Fatalf("AssetURL: got %q want %q (must be the API asset URL, not browser_download_url)", rel.AssetURL, wantAssetURL)
	}
	if rel.ChecksumsURL != wantChecksumsURL {
		t.Fatalf("ChecksumsURL: got %q want %q", rel.ChecksumsURL, wantChecksumsURL)
	}
	if rel.PublishedAt.IsZero() {
		t.Fatal("PublishedAt should be parsed")
	}
}

func TestGitHubProvider_ETagFlowReturns304Cached(t *testing.T) {
	body := loadFixture(t, "github_latest.json")
	var calls atomic.Int32
	_, p := newGithubServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("ETag", `"abc123"`)
			w.Write(body)
			return
		}
		if r.Header.Get("If-None-Match") != `"abc123"` {
			t.Errorf("expected If-None-Match header on second call, got %q", r.Header.Get("If-None-Match"))
		}
		w.WriteHeader(http.StatusNotModified)
	})

	first, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("first LatestRelease: %v", err)
	}
	second, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("second LatestRelease (304): %v", err)
	}
	if first.Version != second.Version || first.AssetURL != second.AssetURL {
		t.Fatalf("304 should return cached release, got first=%#v second=%#v", first, second)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("expected 2 calls, got %d", got)
	}
}

func TestGitHubProvider_404Errors(t *testing.T) {
	_, p := newGithubServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})
	if _, err := p.LatestRelease(context.Background()); err == nil {
		t.Fatal("expected 404 to surface as error")
	}
}

func TestGitHubProvider_MalformedJSONErrors(t *testing.T) {
	_, p := newGithubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{not-json"))
	})
	if _, err := p.LatestRelease(context.Background()); err == nil {
		t.Fatal("expected malformed JSON to surface as error")
	}
}

func TestGitHubProvider_MissingBinaryAssetErrors(t *testing.T) {
	_, p := newGithubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.5.0","assets":[{"name":"checksums.txt","browser_download_url":"https://example.com/checksums.txt","size":1}]}`))
	})
	_, err := p.LatestRelease(context.Background())
	if err == nil {
		t.Fatal("expected missing binary asset to error")
	}
	if !errors.Is(err, ErrAssetNotFound) {
		t.Fatalf("expected ErrAssetNotFound, got %v", err)
	}
}

func TestGitHubProvider_MissingChecksumsAssetErrors(t *testing.T) {
	_, p := newGithubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.5.0","assets":[{"name":"redshell-windows-amd64.exe","browser_download_url":"https://example.com/r.exe","size":1}]}`))
	})
	_, err := p.LatestRelease(context.Background())
	if err == nil {
		t.Fatal("expected missing checksums to error")
	}
	if !errors.Is(err, ErrChecksumsNotFound) {
		t.Fatalf("expected ErrChecksumsNotFound, got %v", err)
	}
}

func TestGitHubProvider_304WithoutCacheErrors(t *testing.T) {
	_, p := newGithubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	})
	if _, err := p.LatestRelease(context.Background()); err == nil {
		t.Fatal("expected first-call 304 with no cached release to error")
	}
}

func TestGitHubProvider_NameIdentifiesProvider(t *testing.T) {
	p, err := NewGitHubProvider("seanmars/redshell", testGithubAssetName, nil)
	if err != nil {
		t.Fatalf("NewGitHubProvider: %v", err)
	}
	if got := p.Name(); got != "github" {
		t.Fatalf("Name: got %q want github", got)
	}
}

func TestAssetNameForFormat(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"windows", "amd64", "redshell-windows-amd64.exe"},
		{"darwin", "arm64", "redshell-darwin-arm64"},
		{"linux", "amd64", "redshell-linux-amd64"},
	}
	for _, c := range cases {
		if got := AssetNameFor(c.goos, c.goarch); got != c.want {
			t.Errorf("AssetNameFor(%s,%s): got %q want %q", c.goos, c.goarch, got, c.want)
		}
	}
}
