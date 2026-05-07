package preferences

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	CloseBehaviorUnset          = "unset"
	CloseBehaviorExit           = "exit"
	CloseBehaviorMinimizeToTray = "minimize-to-tray"

	AutoUpdateSourceGitHub = "github"
	AutoUpdateSourceGitLab = "gitlab"

	defaultAutoUpdateGithubRepo    = "seanmars/redshell"
	defaultAutoUpdateGitlabHost    = "https://gitlab.com"
	defaultAutoUpdateGitlabProject = "seanmars/redshell"
	defaultAutoUpdateInterval      = 6
)

var allowedAutoUpdateIntervalHours = []int{1, 6, 12, 24, 168}

type AutoUpdate struct {
	Enabled       bool   `json:"enabled"`
	IntervalHours int    `json:"intervalHours"`
	Source        string `json:"source"`
	GithubRepo    string `json:"githubRepo"`
	GitlabHost    string `json:"gitlabHost"`
	GitlabProject string `json:"gitlabProject"`
	SkipVersion   string `json:"skipVersion"`
	LastCheckedAt string `json:"lastCheckedAt"`
}

type Preferences struct {
	CloseBehavior string     `json:"closeBehavior"`
	AutoUpdate    AutoUpdate `json:"autoUpdate"`
}

type Service struct {
	filePath string

	mu        sync.RWMutex
	cached    *Preferences
	observers []func(Preferences)
}

func NewService() *Service {
	home, _ := os.UserHomeDir()
	return &Service{
		filePath: filepath.Join(home, ".redshell", "preferences.json"),
	}
}

func NewServiceWithPath(filePath string) *Service {
	return &Service{filePath: filePath}
}

