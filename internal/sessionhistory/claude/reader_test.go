package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeJSONL writes the given lines (each already a JSON-encodable value)
// as a JSONL file at path.
func writeJSONL(t *testing.T, path string, lines []map[string]interface{}) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	var buf strings.Builder
	for _, line := range lines {
		b, err := json.Marshal(line)
		if err != nil {
			t.Fatalf("marshal line: %v", err)
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeRaw(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestListSessions_GroupsByEncodedDir(t *testing.T) {
	root := t.TempDir()
	writeJSONL(t, filepath.Join(root, "F--ws-foo", "aaa.jsonl"), []map[string]interface{}{
		{"type": "user", "cwd": "F:\\ws\\foo", "message": map[string]interface{}{"role": "user", "content": "hi"}},
	})
	writeJSONL(t, filepath.Join(root, "F--ws-foo", "bbb.jsonl"), []map[string]interface{}{
		{"type": "user", "cwd": "F:\\ws\\foo", "message": map[string]interface{}{"role": "user", "content": "second"}},
	})
	writeJSONL(t, filepath.Join(root, "F--ws-bar", "ccc.jsonl"), []map[string]interface{}{
		{"type": "user", "cwd": "F:\\ws\\bar", "message": map[string]interface{}{"role": "user", "content": "third"}},
	})

	r := NewReader(root)
	groups, err := r.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	byDir := map[string]SessionGroup{}
	for _, g := range groups {
		byDir[g.EncodedDir] = g
	}
	foo := byDir["F--ws-foo"]
	if len(foo.Sessions) != 2 {
		t.Errorf("F--ws-foo expected 2 sessions, got %d", len(foo.Sessions))
	}
	if foo.Cwd != "F:\\ws\\foo" {
		t.Errorf("F--ws-foo cwd not resolved from jsonl: %q", foo.Cwd)
	}
	bar := byDir["F--ws-bar"]
	if bar.Cwd != "F:\\ws\\bar" {
		t.Errorf("F--ws-bar cwd: %q", bar.Cwd)
	}
}

func TestListSessions_SkipsEmptyJsonl(t *testing.T) {
	root := t.TempDir()
	// Non-empty session in F--ws-foo.
	writeJSONL(t, filepath.Join(root, "F--ws-foo", "aaa.jsonl"), []map[string]interface{}{
		{"type": "user", "cwd": "F:\\ws\\foo", "message": map[string]interface{}{"role": "user", "content": "hi"}},
	})
	// Zero-byte session in the same dir; should be skipped.
	writeRaw(t, filepath.Join(root, "F--ws-foo", "empty.jsonl"), "")
	// Dir whose only session is empty; should produce no group at all.
	writeRaw(t, filepath.Join(root, "F--ws-bar", "lonely.jsonl"), "")

	r := NewReader(root)
	groups, err := r.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group (empty-only group dropped), got %d", len(groups))
	}
	if groups[0].EncodedDir != "F--ws-foo" {
		t.Errorf("EncodedDir = %q", groups[0].EncodedDir)
	}
	if len(groups[0].Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(groups[0].Sessions))
	}
	if filepath.Base(groups[0].Sessions[0].SessionID) != "aaa" {
		t.Errorf("expected non-empty session aaa, got %q", groups[0].Sessions[0].SessionID)
	}
}

func TestListSessions_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	r := NewReader(root)
	groups, err := r.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected zero groups, got %d", len(groups))
	}
}

func TestListSessions_RootMissing(t *testing.T) {
	r := NewReader(filepath.Join(t.TempDir(), "does-not-exist"))
	groups, err := r.ListSessions()
	if err != nil {
		t.Fatalf("expected nil error for missing root, got %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty listing, got %d groups", len(groups))
	}
}

