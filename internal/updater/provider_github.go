package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultGitHubAPIBase = "https://api.github.com"

type GitHubProvider struct {
	Owner      string
	Repo       string
	APIBase    string
	AssetName  string
	HTTPClient *http.Client

	mu          sync.Mutex
	lastETag    string
	lastRelease *Release
}

func NewGitHubProvider(repoSlug, assetName string, httpClient *http.Client) (*GitHubProvider, error) {
	owner, repo, ok := strings.Cut(repoSlug, "/")
	if !ok || owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github repo %q (expected owner/repo)", repoSlug)
	}
	if assetName == "" {
		return nil, fmt.Errorf("github provider: assetName must not be empty")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GitHubProvider{
		Owner:      owner,
		Repo:       repo,
		APIBase:    defaultGitHubAPIBase,
		AssetName:  assetName,
		HTTPClient: httpClient,
	}, nil
}

func (g *GitHubProvider) Name() string { return "github" }

func (g *GitHubProvider) LatestRelease(ctx context.Context) (Release, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases/latest", strings.TrimRight(g.APIBase, "/"), g.Owner, g.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, fmt.Errorf("github: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	g.mu.Lock()
	etag := g.lastETag
	cached := g.lastRelease
	g.mu.Unlock()
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("github: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		if cached == nil {
			return Release{}, fmt.Errorf("github: 304 returned with no cached release")
		}
		return *cached, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Release{}, fmt.Errorf("github: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Release{}, fmt.Errorf("github: decode response: %w", err)
	}

	release, err := raw.toRelease(g.AssetName)
	if err != nil {
		return Release{}, err
	}

	g.mu.Lock()
	if h := resp.Header.Get("ETag"); h != "" {
		g.lastETag = h
	}
	cp := release
	g.lastRelease = &cp
	g.mu.Unlock()

	return release, nil
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	PublishedAt time.Time     `json:"published_at"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func (r githubRelease) toRelease(binaryName string) (Release, error) {
	if r.TagName == "" {
		return Release{}, fmt.Errorf("github: response missing tag_name")
	}
	var binary, checksums *githubAsset
	for i := range r.Assets {
		switch r.Assets[i].Name {
		case binaryName:
			binary = &r.Assets[i]
		case ChecksumsAssetName:
			checksums = &r.Assets[i]
		}
	}
	if binary == nil {
		return Release{}, fmt.Errorf("github: %w: %s in release %s", ErrAssetNotFound, binaryName, r.TagName)
	}
	if checksums == nil {
		return Release{}, fmt.Errorf("github: %w in release %s", ErrChecksumsNotFound, r.TagName)
	}
	return Release{
		Version:      r.TagName,
		PublishedAt:  r.PublishedAt,
		Notes:        r.Body,
		AssetURL:     binary.URL,
		AssetName:    binary.Name,
		AssetSize:    binary.Size,
		ChecksumsURL: checksums.URL,
	}, nil
}
