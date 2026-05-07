package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	// scannerInitialBuf is the starting buffer size for bufio.Scanner.
	scannerInitialBuf = 64 * 1024
	// scannerMaxBuf is the per-line ceiling. Claude tool inputs and
	// attachments can be hundreds of KB; 16 MiB tolerates anything we
	// have observed and stops short of pathological lines.
	scannerMaxBuf = 16 * 1024 * 1024

	// titlePeekLines bounds the rich-title walk so we never read the
	// whole file just to find a display name.
	titlePeekLines = 200
	// summaryMaxRunes truncates user/assistant text content for the
	// collapsed summary line.
	summaryMaxRunes = 120
)

// parseEventPage streams the file, returns events [offset, offset+limit),
// and reports total event count and skipped (corrupt) line count for the
// whole file.
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

// normalizeEvent classifies one parsed jsonl object into the shared Event
// shape and applies redaction in place.
func normalizeEvent(index int, obj map[string]interface{}) Event {
	redactClaudeFields(obj)
	kind, subtype := classify(obj)
	summary := makeSummary(kind, obj)
	return Event{
		Index:   index,
		Kind:    kind,
		Subtype: subtype,
		Summary: summary,
		Raw:     obj,
	}
}

// classify maps a Claude jsonl event onto the normalized kind taxonomy.
// Returns (kind, subtype-string).
func classify(obj map[string]interface{}) (string, string) {
	t, _ := obj["type"].(string)
	if isMeta, _ := obj["isMeta"].(bool); isMeta {
		return "meta", t
	}
	switch t {
	case "user":
		// tool_result lives under user.message.content[]; if content is
		// an array this is a tool result.
		if msg, ok := obj["message"].(map[string]interface{}); ok {
			if _, isArr := msg["content"].([]interface{}); isArr {
				return "tool_result", "tool_result"
			}
		}
		return "user", "user"
	case "assistant":
		// Assistant lines carry one content block per line; classify by
		// the first block's type.
		if msg, ok := obj["message"].(map[string]interface{}); ok {
			if arr, ok := msg["content"].([]interface{}); ok && len(arr) > 0 {
				if first, ok := arr[0].(map[string]interface{}); ok {
					blockType, _ := first["type"].(string)
					switch blockType {
					case "tool_use":
						return "tool_use", "tool_use"
					case "thinking":
						return "assistant", "assistant.thinking"
					case "text":
						return "assistant", "assistant.text"
					}
				}
			}
		}
		return "assistant", "assistant"
	case "system":
		sub, _ := obj["subtype"].(string)
		if sub != "" {
			return "system", "system." + sub
		}
		return "system", "system"
	case "attachment":
		if att, ok := obj["attachment"].(map[string]interface{}); ok {
			if at, _ := att["type"].(string); at != "" {
				return "attachment", "attachment." + at
			}
		}
		return "attachment", "attachment"
	case "permission-mode", "last-prompt", "custom-title", "agent-name",
		"file-history-snapshot", "queue-operation":
		return "meta", t
	default:
		if t == "" {
			return "meta", "unknown"
		}
		return "meta", t
	}
}

// makeSummary builds the collapsed display line per kind.
func makeSummary(kind string, obj map[string]interface{}) string {
	switch kind {
	case "user":
		return truncateRunes(extractUserText(obj), summaryMaxRunes)
	case "assistant":
		return truncateRunes(extractAssistantText(obj), summaryMaxRunes)
	case "tool_use":
		return summaryToolUse(obj)
	case "tool_result":
		return summaryToolResult(obj)
	case "system":
		t, _ := obj["type"].(string)
		if sub, _ := obj["subtype"].(string); sub != "" {
			return t + "." + sub
		}
		return t
	case "attachment":
		if att, ok := obj["attachment"].(map[string]interface{}); ok {
			at, _ := att["type"].(string)
			if fn, ok := att["filename"].(string); ok && fn != "" {
				return at + ": " + fn
			}
			return at
		}
		return "attachment"
	case "meta":
		t, _ := obj["type"].(string)
		return t
	}
	return ""
}

func extractUserText(obj map[string]interface{}) string {
	msg, ok := obj["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	if s, ok := msg["content"].(string); ok {
		return s
	}
	return ""
}