func (s *Service) Get() (Preferences, error) {
	s.mu.RLock()
	if s.cached != nil {
		p := *s.cached
		s.mu.RUnlock()
		return p, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil {
		return *s.cached, nil
	}

	prefs, err := s.readLocked()
	if err != nil {
		return Preferences{}, err
	}
	s.cached = &prefs
	return prefs, nil
}

func (s *Service) GetCloseBehavior() (string, error) {
	prefs, err := s.Get()
	if err != nil {
		return "", err
	}
	return prefs.CloseBehavior, nil
}

func (s *Service) SetCloseBehavior(value string) error {
	if !isValidCloseBehavior(value) {
		return fmt.Errorf("invalid close behavior: %q", value)
	}

	s.mu.Lock()
	current, err := s.cachedOrReadLocked()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if current.CloseBehavior == value {
		s.mu.Unlock()
		return nil
	}

	next := current
	next.CloseBehavior = value
	if err := s.writeLocked(next); err != nil {
		s.mu.Unlock()
		return err
	}
	s.cached = &next
	observers := append([]func(Preferences){}, s.observers...)
	s.mu.Unlock()

	for _, cb := range observers {
		cb(next)
	}
	return nil
}

func (s *Service) GetAutoUpdate() (AutoUpdate, error) {
	prefs, err := s.Get()
	if err != nil {
		return AutoUpdate{}, err
	}
	return prefs.AutoUpdate, nil
}

func (s *Service) SetAutoUpdateEnabled(value bool) error {
	return s.mutateAutoUpdate(true, func(a *AutoUpdate) error {
		a.Enabled = value
		return nil
	})
}

func (s *Service) SetAutoUpdateInterval(hours int) error {
	if !isValidAutoUpdateIntervalHours(hours) {
		return fmt.Errorf("intervalHours %d is not in allowed set %v", hours, allowedAutoUpdateIntervalHours)
	}
	return s.mutateAutoUpdate(true, func(a *AutoUpdate) error {
		a.IntervalHours = hours
		return nil
	})
}

func (s *Service) SetAutoUpdateSource(source string) error {
	if !isValidAutoUpdateSource(source) {
		return fmt.Errorf("source %q must be %q or %q", source, AutoUpdateSourceGitHub, AutoUpdateSourceGitLab)
	}
	return s.mutateAutoUpdate(true, func(a *AutoUpdate) error {
		a.Source = source
		return nil
	})
}

func (s *Service) SetAutoUpdateGithubRepo(repo string) error {
	if err := validateRepoSlug(repo); err != nil {
		return fmt.Errorf("githubRepo: %w", err)
	}
	return s.mutateAutoUpdate(false, func(a *AutoUpdate) error {
		a.GithubRepo = repo
		return nil
	})
}

func (s *Service) SetAutoUpdateGitlabHost(host string) error {
	if err := validateGitlabHost(host); err != nil {
		return fmt.Errorf("gitlabHost: %w", err)
	}
	return s.mutateAutoUpdate(false, func(a *AutoUpdate) error {
		a.GitlabHost = host
		return nil
	})
}

func (s *Service) SetAutoUpdateGitlabProject(project string) error {
	if err := validateRepoSlug(project); err != nil {
		return fmt.Errorf("gitlabProject: %w", err)
	}
	return s.mutateAutoUpdate(false, func(a *AutoUpdate) error {
		a.GitlabProject = project
		return nil
	})
}

func (s *Service) SetAutoUpdateSkipVersion(version string) error {
	return s.mutateAutoUpdate(true, func(a *AutoUpdate) error {
		a.SkipVersion = version
		return nil
	})
}

func (s *Service) SetAutoUpdateLastCheckedAt(at time.Time) error {
	value := ""
	if !at.IsZero() {
		value = at.UTC().Format(time.RFC3339)
	}
	return s.mutateAutoUpdate(false, func(a *AutoUpdate) error {
		a.LastCheckedAt = value
		return nil
	})
}

func (s *Service) SetAutoUpdate(next AutoUpdate) error {
	next = applyAutoUpdateDefaults(next)
	if err := validateAutoUpdate(next); err != nil {
		return err
	}
	s.mu.Lock()
	current, err := s.cachedOrReadLocked()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if current.AutoUpdate == next {
		s.mu.Unlock()
		return nil
	}
	notify := autoUpdateObservableChange(current.AutoUpdate, next)
	updated := current
	updated.AutoUpdate = next
	if err := s.writeLocked(updated); err != nil {
		s.mu.Unlock()
		return err
	}
	s.cached = &updated
	var observers []func(Preferences)
	if notify {
		observers = append(observers, s.observers...)
	}
	s.mu.Unlock()
	for _, cb := range observers {
		cb(updated)
	}
	return nil
}

func (s *Service) mutateAutoUpdate(observable bool, mutate func(*AutoUpdate) error) error {
	s.mu.Lock()
	current, err := s.cachedOrReadLocked()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	next := current.AutoUpdate
	if err := mutate(&next); err != nil {
		s.mu.Unlock()
		return err
	}
	if next == current.AutoUpdate {
		s.mu.Unlock()
		return nil
	}
	if err := validateAutoUpdate(next); err != nil {
		s.mu.Unlock()
		return err
	}
	updated := current
	updated.AutoUpdate = next
	if err := s.writeLocked(updated); err != nil {
		s.mu.Unlock()
		return err
	}
	s.cached = &updated
	var observers []func(Preferences)
	if observable {
		observers = append(observers, s.observers...)
	}
	s.mu.Unlock()
	for _, cb := range observers {
		cb(updated)
	}
	return nil
}

func autoUpdateObservableChange(prev, next AutoUpdate) bool {
	return prev.Enabled != next.Enabled ||
		prev.IntervalHours != next.IntervalHours ||
		prev.Source != next.Source ||
		prev.SkipVersion != next.SkipVersion
}

func (s *Service) OnChange(cb func(Preferences)) {
	if cb == nil {
		return
	}
	s.mu.Lock()
	s.observers = append(s.observers, cb)
	s.mu.Unlock()
}

func (s *Service) cachedOrReadLocked() (Preferences, error) {
	if s.cached != nil {
		return *s.cached, nil
	}
	prefs, err := s.readLocked()
	if err != nil {
		return Preferences{}, err
	}
	s.cached = &prefs
	return prefs, nil
}

func (s *Service) readLocked() (Preferences, error) {
	data, err := os.ReadFile(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return defaultPreferences(), nil
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("read preferences: %w", err)
	}

	var prefs Preferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Preferences{}, fmt.Errorf("parse preferences: %w", err)
	}
	if prefs.CloseBehavior == "" {
		prefs.CloseBehavior = CloseBehaviorUnset
	}
	if !isValidCloseBehavior(prefs.CloseBehavior) {
		return Preferences{}, fmt.Errorf("invalid close behavior in preferences file: %q", prefs.CloseBehavior)
	}
	prefs.AutoUpdate = applyAutoUpdateDefaults(prefs.AutoUpdate)
	if err := validateAutoUpdate(prefs.AutoUpdate); err != nil {
		return Preferences{}, fmt.Errorf("invalid autoUpdate in preferences file: %w", err)
	}
	return prefs, nil
}

