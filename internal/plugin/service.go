package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"redshell/internal/agent"
	"redshell/internal/marketplace"
	"redshell/internal/sysproc"
)

type MarketplacePlugin struct {
	Name            string `json:"name"`
	Project         string `json:"project"`
	Marketplace     string `json:"marketplace"`
	MarketplaceName string `json:"marketplaceName"`
	InstallName     string `json:"installName"`
	Description     string `json:"description,omitempty"`
	Agent           string `json:"agent"`
}

type InstalledPlugin struct {
	DisplayName     string `json:"displayName"`
	UninstallName   string `json:"uninstallName"`
	Agent           string `json:"agent"`
	MarketplaceName string `json:"marketplaceName"`
}

type FetchAllResult struct {
	Plugins []MarketplacePlugin `json:"plugins"`
	Errors  []string            `json:"errors"`
}

type AgentUpdateOutcome struct {
	AgentID string `json:"agentId"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

type UpdateAgentMarketplacesResult struct {
	Outcomes []AgentUpdateOutcome `json:"outcomes"`
}

type manifestPlugin struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type marketplaceManifest struct {
	Name    string           `json:"name"`
	Plugins []manifestPlugin `json:"plugins"`
}

type Service struct {
	marketplaceSvc *marketplace.Service
	agentSvc       *agent.Service
	settingsSvc    *agent.SettingsService
}

func NewService(mktSvc *marketplace.Service, agtSvc *agent.Service, settingsSvc *agent.SettingsService) *Service {
	return &Service{
		marketplaceSvc: mktSvc,
		agentSvc:       agtSvc,
		settingsSvc:    settingsSvc,
	}
}

func (s *Service) FetchAll() FetchAllResult {
	marketplaces, err := s.marketplaceSvc.List()
	if err != nil {
		return FetchAllResult{Errors: []string{err.Error()}}
	}

	enabledAgents, err := s.enabledAgents()
	if err != nil {
		return FetchAllResult{Errors: []string{err.Error()}}
	}

	var plugins []MarketplacePlugin
	var errs []string

	for _, m := range marketplaces {
		for _, agentID := range enabledAgents {
			items, err := s.fetchForAgent(m, agentID)
			if err != nil {
				errs = append(errs, fmt.Sprintf("[%s/%s] %s", m.ID, agentID, err.Error()))
				continue
			}
			plugins = append(plugins, items...)
		}
	}

	return FetchAllResult{Plugins: plugins, Errors: errs}
}

func (s *Service) fetchForAgent(m marketplace.Marketplace, agentID string) ([]MarketplacePlugin, error) {
	manifestPath, ok := marketplace.AgentMarketplaceFiles[agentID]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", agentID)
	}

	cachePath := filepath.Join(s.marketplaceSvc.CacheDir(m.ID), filepath.FromSlash(manifestPath))
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("cache missing; click Refresh to re-clone")
		}
		return nil, err
	}

	var manifest marketplaceManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("manifest parse error: %w", err)
	}

	marketplaceName := manifest.Name
	if marketplaceName == "" {
		marketplaceName = m.ID
		if m.Name != nil {
			if n, ok := m.Name[agentID]; ok && n != "" {
				marketplaceName = n
			}
		}
	}

	parsed, _ := parseGitURL(m.URL)
	result := make([]MarketplacePlugin, 0, len(manifest.Plugins))
	for _, p := range manifest.Plugins {
		if p.Name == "" || p.Source == "" {
			continue
		}
		result = append(result, MarketplacePlugin{
			Name:            p.Name,
			Project:         parsed.repo,
			Marketplace:     m.ID,
			MarketplaceName: marketplaceName,
			InstallName:     p.Name + "@" + marketplaceName,
			Description:     p.Description,
			Agent:           agentID,
		})
	}
	return result, nil
}

func (s *Service) EnsureMarketplace(marketplaceURL, agentID string) error {
	home, _ := os.UserHomeDir()
	var configPath string
	switch agentID {
	case "claude":
		configPath = filepath.Join(home, ".claude", "plugins", "known_marketplaces.json")
	case "copilot":
		configPath = filepath.Join(home, ".copilot", "config.json")
	}

	if registered := isMarketplaceRegistered(configPath, marketplaceURL, agentID); registered {
		return nil
	}

	gitURL := strings.TrimSuffix(marketplaceURL, ".git") + ".git"
	err := runAgentCmd(agentID, []string{"plugin", "marketplace", "add", gitURL})
	if err != nil && strings.Contains(err.Error(), "already registered") {
		return nil
	}
	return err
}

func (s *Service) Install(agentID string, plugins []MarketplacePlugin, logFn func(string)) error {
	if err := s.ensureAgentEnabled(agentID); err != nil {
		return err
	}

	marketplaces, err := s.marketplaceSvc.List()
	if err != nil {
		return fmt.Errorf("failed to load marketplaces: %w", err)
	}
	mktByID := make(map[string]marketplace.Marketplace, len(marketplaces))
	for _, m := range marketplaces {
		mktByID[m.ID] = m
	}

	for _, p := range plugins {
		m, ok := mktByID[p.Marketplace]
		if !ok {
			return fmt.Errorf("marketplace not found: %s", p.Marketplace)
		}
		logFn(fmt.Sprintf("Ensuring marketplace: %s", m.URL))
		if err := s.EnsureMarketplace(m.URL, agentID); err != nil {
			return fmt.Errorf("failed to register marketplace %s: %w", p.Marketplace, err)
		}
		logFn(fmt.Sprintf("Installing: %s", p.InstallName))
		if err := runAgentCmd(agentID, []string{"plugin", "install", p.InstallName}); err != nil {
			return fmt.Errorf("failed to install %s: %w", p.InstallName, err)
		}
		logFn(fmt.Sprintf("Installed: %s", p.Name))
	}
	return nil
}

func (s *Service) ListInstalled(agentID string) ([]InstalledPlugin, error) {
	if err := s.ensureAgentEnabled(agentID); err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()
	switch agentID {
	case "claude":
		return readClaudeInstalled(home)
	case "copilot":
		return readCopilotInstalled(home)
	default:
		return nil, fmt.Errorf("unknown agent: %s", agentID)
	}
}

func (s *Service) Uninstall(agentID, pluginID string) error {
	if err := s.ensureAgentEnabled(agentID); err != nil {
		return err
	}
	return runAgentCmd(agentID, []string{"plugin", "uninstall", pluginID})
}

func (s *Service) UpdateAgentMarketplace(agentID string, logFn func(string)) AgentUpdateOutcome {
	if logFn == nil {
		logFn = func(string) {}
	}
	if err := s.ensureAgentEnabled(agentID); err != nil {
		msg := "[" + agentID + "] " + err.Error()
		logFn(msg)
		return AgentUpdateOutcome{AgentID: agentID, OK: false, Error: msg}
	}

	prefix := "[" + agentID + "] "
	stream := func(line string) { logFn(prefix + line) }
	runErr := runAgentCmdStreaming(agentID, []string{"plugin", "marketplace", "update"}, stream)
	outcome := AgentUpdateOutcome{AgentID: agentID, OK: runErr == nil}
	if runErr != nil {
		outcome.Error = prefix + runErr.Error()
		logFn(outcome.Error)
	}
	return outcome
}

func (s *Service) UpdateAgentMarketplaces(logFn func(string)) UpdateAgentMarketplacesResult {
	agents, err := s.enabledAgents()
	if err != nil {
		return UpdateAgentMarketplacesResult{
			Outcomes: []AgentUpdateOutcome{{
				AgentID: "",
				OK:      false,
				Error:   err.Error(),
			}},
		}
	}

	outcomes := make([]AgentUpdateOutcome, 0, len(agents))
	for _, agentID := range agents {
		outcomes = append(outcomes, s.UpdateAgentMarketplace(agentID, logFn))
	}
	return UpdateAgentMarketplacesResult{Outcomes: outcomes}
}

func (s *Service) enabledAgents() ([]string, error) {
	if s.settingsSvc == nil {
		return agent.SupportedAgentIDs(), nil
	}
	return s.settingsSvc.GetEnabledAgents()
}

func (s *Service) ensureAgentEnabled(agentID string) error {
	if s.settingsSvc == nil {
		if !containsAgent(agent.SupportedAgentIDs(), agentID) {
			return fmt.Errorf("unknown agent: %s", agentID)
		}
		return nil
	}

	enabled, err := s.settingsSvc.IsAgentEnabled(agentID)
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("agent is disabled: %s", agentID)
	}
	return nil
}

func containsAgent(agentIDs []string, target string) bool {
	for _, agentID := range agentIDs {
		if agentID == target {
			return true
		}
	}
	return false
}

func runAgentCmd(agentID string, args []string) error {
	cmd := exec.Command(agentID, args...)
	cmd.SysProcAttr = sysproc.Hidden()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("agent CLI '%s' is not installed: please install it first", agentID)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s %s: %s", agentID, strings.Join(args, " "), msg)
	}
	return nil
}

var runAgentCmdStreaming = realRunAgentCmdStreaming

func realRunAgentCmdStreaming(agentID string, args []string, stdoutFn func(string)) error {
	cmd := exec.Command(agentID, args...)
	cmd.SysProcAttr = sysproc.Hidden()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout := &lineWriter{fn: stdoutFn}
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		stdout.flush()
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("agent CLI '%s' is not installed: please install it first", agentID)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s %s: %s", agentID, strings.Join(args, " "), msg)
	}
	stdout.flush()
	return nil
}

type lineWriter struct {
	buf bytes.Buffer
	fn  func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		raw := w.buf.Bytes()
		idx := bytes.IndexByte(raw, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(raw[:idx]), "\r")
		if w.fn != nil {
			w.fn(line)
		}
		w.buf.Next(idx + 1)
	}
	return len(p), nil
}

func (w *lineWriter) flush() {
	if w.buf.Len() == 0 {
		return
	}
	line := strings.TrimRight(w.buf.String(), "\r")
	w.buf.Reset()
	if w.fn != nil {
		w.fn(line)
	}
}

func normalizeURL(u string) string {
	return strings.TrimSuffix(u, ".git")
}

func isMarketplaceRegistered(configPath, marketplaceURL, agentID string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	want := normalizeURL(marketplaceURL)
	switch agentID {
	case "claude":
		var cfg map[string]struct {
			Source struct {
				URL string `json:"url"`
			} `json:"source"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return false
		}
		for _, entry := range cfg {
			if normalizeURL(entry.Source.URL) == want {
				return true
			}
		}
	case "copilot":
		var cfg struct {
			Marketplaces map[string]struct {
				Source struct {
					URL string `json:"url"`
				} `json:"source"`
			} `json:"marketplaces"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return false
		}
		for _, entry := range cfg.Marketplaces {
			if normalizeURL(entry.Source.URL) == want {
				return true
			}
		}
	}
	return false
}

func readClaudeInstalled(home string) ([]InstalledPlugin, error) {
	path := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []InstalledPlugin{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Plugins map[string]json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	result := make([]InstalledPlugin, 0, len(cfg.Plugins))
	for key := range cfg.Plugins {
		var mktName string
		if idx := strings.LastIndex(key, "@"); idx >= 0 {
			mktName = key[idx+1:]
		}
		result = append(result, InstalledPlugin{
			DisplayName:     key,
			UninstallName:   key,
			Agent:           "claude",
			MarketplaceName: mktName,
		})
	}
	return result, nil
}

func readCopilotInstalled(home string) ([]InstalledPlugin, error) {
	path := filepath.Join(home, ".copilot", "config.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []InstalledPlugin{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg struct {
		InstalledPlugins []struct {
			Name        string `json:"name"`
			Marketplace string `json:"marketplace"`
		} `json:"installed_plugins"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	result := make([]InstalledPlugin, 0, len(cfg.InstalledPlugins))
	for _, p := range cfg.InstalledPlugins {
		display := p.Name + "@" + p.Marketplace
		result = append(result, InstalledPlugin{
			DisplayName:     display,
			UninstallName:   display,
			Agent:           "copilot",
			MarketplaceName: p.Marketplace,
		})
	}
	return result, nil
}

type parsedURL struct {
	hostname string
	owner    string
	repo     string
}

func parseGitURL(rawURL string) (parsedURL, error) {
	rawURL = strings.TrimSuffix(rawURL, ".git")
	u, err := url.Parse(rawURL)
	if err != nil {
		return parsedURL{}, fmt.Errorf("invalid URL: %s", rawURL)
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 2 {
		return parsedURL{}, fmt.Errorf("URL must contain owner/repo: %s", rawURL)
	}
	repo := parts[len(parts)-1]
	owner := strings.Join(parts[:len(parts)-1], "/")
	return parsedURL{hostname: u.Hostname(), owner: owner, repo: repo}, nil
}
