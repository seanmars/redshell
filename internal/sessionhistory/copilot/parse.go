package copilot

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	scannerInitialBuf = 64 * 1024
	scannerMaxBuf     = 16 * 1024 * 1024
	titlePeekLines    = 200
	summaryMaxRunes   = 120
)

// parseEventPage streams events.jsonl, returns events [offset, offset+limit)
// and reports total / skippedLines for the whole file.
func parseEventPage(path string, offset, limit int) (EventPage, error) {
	f, err := os.Open(path)
	if err != nil {
		return EventPage{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxBuf)

	page := EventPage{Events: make([]Event, 0, limit)}
	idx := 0
	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			page.SkippedLines++
			continue
		}
		if idx >= offset && len(page.Events) < limit {
			page.Events = append(page.Events, normalizeEvent(idx, obj))
		}
		idx++
	}
	if err := scanner.Err(); err != nil {
		return EventPage{}, err
	}
	page.Total = idx
	page.HasMore = (offset + limit) < idx
	return page, nil
}

// normalizeEvent classifies one parsed jsonl object and applies redaction.
func normalizeEvent(index int, obj map[string]interface{}) Event {
	redactCopilotFields(obj)
	kind, subtype := classify(obj)
	summary := makeSummary(kind, obj)
	children := extractAttachmentChildren(kind, obj)
	return Event{
		Index:    index,
		Kind:     kind,
		Subtype:  subtype,
		Summary:  summary,
		Raw:      obj,
		Children: children,
	}
}

// classify maps a Copilot event onto the normalized kind taxonomy.
func classify(obj map[string]interface{}) (string, string) {
	t, _ := obj["type"].(string)
	switch t {
	case "user.message":
		return "user", t
	case "assistant.message", "assistant.reasoning",
		"assistant.turn_start", "assistant.turn_end":
		return "assistant", t
	case "tool.execution_start", "tool.user_requested":
		return "tool_use", t
	case "tool.execution_complete":
		return "tool_result", t
	case "system.message":
		return "system", t
	case "abort":
		return "system", t
	case "skill.invoked":
		return "system", t
	default:
		if strings.HasPrefix(t, "session.") {
			return "system", t
		}
		if t == "" {
			return "meta", "unknown"
		}
		return "system", t
	}
}

// makeSummary produces the collapsed display line per kind. Per the
// analysis doc, user.message must show data.content (not transformedContent).
func makeSummary(kind string, obj map[string]interface{}) string {
	data, _ := obj["data"].(map[string]interface{})
	t, _ := obj["type"].(string)

	switch kind {
	case "user":
		if data != nil {
			if s, _ := data["content"].(string); s != "" {
				return truncateRunes(s, summaryMaxRunes)
			}
		}
		return "user.message"
	case "assistant":
		if data != nil {
			if s, _ := data["content"].(string); s != "" {
				return truncateRunes(s, summaryMaxRunes)
			}
			if s, _ := data["reasoningText"].(string); s != "" {
				return truncateRunes(s, summaryMaxRunes)
			}
			// reasoning event uses `content` directly (no `text` blocks)
			// already handled above; turn_start / turn_end carry no text.
			if t == "assistant.reasoning" {
				return "(reasoning)"
			}
			if t == "assistant.turn_start" {
				return "turn start"
			}
			if t == "assistant.turn_end" {
				return "turn end"
			}
		}
		return t
	case "tool_use":
		if data != nil {
			tool, _ := data["toolName"].(string)
			digest := digestArgs(data["arguments"])
			if tool == "" {
				return t
			}
			if digest == "" {
				return tool
			}
			return tool + " " + digest
		}
		return t
	case "tool_result":
		if data != nil {
			success, _ := data["success"].(bool)
			if success {
				return "tool_result (ok)"
			}
			if errMsg, _ := data["error"].(string); errMsg != "" {
				return "tool_result (error): " + truncateRunes(errMsg, 80)
			}
			return "tool_result (error)"
		}
		return t
	case "system":
		return t
	}
	return t
}

// extractAttachmentChildren turns user.message.data.attachments[] entries
// into child Events with kind "attachment", per design Decision 8.
func extractAttachmentChildren(kind string, obj map[string]interface{}) []Event {
	if kind != "user" {
		return nil
	}
	data, _ := obj["data"].(map[string]interface{})
	if data == nil {
		return nil
	}
	atts, ok := data["attachments"].([]interface{})
	if !ok || len(atts) == 0 {
		return nil
	}
	out := make([]Event, 0, len(atts))
	for i, raw := range atts {
		att, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		at, _ := att["type"].(string)
		dn, _ := att["displayName"].(string)
		summary := at
		if dn != "" {
			summary = at + ": " + dn
		}
		out = append(out, Event{
			Index:   i,
			Kind:    "attachment",
			Subtype: "attachment." + at,
			Summary: summary,
			Raw:     att,
		})
	}
	return out
}

// peekFirstUserContent finds the first user.message and returns its
// data.content, skipping system / assistant rows that may appear first.
func peekFirstUserContent(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxBuf)
	for i := 0; i < titlePeekLines && scanner.Scan(); i++ {
		var obj map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil {
			continue
		}
		if obj["type"] != "user.message" {
			continue
		}
		data, ok := obj["data"].(map[string]interface{})
		if !ok {
			continue
		}
		if s, _ := data["content"].(string); s != "" {
			return truncateRunes(s, 80)
		}
	}
	return ""
}

// redactCopilotFields strips data.encryptedContent and data.reasoningOpaque
// from the parsed event in place, leaving a sentinel marker.
func redactCopilotFields(obj map[string]interface{}) {
	data, ok := obj["data"].(map[string]interface{})
	if !ok {
		return
	}
	for _, field := range []string{"encryptedContent", "reasoningOpaque"} {
		v, ok := data[field].(string)
		if !ok {
			continue
		}
		data[field] = redactedSentinel(field, len(v))
	}
}

func redactedSentinel(field string, size int) map[string]interface{} {
	return map[string]interface{}{
		"_redacted": field,
		"_size":     size,
	}
}

func digestArgs(v interface{}) string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(m))
	for _, k := range keys {
		val := m[k]
		var rendered string
		switch tv := val.(type) {
		case string:
			if len(tv) > 60 {
				rendered = "<" + strconv.Itoa(len(tv)) + " chars>"
			} else {
				rendered = tv
			}
		case bool:
			rendered = strconv.FormatBool(tv)
		case float64:
			if tv == float64(int64(tv)) {
				rendered = strconv.FormatInt(int64(tv), 10)
			} else {
				rendered = strconv.FormatFloat(tv, 'f', -1, 64)
			}
		default:
			rendered = "<...>"
		}
		parts = append(parts, k+"="+rendered)
		if len(parts) >= 3 {
			break
		}
	}
	return strings.Join(parts, " ")
}

func truncateRunes(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
