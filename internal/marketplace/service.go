package marketplace

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"redshell/internal/sysproc"
)

type Marketplace struct {
	ID      string            `json:"id"`
	URL     string            `json:"url"`
	Name    map[string]string `json:"name,omitempty"`
	AddedAt string            `json:"addedAt"`
}

type Service struct {
	filePath  string
	cacheRoot string

	cacheMuLock sync.Mutex
	cacheMu     map[string]*sync.Mutex
}

// AgentMarketplaceFiles maps agent id → path inside the repo.
var AgentMarketplaceFiles = map[string]string{
	"claude":  ".claude-plugin/marketplace.json",
	"copilot": ".github/plugin/marketplace.json",
}

func NewService() *Service {
	home, _ := os.UserHomeDir()
	return &Service{
		filePath:  filepath.Join(home, ".redshell", "marketplace.json"),
		cacheRoot: filepath.Join(home, ".redshell", ".cache"),
		cacheMu:   make(map[string]*sync.Mutex),
	}
}

func NewServiceWithPath(path string) *Service {
	dir := filepath.Dir(path)
	return &Service{
		filePath:  path,
		cacheRoot: filepath.Join(dir, ".cache"),
		cacheMu:   make(map[string]*sync.Mutex),
	}
}

func NewServiceWithCacheRoot(filePath, cacheRoot string) *Service {
	return &Service{
		filePath:  filePath,
		cacheRoot: cacheRoot,
		cacheMu:   make(map[string]*sync.Mutex),
	}
}

// CacheRoot returns the directory under which all marketplace caches live.
func (s *Service) CacheRoot() string {
	return s.cacheRoot
}

// CacheDir returns the absolute path of a single marketplace's cache directory.
func (s *Service) CacheDir(id string) string {
	return filepath.Join(s.cacheRoot, CacheDirName(id))
}

// CacheDirName converts a marketplace ID to a filesystem-safe directory name
// by replacing characters that are invalid on Windows or POSIX path components.
// '@' is intentionally preserved because it is filesystem-safe everywhere and
// keeps the directory name readable as the original ID.
func CacheDirName(id string) string {
	return strings.NewReplacer(
		":", "-",
		"/", "-",
		"\\", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	).Replace(id)
}

func (s *Service) cacheLock(id string) *sync.Mutex {
	s.cacheMuLock.Lock()
	defer s.cacheMuLock.Unlock()
	if mu, ok := s.cacheMu[id]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	s.cacheMu[id] = mu
	return mu
}

func (s *Service) List() ([]Marketplace, error) {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return []Marketplace{}, nil
	}
	if err != nil {
		return nil, err
	}
	var list []Marketplace
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Service) Add(rawURL string) (Marketplace, error) {
	cleanURL, err := normalizeGitURL(rawURL)
	if err != nil {
		return Marketplace{}, err
	}
	return s.addNormalized(cleanURL)
}

// addNormalized performs the cache-and-register flow for an already-validated
// URL. It is the seam used by tests that need to exercise cache logic against
// local file:// fixtures without going through normalizeGitURL.
func (s *Service) addNormalized(cleanURL string) (Marketplace, error) {
	id := GenerateID(cleanURL)
	list, err := s.List()
	if err != nil {
		return Marketplace{}, err
	}
	for _, m := range list {
		if m.ID == id {
			return Marketplace{}, errors.New("marketplace already registered: " + id)
		}
	}

	names, err := s.ensureCacheAndReadNames(id, cleanURL)
	if err != nil {
		return Marketplace{}, err
	}

	m := Marketplace{
		ID:      id,
		URL:     cleanURL,
		AddedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if len(names) > 0 {
		m.Name = names
	}

	list = append(list, m)
	if err := s.write(list); err != nil {
		return Marketplace{}, err
	}
	return m, nil
}

func (s *Service) Remove(id string) error {
	list, err := s.List()
	if err != nil {
		return err
	}
	filtered := list[:0]
	found := false
	for _, m := range list {
		if m.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, m)
	}
	if !found {
		return errors.New("marketplace not found: " + id)
	}
	if err := s.write(filtered); err != nil {
		return err
	}

	mu := s.cacheLock(id)
	mu.Lock()
	defer mu.Unlock()
	// Best-effort cache cleanup; surface but do not fail the removal.
	if rmErr := os.RemoveAll(s.CacheDir(id)); rmErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove cache dir for %s: %v\n", id, rmErr)
	}
	return nil
}

// Refresh updates a single marketplace's cache. If the cache directory is
// missing or has no .git subdirectory it performs a fresh shallow clone;
// otherwise it does git fetch + reset --hard FETCH_HEAD.
func (s *Service) Refresh(id string) error {
	list, err := s.List()
	if err != nil {
		return err
	}
	var target *Marketplace
	for i := range list {
		if list[i].ID == id {
			target = &list[i]
			break
		}
	}
	if target == nil {
		return errors.New("marketplace not found: " + id)
	}

	mu := s.cacheLock(id)
	mu.Lock()
	defer mu.Unlock()

	dir := s.CacheDir(id)
	if !cacheIsClone(dir) {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			return fmt.Errorf("git refresh: cleanup partial cache: %w", rmErr)
		}
		if err := gitClone(target.URL, dir); err != nil {
			return fmt.Errorf("git refresh: %w", err)
		}
		return nil
	}

	if err := gitFetchReset(dir); err != nil {
		return fmt.Errorf("git refresh: %w", err)
	}
	return nil
}

