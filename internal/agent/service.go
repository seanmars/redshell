package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"redshell/internal/sysproc"
)

type Agent struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	ConfigDir    string `json:"configDir"`
	SettingsFile string `json:"settingsFile"`
	Version      string `json:"version"`
	Configured   bool   `json:"configured"`
}

type execRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type Service struct {
	run execRunner
}

func NewService() *Service {
	return &Service{run: defaultRunner}
}

func NewServiceWithRunner(run execRunner) *Service {
	return &Service{run: run}
}

func defaultRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = sysproc.Hidden()
	return cmd.CombinedOutput()
}

type agentSpec struct {
	id           string
	label        string
	configDir    string
	settingsFile string
	dotfileDir   string
	bin          string
}

func agentSpecs() []agentSpec {
	home, _ := os.UserHomeDir()
	return []agentSpec{
		{
			id:           "claude",
			label:        "Claude Code",
			configDir:    "~/.claude",
			settingsFile: "~/.claude/settings.json",
			dotfileDir:   filepath.Join(home, ".claude"),
			bin:          "claude",
		},
		{
			id:           "copilot",
			label:        "GitHub Copilot",
			configDir:    "~/.copilot",
			settingsFile: "~/.copilot/config.json",
			dotfileDir:   filepath.Join(home, ".copilot"),
			bin:          "copilot",
		},
	}
}

func (s *Service) ListAgents() []Agent {
	specs := agentSpecs()
	out := make([]Agent, len(specs))
	versions := make([]string, len(specs))

	var wg sync.WaitGroup
	for i, spec := range specs {
		wg.Add(1)
		go func(i int, spec agentSpec) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			versions[i] = s.probeVersion(ctx, spec.bin)
		}(i, spec)
	}
	wg.Wait()

	for i, spec := range specs {
		out[i] = Agent{
			ID:           spec.id,
			Label:        spec.label,
			ConfigDir:    spec.configDir,
			SettingsFile: spec.settingsFile,
			Version:      versions[i],
			Configured:   dirExists(spec.dotfileDir),
		}
	}
	return out
}

var versionRe = regexp.MustCompile(`\b(\d+\.\d+\.\d+)\b`)

func (s *Service) probeVersion(ctx context.Context, bin string) string {
	out, err := s.run(ctx, bin, "--version")
	if err != nil && len(out) == 0 {
		return ""
	}
	match := versionRe.FindSubmatch(out)
	if len(match) < 2 {
		return ""
	}
	return string(match[1])
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
