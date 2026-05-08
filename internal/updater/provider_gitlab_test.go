package updater

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func newGitlabServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GitLabProvider) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p, err := NewGitLabProvider(srv.URL, "seanmars/redshell", testGithubAssetName, srv.Client())
	if err != nil {
		t.Fatalf("NewGitLabProvider: %v", err)
	}
	return srv, p
}

func TestGitLabProvider_NewValidatesInputs(t *testing.T) {
	cases := []struct {
		host, project, asset string
	}{
		{"ftp://gitlab.com", "g/p", "x"},
		{"://", "g/p", "x"},
		{"https://gitlab.com", "noslash", "x"},
		{"https://gitlab.com", "", "x"},
		{"https://gitlab.com", "g/p", ""},
	}
	for _, c := range cases {
		if _, err := NewGitLabProvider(c.host, c.project, c.asset, nil); err == nil {
			t.Errorf("expected (host=%q project=%q asset=%q) to be rejected", c.host, c.project, c.asset)
		}
	}
}

func TestGitLabProvider_LatestReleaseHappyPath(t *testing.T) {
	body := loadFixture(t, "gitlab_latest.json")
	var captured struct {
		Path   string
		Accept string
	}
	_, p := newGitlabServer(t, func(w http.ResponseWriter, r *http.Request) {
		captured.Path = r.URL.EscapedPath()
		captured.Accept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"gl-abc"`)
		w.Write(body)
	})
	p.SetInstallerAssetName("RedShell-amd64-installer.exe")

	rel, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	wantPath := "/api/v4/projects/seanmars%2Fredshell/releases/permalink/latest"
	if captured.Path != wantPath {
		t.Fatalf("path mismatch: got %q want %q", captured.Path, wantPath)
	}
	if !strings.Contains(captured.Accept, "json") {
		t.Fatalf("expected JSON Accept header, got %q", captured.Accept)
	}
	if rel.Version != "v0.5.0" {
		t.Fatalf("Version: got %q want v0.5.0", rel.Version)
	}
	if rel.AssetName != testGithubAssetName {
		t.Fatalf("AssetName: got %q want %q", rel.AssetName, testGithubAssetName)
	}
	if !strings.HasSuffix(rel.AssetURL, "/redshell-windows-amd64.exe") {
		t.Fatalf("AssetURL unexpected: %q", rel.AssetURL)
	}
	if !strings.HasSuffix(rel.ChecksumsURL, "/checksums.txt") {
		t.Fatalf("ChecksumsURL unexpected: %q", rel.ChecksumsURL)
	}
	if rel.PublishedAt.IsZero() {
		t.Fatal("PublishedAt should be parsed")
	}
	if rel.InstallerAssetName != "RedShell-amd64-installer.exe" {
		t.Fatalf("InstallerAssetName: got %q want RedShell-amd64-installer.exe", rel.InstallerAssetName)
	}
	if !strings.HasSuffix(rel.InstallerAssetURL, "/RedShell-amd64-installer.exe") {
		t.Fatalf("InstallerAssetURL unexpected: %q", rel.InstallerAssetURL)
	}
}

func TestGitLabProvider_ETagFlowReturns304Cached(t *testing.T) {
	body := loadFixture(t, "gitlab_latest.json")
	var calls atomic.Int32
	_, p := newGitlabServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("ETag", `"gl-1"`)
			w.Write(body)
			return
		}
		if r.Header.Get("If-None-Match") != `"gl-1"` {
			t.Errorf("expected If-None-Match on second call, got %q", r.Header.Get("If-None-Match"))
		}
		w.WriteHeader(http.StatusNotModified)
	})

	first, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.Version != second.Version {
		t.Fatalf("304 should return cached release")
	}
}

func TestGitLabProvider_404Errors(t *testing.T) {
	_, p := newGitlabServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
	})
	if _, err := p.LatestRelease(context.Background()); err == nil {
		t.Fatal("expected 404 to surface")
	}
}

func TestGitLabProvider_MalformedJSONErrors(t *testing.T) {
	_, p := newGitlabServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{not-json"))
	})
	if _, err := p.LatestRelease(context.Background()); err == nil {
		t.Fatal("expected malformed JSON to error")
	}
}

func TestGitLabProvider_MissingPortableAssetReturnsEmptyAssetFields(t *testing.T) {
	// See TestGitHubProvider_MissingPortableAssetReturnsEmptyAssetFields
	// for the rationale: installer-only releases don't ship the portable.
	_, p := newGitlabServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.5.0","assets":{"links":[
			{"name":"RedShell-amd64-installer.exe","url":"https://example.com/installer"},
			{"name":"checksums.txt","url":"https://example.com/c.txt"}
		]}}`))
	})
	p.SetInstallerAssetName("RedShell-amd64-installer.exe")
	rel, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("installer-only release should not error at provider level: %v", err)
	}
	if rel.AssetURL != "" || rel.AssetName != "" {
		t.Fatalf("portable fields should be empty when absent, got url=%q name=%q", rel.AssetURL, rel.AssetName)
	}
	if rel.InstallerAssetName != "RedShell-amd64-installer.exe" {
		t.Fatalf("installer field should still be populated, got %q", rel.InstallerAssetName)
	}
}

func TestGitLabProvider_MissingChecksumsErrors(t *testing.T) {
	_, p := newGitlabServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.5.0","assets":{"links":[{"name":"redshell-windows-amd64.exe","url":"https://example.com/r.exe"}]}}`))
	})
	_, err := p.LatestRelease(context.Background())
	if err == nil {
		t.Fatal("expected missing checksums to error")
	}
	if !errors.Is(err, ErrChecksumsNotFound) {
		t.Fatalf("expected ErrChecksumsNotFound, got %v", err)
	}
}

func TestGitLabProvider_DirectAssetURLPreferred(t *testing.T) {
	body := []byte(`{"tag_name":"v0.5.0","released_at":"2026-05-01T10:00:00Z","assets":{"links":[
		{"name":"redshell-windows-amd64.exe","url":"https://gitlab.com/redirect/r.exe","direct_asset_url":"https://gitlab.com/direct/r.exe"},
		{"name":"checksums.txt","url":"https://gitlab.com/redirect/c.txt"}
	]}}`)
	_, p := newGitlabServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	})
	rel, err := p.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.AssetURL != "https://gitlab.com/direct/r.exe" {
		t.Fatalf("expected direct_asset_url to be preferred, got %q", rel.AssetURL)
	}
	if rel.ChecksumsURL != "https://gitlab.com/redirect/c.txt" {
		t.Fatalf("expected url fallback when direct missing, got %q", rel.ChecksumsURL)
	}
}

func TestGitLabProvider_NameIdentifiesProvider(t *testing.T) {
	p, err := NewGitLabProvider("https://gitlab.com", "seanmars/redshell", testGithubAssetName, nil)
	if err != nil {
		t.Fatalf("NewGitLabProvider: %v", err)
	}
	if got := p.Name(); got != "gitlab" {
		t.Fatalf("Name: got %q want gitlab", got)
	}
}
