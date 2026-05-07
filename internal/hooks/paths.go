package hooks

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrPluginPathRejected is returned by ResolvePluginHookPath when the
// caller-supplied installPath crosses a .git/hooks/ segment, which the
// scanner treats as a defensive guard against accidentally loading
// non-Claude git hook scripts.
var ErrPluginPathRejected = errors.New("plugin hook path rejected")

const (
	claudeUserSettings  = ".claude/settings.json"
	claudeLocalSettings = ".claude/settings.local.json"
	claudeInstalled     = ".claude/plugins/installed_plugins.json"
	pluginHooksSuffix   = "hooks/hooks.json"

	copilotPolicyRel = ".github/hooks/copilot-cli-policy.json"
)

// ClaudeUserSettingsPath returns the absolute path to the user-level
// Claude settings file under the supplied home directory.
func ClaudeUserSettingsPath(home string) string {
	return filepath.Join(home, filepath.FromSlash(claudeUserSettings))
}

// ClaudeLocalSettingsPath returns the absolute path to the local-scope
// Claude settings file under the supplied home directory.
func ClaudeLocalSettingsPath(home string) string {
	return filepath.Join(home, filepath.FromSlash(claudeLocalSettings))
}

// ClaudeInstalledPluginsPath returns the absolute path to the plugin
// install registry maintained by Claude Code.
func ClaudeInstalledPluginsPath(home string) string {
	return filepath.Join(home, filepath.FromSlash(claudeInstalled))
}

// ResolvePluginHookPath joins installPath with the canonical
// "hooks/hooks.json" suffix and refuses any path that crosses a
// `.git/hooks/` segment. installPath is expected to be the absolute path
// recorded in installed_plugins.json's `installPath` field.
func ResolvePluginHookPath(installPath string) (string, error) {
	if installPath == "" {
		return "", ErrPluginPathRejected
	}
	cleaned := filepath.Clean(installPath)
	if containsGitHooks(cleaned) {
		return "", ErrPluginPathRejected
	}
	return filepath.Join(cleaned, filepath.FromSlash(pluginHooksSuffix)), nil
}

// CopilotPolicyPath returns the absolute path to the Copilot CLI hook
// policy file under the supplied workspace directory. v1 callers pass an
// empty workspace and never invoke this; the function is shipped so the
// future B-route only needs to wire it.
func CopilotPolicyPath(workspace string) string {
	if workspace == "" {
		return ""
	}
	return filepath.Join(workspace, filepath.FromSlash(copilotPolicyRel))
}

// containsGitHooks reports whether p has a `.git/hooks` segment somewhere
// in its directory chain. The scanner uses this as a defensive guard so a
// pathological installPath cannot lure us into loading git's own hook
// scripts as Claude hooks.
func containsGitHooks(p string) bool {
	parts := strings.Split(filepath.ToSlash(p), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == ".git" && parts[i+1] == "hooks" {
			return true
		}
	}
	return false
}
