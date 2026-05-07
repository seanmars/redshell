package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultGitLabHost = "https://gitlab.com"

type GitLabProvider struct {
	Host       string
	Project    string
	AssetName  string
	HTTPClient *http.Client

	mu          sync.Mutex
	lastETag    string
	lastRelease *Release
}

func NewGitLabProvider(host, project, assetName string, httpClient *http.Client) (*GitLabProvider, error) {
	if host == "" {
		host = defaultGitLabHost
	}
	parsed, err := url.Parse(host)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, fmt.Errorf("invalid gitlab host %q (must be http or https URL)", host)
	}
	if !strings.Contains(project, "/") || strings.TrimSpace(project) == "" {
		return nil, fmt.Errorf("invalid gitlab project %q (expected group/project)", project)
	}
	if assetName == "" {
		return nil, fmt.Errorf("gitlab provider: assetName must not be empty")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GitLabProvider{
		Host:       strings.TrimRight(host, "/"),
		Project:    project,
		AssetName:  assetName,
		HTTPClient: httpClient,
	}, nil
}

func (g *GitLabProvider) Name() string { return "gitlab" }

func (g *GitLabProvider) LatestRelease(ctx context.Context) (Release, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/releases/permalink/latest", g.Host, url.PathEscape(g.Project))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, fmt.Errorf("gitlab: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	g.mu.Lock()
	etag := g.lastETag
	cached := g.lastRelease
	g.mu.Unlock()
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("gitlab: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		if cached == nil {
			return Release{}, fmt.Errorf("gitlab: 304 returned with no cached release")
		}
		return *cached, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Release{}, fmt.Errorf("gitlab: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw gitlabRelease
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Release{}, fmt.Errorf("gitlab: decode response: %w", err)
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

type gitlabRelease struct {
	TagName     string       `json:"tag_name"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	ReleasedAt  time.Time    `json:"released_at"`
	Assets      gitlabAssets `json:"assets"`
}

type gitlabAssets struct {
	Links []gitlabLink `json:"links"`
}

type gitlabLink struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	DirectAssetURL string `json:"direct_asset_url"`
}

func (l gitlabLink) downloadURL() string {
	if l.DirectAssetURL != "" {
		return l.DirectAssetURL
	}
	return l.URL
}

func (r gitlabRelease) toRelease(binaryName string) (Release, error) {
	if r.TagName == "" {
		return Release{}, fmt.Errorf("gitlab: response missing tag_name")
	}
	var binary, checksums *gitlabLink
	for i := range r.Assets.Links {
		switch r.Assets.Links[i].Name {
		case binaryName:
			binary = &r.Assets.Links[i]
		case ChecksumsAssetName:
			checksums = &r.Assets.Links[i]
		}
	}
	if binary == nil {
		return Release{}, fmt.Errorf("gitlab: %w: %s in release %s", ErrAssetNotFound, binaryName, r.TagName)
	}
	if checksums == nil {
		return Release{}, fmt.Errorf("gitlab: %w in release %s", ErrChecksumsNotFound, r.TagName)
	}
	return Release{
		Version:      r.TagName,
		PublishedAt:  r.ReleasedAt,
		Notes:        r.Description,
		AssetURL:     binary.downloadURL(),
		AssetName:    binary.Name,
		AssetSize:    0,
		ChecksumsURL: checksums.downloadURL(),
	}, nil
}
