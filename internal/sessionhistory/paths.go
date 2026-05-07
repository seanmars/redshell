package sessionhistory

import (
	"os"
	"path/filepath"
)

// AgentSessionRoots is the single source of truth for per-agent session
// directory locations relative to the user's home directory. Adding a new
// agent only requires a new entry here plus a reader package implementing
// the Reader interface.
var AgentSessionRoots = map[string]string{
	"claude":  filepath.Join(".claude", "projects"),
	"copilot": filepath.Join(".copilot", "session-state"),
}

// resolveProductionRoots expands the relative paths in AgentSessionRoots
// against the current user's home directory. Returns the map keyed by
// agentID with absolute paths.
func resolveProductionRoots() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(AgentSessionRoots))
	for agentID, rel := range AgentSessionRoots {
		out[agentID] = filepath.Join(home, rel)
	}
	return out, nil
}