// RefreshAll iterates every registered marketplace and refreshes each cache.
// Per-marketplace failures do not abort the loop. Errors are formatted as
// "[<id>] git refresh: <reason>" so the frontend can attribute them to a
// specific marketplace section.
func (s *Service) RefreshAll() ([]string, []string) {
	list, err := s.List()
	if err != nil {
		return nil, []string{err.Error()}
	}
	var refreshed []string
	var errs []string
	for _, m := range list {
		if err := s.Refresh(m.ID); err != nil {
			errs = append(errs, fmt.Sprintf("[%s] %s", m.ID, err.Error()))
			continue
		}
		refreshed = append(refreshed, m.ID)
	}
	return refreshed, errs
}

func GenerateID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return sanitize(rawURL)
	}
	path := strings.TrimSuffix(u.Path, ".git")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return sanitize(rawURL)
	}
	project := parts[len(parts)-1]
	group := strings.Join(parts[:len(parts)-1], "/")
	return u.Hostname() + "::" + group + "@" + project
}

// ensureCacheAndReadNames clones the repo into the persistent cache (or leaves
// an existing clone in place) and reads each agent's marketplace.json to
// extract its display name. Caller must already hold registry-level coordination
// if needed; this function locks the per-cache mutex.
func (s *Service) ensureCacheAndReadNames(id, repoURL string) (map[string]string, error) {
	mu := s.cacheLock(id)
	mu.Lock()
	defer mu.Unlock()

	dir := s.CacheDir(id)
	if !cacheIsClone(dir) {
		// Remove any partial directory before cloning so we never inherit
		// half-written state from a previous failed Add.
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			return nil, fmt.Errorf("cleanup partial cache: %w", rmErr)
		}
		if err := os.MkdirAll(s.cacheRoot, 0o755); err != nil {
			return nil, fmt.Errorf("create cache root: %w", err)
		}
		if err := gitClone(repoURL, dir); err != nil {
			// Clean up partial clone before returning.
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("git clone: %w", err)
		}
	}

	names := make(map[string]string)
	for agentID, relPath := range AgentMarketplaceFiles {
		full := filepath.Join(dir, filepath.FromSlash(relPath))
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		var mktJson struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &mktJson); err != nil || mktJson.Name == "" {
			continue
		}
		names[agentID] = mktJson.Name
	}
	return names, nil
}

// cacheIsClone reports whether dir exists and contains a .git subdirectory.
func cacheIsClone(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

func gitClone(repoURL, dir string) error {
	cmd := exec.Command("git", "clone", "--depth=1", repoURL, dir)
	cmd.SysProcAttr = sysproc.Hidden()
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	return nil
}

func gitFetchReset(dir string) error {
	fetchCmd := exec.Command("git", "-C", dir, "fetch", "--depth=1", "origin")
	fetchCmd.SysProcAttr = sysproc.Hidden()
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	resetCmd := exec.Command("git", "-C", dir, "reset", "--hard", "FETCH_HEAD")
	resetCmd.SysProcAttr = sysproc.Hidden()
	if out, err := resetCmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	return nil
}

func (s *Service) write(list []Marketplace) error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0o644)
}

func sanitize(s string) string {
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, ".git")
	return strings.NewReplacer("/", "-", ".", "-").Replace(s)
}

// normalizeGitURL accepts common git URL variants and returns a canonical
// https:// URL without a .git suffix, or an error if the input is not a
// recognizable git repository URL.
func normalizeGitURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	// Convert SSH format: git@host:path/to/repo or git@host:path/to/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		rest := strings.TrimPrefix(rawURL, "git@")
		idx := strings.Index(rest, ":")
		if idx < 0 {
			return "", errors.New("invalid SSH URL: missing ':' separator")
		}
		host := rest[:idx]
		path := strings.TrimPrefix(rest[idx+1:], "/")
		rawURL = "https://" + host + "/" + path
	}

	// Add https:// scheme if missing entirely
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("URL must use https:// scheme, got: %s", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("URL is missing a host")
	}

	// Strip .git suffix and validate owner/repo structure
	path := strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), ".git")
	parts := strings.Split(path, "/")
	// Require at least two non-empty path segments (owner + repo)
	if len(parts) < 2 || parts[0] == "" || parts[len(parts)-1] == "" {
		return "", errors.New("URL must contain at least owner/repo (e.g. https://github.com/owner/repo)")
	}

	return u.Scheme + "://" + u.Host + "/" + path, nil
}
