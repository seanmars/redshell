package hooks

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// claudeParseResult is the per-file output of the Claude parser. It is
// intentionally decoupled from Source so the same parser can serve user,
// local, and plugin sources.
type claudeParseResult struct {
	Hooks        []Hook
	DisableAll   bool
	UnknownTypes int // counted but not surfaced; tolerance is the requirement
}

// parseClaudeFile parses one Claude-shape settings file and returns flat
// Hook entries plus a disableAllHooks flag. raw can be empty (caller has
// already decided the file should be loaded). The parser tolerates unknown
// fields and unknown handler types, preserving the original entry as Raw.
func parseClaudeFile(raw []byte, sourceID string) (claudeParseResult, error) {
	var doc struct {
		DisableAllHooks bool                         `json:"disableAllHooks"`
		Hooks           map[string][]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return claudeParseResult{}, fmt.Errorf("parse: %w", err)
	}

	out := claudeParseResult{DisableAll: doc.DisableAllHooks}
	events := sortedKeys(doc.Hooks)
	for _, event := range events {
		groups := doc.Hooks[event]
		for groupIdx, groupRaw := range groups {
			group, err := decodeMatcherGroup(groupRaw)
			if err != nil {
				continue
			}
			for handlerIdx, handlerRaw := range group.Hooks {
				var handler map[string]interface{}
				if err := json.Unmarshal(handlerRaw, &handler); err != nil {
					continue
				}
				typeStr, _ := handler["type"].(string)
				summary := claudeSummary(typeStr, handler)
				id := hookID(sourceID, event, groupIdx, handlerIdx)
				out.Hooks = append(out.Hooks, Hook{
					ID:       id,
					SourceID: sourceID,
					Event:    event,
					Matcher:  group.Matcher,
					Type:     typeStr,
					Summary:  summary,
					Raw:      handler,
					DupCount: 1,
				})
			}
		}
	}
	return out, nil
}

type matcherGroup struct {
	Matcher string            `json:"matcher"`
	Hooks   []json.RawMessage `json:"hooks"`
}

func decodeMatcherGroup(raw json.RawMessage) (matcherGroup, error) {
	var g matcherGroup
	if err := json.Unmarshal(raw, &g); err != nil {
		return matcherGroup{}, err
	}
	return g, nil
}

// claudeSummary returns a short single-line digest for a handler entry.
// command -> command field, http -> url, mcp_tool -> server:tool,
// prompt/agent -> prompt. Unknown types fall back to the raw type label.
func claudeSummary(typeStr string, handler map[string]interface{}) string {
	switch typeStr {
	case "command":
		return stringField(handler, "command")
	case "http":
		return stringField(handler, "url")
	case "mcp_tool":
		server := stringField(handler, "server")
		tool := stringField(handler, "tool")
		if server != "" && tool != "" {
			return server + ":" + tool
		}
		if tool != "" {
			return tool
		}
		return server
	case "prompt", "agent":
		return stringField(handler, "prompt")
	}
	return ""
}

func stringField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// dedupKey returns a normalized string used to count cross-source
// duplicates. Same-typed handlers with the same identifying field collide;
// different types never collide. Empty fields produce empty keys, which
// the caller treats as "no dedup signal".
func dedupKey(h Hook) string {
	switch h.Type {
	case "command", "http", "prompt", "agent":
		return h.Type + ":" + h.Summary
	case "mcp_tool":
		return "mcp_tool:" + h.Summary
	}
	if h.Type == "" {
		return ""
	}
	return h.Type + ":" + h.Summary
}

func sortedKeys(m map[string][]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// hookID is a stable, frontend-selection-safe identifier derived from the
// owning source and the entry's positional coordinates. It is independent
// of the handler's contents so reordering content does not invalidate the
// frontend's selection state mid-fetch.
func hookID(sourceID, event string, groupIdx, handlerIdx int) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s|%s|%d|%d", sourceID, event, groupIdx, handlerIdx)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// summaryFirstLine collapses multiline summaries to one line. The view
// already truncates visually, but newlines in tooltips look broken.
func summaryFirstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
