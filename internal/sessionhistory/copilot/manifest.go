// Package copilot reads GitHub Copilot CLI session directories under
// ~/.copilot/session-state/<sessionID>/. Each session is a directory with
// a workspace.yaml manifest plus an optional events.jsonl, plan.md,
// session.db, and other artifacts; this reader only consults the manifest
// and events.jsonl per the design's non-goals list.
package copilot

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Workspace mirrors the workspace.yaml manifest. All fields are optional;
// the analysis doc shows minimum-shape sessions with only id /
// summary_count / created_at / updated_at, so callers must tolerate any
// subset.
type Workspace struct {
	ID           string `yaml:"id"`
	Cwd          string `yaml:"cwd"`
	GitRoot      string `yaml:"git_root"`
	Repository   string `yaml:"repository"`
	HostType     string `yaml:"host_type"`
	Branch       string `yaml:"branch"`
	Summary      string `yaml:"summary"`
	SummaryCount int    `yaml:"summary_count"`
	CreatedAt    string `yaml:"created_at"`
	UpdatedAt    string `yaml:"updated_at"`
}

// readWorkspace parses workspace.yaml at sessionDir. Returns a zero
// Workspace if the file is missing.
func readWorkspace(sessionDir string) (Workspace, error) {
	data, err := os.ReadFile(filepath.Join(sessionDir, "workspace.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return Workspace{}, nil
		}
		return Workspace{}, err
	}
	var ws Workspace
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return Workspace{}, err
	}
	return ws, nil
}