func (s *Service) writeLocked(prefs Preferences) error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return fmt.Errorf("ensure preferences dir: %w", err)
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("encode preferences: %w", err)
	}
	if err := os.WriteFile(s.filePath, data, 0o644); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}
	return nil
}

func defaultPreferences() Preferences {
	return Preferences{
		CloseBehavior: CloseBehaviorUnset,
		AutoUpdate:    defaultAutoUpdate(),
	}
}

func defaultAutoUpdate() AutoUpdate {
	return AutoUpdate{
		Enabled:       true,
		IntervalHours: defaultAutoUpdateInterval,
		Source:        AutoUpdateSourceGitHub,
		GithubRepo:    defaultAutoUpdateGithubRepo,
		GitlabHost:    defaultAutoUpdateGitlabHost,
		GitlabProject: defaultAutoUpdateGitlabProject,
		SkipVersion:   "",
		LastCheckedAt: "",
	}
}

func applyAutoUpdateDefaults(a AutoUpdate) AutoUpdate {
	d := defaultAutoUpdate()
	if a.IntervalHours == 0 {
		a.IntervalHours = d.IntervalHours
	}
	if a.Source == "" {
		a.Source = d.Source
	}
	if a.GithubRepo == "" {
		a.GithubRepo = d.GithubRepo
	}
	if a.GitlabHost == "" {
		a.GitlabHost = d.GitlabHost
	}
	if a.GitlabProject == "" {
		a.GitlabProject = d.GitlabProject
	}
	return a
}

func isValidCloseBehavior(value string) bool {
	switch value {
	case CloseBehaviorUnset, CloseBehaviorExit, CloseBehaviorMinimizeToTray:
		return true
	default:
		return false
	}
}

func isValidAutoUpdateSource(value string) bool {
	return value == AutoUpdateSourceGitHub || value == AutoUpdateSourceGitLab
}

func isValidAutoUpdateIntervalHours(value int) bool {
	for _, allowed := range allowedAutoUpdateIntervalHours {
		if allowed == value {
			return true
		}
	}
	return false
}

func validateRepoSlug(value string) error {
	if value == "" {
		return errors.New("must not be empty")
	}
	if strings.ContainsAny(value, " \t\n") {
		return errors.New("must not contain whitespace")
	}
	if !strings.Contains(value, "/") {
		return errors.New("must contain at least one '/'")
	}
	return nil
}

func validateGitlabHost(value string) error {
	if value == "" {
		return errors.New("must not be empty")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("must be a valid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("must use https scheme, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return errors.New("must contain a host")
	}
	return nil
}

func validateAutoUpdate(a AutoUpdate) error {
	if !isValidAutoUpdateIntervalHours(a.IntervalHours) {
		return fmt.Errorf("intervalHours %d is not in allowed set %v", a.IntervalHours, allowedAutoUpdateIntervalHours)
	}
	if !isValidAutoUpdateSource(a.Source) {
		return fmt.Errorf("source %q must be %q or %q", a.Source, AutoUpdateSourceGitHub, AutoUpdateSourceGitLab)
	}
	if err := validateRepoSlug(a.GithubRepo); err != nil {
		return fmt.Errorf("githubRepo: %w", err)
	}
	if err := validateRepoSlug(a.GitlabProject); err != nil {
		return fmt.Errorf("gitlabProject: %w", err)
	}
	if err := validateGitlabHost(a.GitlabHost); err != nil {
		return fmt.Errorf("gitlabHost: %w", err)
	}
	if a.LastCheckedAt != "" {
		if _, err := time.Parse(time.RFC3339, a.LastCheckedAt); err != nil {
			return fmt.Errorf("lastCheckedAt: must be RFC 3339 or empty: %w", err)
		}
	}
	return nil
}