func TestSessionMeta_RichTitlePriorityChain(t *testing.T) {
	cases := []struct {
		name    string
		lines   []map[string]interface{}
		want    string
		wantCwd string
	}{
		{
			name: "custom-title wins over everything",
			lines: []map[string]interface{}{
				{"type": "user", "cwd": "F:\\proj", "message": map[string]interface{}{"role": "user", "content": "hello"}},
				{"type": "agent-name", "agentName": "Lower"},
				{"type": "custom-title", "customTitle": "Top Title"},
			},
			want:    "Top Title",
			wantCwd: "F:\\proj",
		},
		{
			name: "agent-name when no custom title",
			lines: []map[string]interface{}{
				{"type": "agent-name", "agentName": "Agent X"},
				{"type": "user", "cwd": "F:\\p", "message": map[string]interface{}{"role": "user", "content": "hi"}},
			},
			want:    "Agent X",
			wantCwd: "F:\\p",
		},
		{
			name: "first non-meta user message",
			lines: []map[string]interface{}{
				{"type": "user", "cwd": "F:\\p", "isMeta": true, "message": map[string]interface{}{"role": "user", "content": "<local-command-caveat>boilerplate</local-command-caveat>"}},
				{"type": "user", "cwd": "F:\\p", "message": map[string]interface{}{"role": "user", "content": "<system-reminder>noise</system-reminder>"}},
				{"type": "user", "cwd": "F:\\p", "message": map[string]interface{}{"role": "user", "content": "Real first prompt"}},
			},
			want:    "Real first prompt",
			wantCwd: "F:\\p",
		},
		{
			name: "slug fallback when no titles or user msg",
			lines: []map[string]interface{}{
				{"type": "permission-mode", "permissionMode": "default", "slug": "happy-cat"},
			},
			want:    "happy-cat",
			wantCwd: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeJSONL(t, filepath.Join(root, "F--ws", "deadbeef-0000-0000-0000-000000000000.jsonl"), tc.lines)
			r := NewReader(root)
			meta, err := r.SessionMeta(filepath.Join("F--ws", "deadbeef-0000-0000-0000-000000000000"))
			if err != nil {
				t.Fatalf("SessionMeta: %v", err)
			}
			if meta.DisplayName != tc.want {
				t.Errorf("DisplayName = %q, want %q", meta.DisplayName, tc.want)
			}
			if meta.Cwd != tc.wantCwd {
				t.Errorf("Cwd = %q, want %q", meta.Cwd, tc.wantCwd)
			}
		})
	}
}

