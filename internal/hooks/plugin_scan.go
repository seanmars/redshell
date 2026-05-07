package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// installedPluginsFile mirrors the v2 schema of
// ~/.claude/plugins/installed_plugins.json:
//
//	{
//	  "version": 2,
//	  "plugins": {
//	    "<pluginID>@<marketplaceID>": [
//	      { "scope": "user", "installPath": "...", "version": "..." },
//	      ...
//	    ],
//	    ...
//	  }
//	}
//
// We tolerate v1 (without the array wrapper) by silently ignoring entries
// that don't decode as the array form.
type installedPluginsFile struct {
	Version int                               `json:"version"`
	Plugins map[string][]installedPluginEntry `json:"plugins"`
}

type installedPluginEntry struct {
	Scope       string `json:"scope"`
	InstallPath string `json:"installPath"`
	Version     string `json:"version"`
}

// pluginCandidate is the per-entry result of scanning installed_plugins.json.
// HookFile is the resolved absolute path to the entry's hooks.json. Label
// is the user-visible source label produced by formatPluginLabel.
type pluginCandidate struct {
	Key         string
	PluginID    string
	Marketplace string
	Scope       string
	InstallPath string
	HookFile    string
	Label       string
}

// scanPluginCandidates reads installed_plugins.json under the supplied
// home directory and returns one pluginCandidate per entry. Entries whose
// installPath is empty or whose resolved hook path is rejected by
// ResolvePluginHookPath are skipped. The function does NOT stat the hook
// file — the caller decides whether to load it. Sort order is alphabetic
// by Label (case-insensitive); entries within the same key are
// stable-ordered by scope so the label suffix is deterministic.
func scanPluginCandidates(home string) ([]pluginCandidate, error) {
	data, err := os.ReadFile(ClaudeInstalledPluginsPath(home))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var doc installedPluginsFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse installed_plugins.json: %w", err)
	}

	var out []pluginCandidate
	for key, entries := range doc.Plugins {
		pluginID, marketplace := splitPluginKey(key)
		multi := len(entries) > 1
		for _, e := range entries {
			if e.InstallPath == "" {
				continue
			}
			hookFile, err := ResolvePluginHookPath(e.InstallPath)
			if err != nil {
				continue
			}
			out = append(out, pluginCandidate{
				Key:         key,
				PluginID:    pluginID,
				Marketplace: marketplace,
				Scope:       e.Scope,
				InstallPath: e.InstallPath,
				HookFile:    hookFile,
				Label:       formatPluginLabel(pluginID, marketplace, e.Scope, multi),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		li := strings.ToLower(out[i].Label)
		lj := strings.ToLower(out[j].Label)
		return li < lj
	})
	return out, nil
}

// splitPluginKey splits "<pluginID>@<marketplaceID>" into its halves on
// the LAST "@" so plugin ids that contain "@" themselves still resolve a
// sensible marketplace label. Keys without "@" fall back to the whole
// string as pluginID and an empty marketplace.
func splitPluginKey(key string) (pluginID, marketplace string) {
	if i := strings.LastIndex(key, "@"); i >= 0 {
		return key[:i], key[i+1:]
	}
	return key, ""
}

// formatPluginLabel produces the human-facing source label. When multiple
// entries share the same key, append the scope to disambiguate.
func formatPluginLabel(pluginID, marketplace, scope string, multi bool) string {
	base := "Plugin: " + pluginID
	if marketplace != "" {
		base += "@" + marketplace
	}
	if multi && scope != "" {
		base += " (" + scope + ")"
	}
	return base
}

// readPluginHookFile loads and parses a plugin's hook file. Returns
// (parseResult, ok=true) when the file exists and parses; (zero, false,
// nil) when the file is missing; (zero, false, err) on parse errors.
func readPluginHookFile(hookFile, sourceID string) (claudeParseResult, bool, error) {
	data, err := os.ReadFile(hookFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return claudeParseResult{}, false, nil
		}
		return claudeParseResult{}, false, err
	}
	res, err := parseClaudeFile(data, sourceID)
	if err != nil {
		return claudeParseResult{}, false, err
	}
	return res, true, nil
}