func extractAssistantText(obj map[string]interface{}) string {
	msg, ok := obj["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	arr, ok := msg["content"].([]interface{})
	if !ok || len(arr) == 0 {
		return ""
	}
	for _, block := range arr {
		b, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		switch b["type"] {
		case "text":
			if s, ok := b["text"].(string); ok && s != "" {
				return s
			}
		case "thinking":
			return "(thinking)"
		}
	}
	return ""
}

func summaryToolUse(obj map[string]interface{}) string {
	msg, ok := obj["message"].(map[string]interface{})
	if !ok {
		return "tool_use"
	}
	arr, ok := msg["content"].([]interface{})
	if !ok || len(arr) == 0 {
		return "tool_use"
	}
	for _, block := range arr {
		b, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		if b["type"] != "tool_use" {
			continue
		}
		name, _ := b["name"].(string)
		digest := digestArgs(b["input"])
		if digest == "" {
			return name
		}
		return name + " " + digest
	}
	return "tool_use"
}

func summaryToolResult(obj map[string]interface{}) string {
	msg, ok := obj["message"].(map[string]interface{})
	if !ok {
		return "tool_result"
	}
	arr, ok := msg["content"].([]interface{})
	if !ok || len(arr) == 0 {
		return "tool_result"
	}
	for _, block := range arr {
		b, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		if b["type"] != "tool_result" {
			continue
		}
		isErr, _ := b["is_error"].(bool)
		var head string
		if c, ok := b["content"].(string); ok {
			head = truncateRunes(c, summaryMaxRunes)
		}
		if isErr {
			if head == "" {
				return "tool_result (error)"
			}
			return "tool_result (error): " + head
		}
		if head == "" {
			return "tool_result"
		}
		return "tool_result: " + head
	}
	return "tool_result"
}

// digestArgs renders a one-line digest of a tool's input arguments. Long
// values are summarized as "<N chars>"; short keys/values are embedded.
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

// resolveRichTitle walks the file once, prefers the strongest title source
// from the top of the file, and also returns the first cwd it sees.
func resolveRichTitle(path string) (string, string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxBuf)

	var customTitle, agentName, slug, firstUser, cwd string
	lines := 0
	for scanner.Scan() && lines < titlePeekLines {
		lines++
		var obj map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil {
			continue
		}
		if cwd == "" {
			if c, _ := obj["cwd"].(string); c != "" {
				cwd = c
			}
		}
		if slug == "" {
			if s, _ := obj["slug"].(string); s != "" {
				slug = s
			}
		}
		switch obj["type"] {
		case "custom-title":
			if t, _ := obj["customTitle"].(string); t != "" {
				customTitle = t
			}
		case "agent-name":
			if t, _ := obj["agentName"].(string); t != "" {
				agentName = t
			}
		case "user":
			if firstUser != "" {
				continue
			}
			if isMeta, _ := obj["isMeta"].(bool); isMeta {
				continue
			}
			msg, ok := obj["message"].(map[string]interface{})
			if !ok {
				continue
			}
			s, ok := msg["content"].(string)
			if !ok {
				continue
			}
			if isInjectedUserContent(s) {
				continue
			}
			firstUser = s
		}
	}

	switch {
	case customTitle != "":
		return customTitle, cwd
	case agentName != "":
		return agentName, cwd
	case firstUser != "":
		return truncateRunes(firstUser, 80), cwd
	case slug != "":
		return slug, cwd
	}
	return "", cwd
}

// peekFirstCwd reads up to maxLines and returns the first non-empty cwd.
// Used during ListSessions to show a per-group decoded cwd cheaply.
func peekFirstCwd(path string, maxLines int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxBuf)
	for i := 0; i < maxLines && scanner.Scan(); i++ {
		var obj map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil {
			continue
		}
		if c, _ := obj["cwd"].(string); c != "" {
			return c
		}
	}
	return ""
}

// isInjectedUserContent recognizes the system-injected user-role messages
// described in the analysis doc (caveats, slash-command echoes, hook
// outputs) so the rich-title resolver skips past them.
func isInjectedUserContent(s string) bool {
	prefixes := []string{
		"<local-command-",
		"<command-",
		"<system-reminder>",
		"Caveat:",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// redactClaudeFields walks an assistant.message.content[] array and replaces
// each block's `thinking` and `signature` fields with a sentinel marker.
func redactClaudeFields(obj map[string]interface{}) {
	msg, ok := obj["message"].(map[string]interface{})
	if !ok {
		return
	}
	arr, ok := msg["content"].([]interface{})
	if !ok {
		return
	}
	for _, block := range arr {
		b, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := b["type"].(string); t != "thinking" {
			continue
		}
		if v, ok := b["thinking"].(string); ok {
			b["thinking"] = redactedSentinel("thinking", len(v))
		}
		if v, ok := b["signature"].(string); ok {
			b["signature"] = redactedSentinel("signature", len(v))
		}
	}
}

func redactedSentinel(field string, size int) map[string]interface{} {
	return map[string]interface{}{
		"_redacted": field,
		"_size":     size,
	}
}

func truncateRunes(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