func TestSessionMeta_InjectedUserMessagesSkipped(t *testing.T) {
	root := t.TempDir()
	writeJSONL(t, filepath.Join(root, "F--p", "x.jsonl"), []map[string]interface{}{
		{"type": "user", "message": map[string]interface{}{"role": "user", "content": "Caveat: blah"}},
		{"type": "user", "message": map[string]interface{}{"role": "user", "content": "<command-name>foo</command-name>"}},
		{"type": "user", "message": map[string]interface{}{"role": "user", "content": "first real prompt"}},
	})
	r := NewReader(root)
	meta, err := r.SessionMeta(filepath.Join("F--p", "x"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if meta.DisplayName != "first real prompt" {
		t.Errorf("got %q", meta.DisplayName)
	}
}

func TestReadEvents_PaginationAndTotal(t *testing.T) {
	root := t.TempDir()
	lines := make([]map[string]interface{}, 0, 7)
	for i := 0; i < 7; i++ {
		lines = append(lines, map[string]interface{}{
			"type":    "user",
			"message": map[string]interface{}{"role": "user", "content": fmt.Sprintf("msg %d", i)},
		})
	}
	writeJSONL(t, filepath.Join(root, "F--p", "x.jsonl"), lines)
	r := NewReader(root)

	page, err := r.ReadEvents(filepath.Join("F--p", "x"), 0, 3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if page.Total != 7 {
		t.Errorf("Total = %d", page.Total)
	}
	if !page.HasMore {
		t.Errorf("HasMore should be true")
	}
	if len(page.Events) != 3 {
		t.Fatalf("len(Events) = %d", len(page.Events))
	}
	if page.Events[0].Index != 0 || page.Events[2].Index != 2 {
		t.Errorf("indices = %d %d", page.Events[0].Index, page.Events[2].Index)
	}

	page2, err := r.ReadEvents(filepath.Join("F--p", "x"), 5, 3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if page2.HasMore {
		t.Errorf("last page should not HasMore")
	}
	if len(page2.Events) != 2 {
		t.Errorf("got %d events, want 2", len(page2.Events))
	}
	if page2.Events[0].Index != 5 {
		t.Errorf("offset 5 first index = %d", page2.Events[0].Index)
	}
}

func TestReadEvents_SkipsCorruptLines(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "F--p", "x.jsonl")
	good := `{"type":"user","message":{"role":"user","content":"hi"}}`
	bad := `{"type":"user", broken json`
	writeRaw(t, path, good+"\n"+bad+"\n"+good+"\n")

	r := NewReader(root)
	page, err := r.ReadEvents(filepath.Join("F--p", "x"), 0, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if page.Total != 2 {
		t.Errorf("Total = %d, want 2", page.Total)
	}
	if page.SkippedLines != 1 {
		t.Errorf("SkippedLines = %d, want 1", page.SkippedLines)
	}
	if len(page.Events) != 2 {
		t.Errorf("got %d events", len(page.Events))
	}
}

func TestNormalizeEvent_RedactsThinking(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "F--p", "x.jsonl")
	writeJSONL(t, path, []map[string]interface{}{
		{
			"type": "assistant",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"type":      "thinking",
						"thinking":  "secret",
						"signature": "sig123",
					},
				},
			},
		},
	})
	r := NewReader(root)
	page, err := r.ReadEvents(filepath.Join("F--p", "x"), 0, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(page.Events) != 1 {
		t.Fatalf("expected 1 event")
	}
	ev := page.Events[0]
	if ev.Kind != "assistant" {
		t.Errorf("Kind = %q, want assistant", ev.Kind)
	}
	if ev.Subtype != "assistant.thinking" {
		t.Errorf("Subtype = %q", ev.Subtype)
	}
	msg := ev.Raw["message"].(map[string]interface{})
	block := msg["content"].([]interface{})[0].(map[string]interface{})
	red, ok := block["thinking"].(map[string]interface{})
	if !ok {
		t.Fatalf("thinking field not redacted: %T", block["thinking"])
	}
	if red["_redacted"] != "thinking" {
		t.Errorf("_redacted field = %v", red["_redacted"])
	}
	sig, ok := block["signature"].(map[string]interface{})
	if !ok {
		t.Fatalf("signature field not redacted: %T", block["signature"])
	}
	if sig["_redacted"] != "signature" {
		t.Errorf("signature _redacted = %v", sig["_redacted"])
	}
}

func TestNormalizeEvent_SplitsToolUse(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "F--p", "x.jsonl")
	writeJSONL(t, path, []map[string]interface{}{
		{
			"type": "assistant",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"type":  "tool_use",
						"id":    "toolu_x",
						"name":  "Edit",
						"input": map[string]interface{}{"file_path": "/tmp/foo", "old_string": "a", "new_string": "b"},
					},
				},
			},
		},
	})
	r := NewReader(root)
	page, _ := r.ReadEvents(filepath.Join("F--p", "x"), 0, 10)
	if page.Events[0].Kind != "tool_use" {
		t.Errorf("Kind = %q, want tool_use", page.Events[0].Kind)
	}
	if !strings.Contains(page.Events[0].Summary, "Edit") {
		t.Errorf("Summary missing tool name: %q", page.Events[0].Summary)
	}
}

func TestNormalizeEvent_UserToolResult(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "F--p", "x.jsonl")
	writeJSONL(t, path, []map[string]interface{}{
		{
			"type": "user",
			"message": map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_x",
						"content":     "file edited",
					},
				},
			},
		},
	})
	r := NewReader(root)
	page, _ := r.ReadEvents(filepath.Join("F--p", "x"), 0, 10)
	if page.Events[0].Kind != "tool_result" {
		t.Errorf("Kind = %q", page.Events[0].Kind)
	}
}

func TestNormalizeEvent_MetaIsMeta(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "F--p", "x.jsonl")
	writeJSONL(t, path, []map[string]interface{}{
		{"type": "user", "isMeta": true, "message": map[string]interface{}{"role": "user", "content": "<local-command-caveat>boilerplate</local-command-caveat>"}},
	})
	r := NewReader(root)
	page, _ := r.ReadEvents(filepath.Join("F--p", "x"), 0, 10)
	if page.Events[0].Kind != "meta" {
		t.Errorf("Kind = %q, want meta", page.Events[0].Kind)
	}
}
