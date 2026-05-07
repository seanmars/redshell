package hooks

import (
	"encoding/json"
	"fmt"
	"runtime"
)

// parseCopilotFile flattens the Copilot CLI policy file into the same flat
// Hook list shape used by Claude. Copilot has no matcher concept, so
// Hook.Matcher is left empty. The summary preference follows the OS-native
// script field when both `bash` and `powershell` are populated.
func parseCopilotFile(raw []byte, sourceID string) ([]Hook, error) {
	var doc struct {
		Version int                          `json:"version"`
		Hooks   map[string][]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	out := make([]Hook, 0)
	for _, event := range sortedKeys(doc.Hooks) {
		entries := doc.Hooks[event]
		for handlerIdx, entryRaw := range entries {
			var handler map[string]interface{}
			if err := json.Unmarshal(entryRaw, &handler); err != nil {
				continue
			}
			typeStr, _ := handler["type"].(string)
			summary := copilotSummary(handler)
			id := hookID(sourceID, event, 0, handlerIdx)
			out = append(out, Hook{
				ID:       id,
				SourceID: sourceID,
				Event:    event,
				Type:     typeStr,
				Summary:  summary,
				Raw:      handler,
				DupCount: 1,
			})
		}
	}
	return out, nil
}

// copilotSummary returns the preferred script body. If both bash and
// powershell are set, prefer the OS-native one so the user sees what will
// actually run on their machine.
func copilotSummary(handler map[string]interface{}) string {
	bash := stringField(handler, "bash")
	ps := stringField(handler, "powershell")
	switch {
	case bash != "" && ps != "":
		if runtime.GOOS == "windows" {
			return ps
		}
		return bash
	case bash != "":
		return bash
	case ps != "":
		return ps
	}
	return stringField(handler, "command")
}
