package copilot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeWorkspace(t *testing.T, sessionDir string, ws map[string]interface{}) {
	t.Helper()
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	var buf strings.Builder
	for k, v := range ws {
		switch tv := v.(type) {
		case string:
			fmt.Fprintf(&buf, "%s: %q\n", k, tv)
		case int, int64, bool:
			fmt.Fprintf(&buf, "%s: %v\n", k, tv)
		default:
			fmt.Fprintf(&buf, "%s: %v\n", k, tv)
		}
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "workspace.yaml"), []byte(buf.String()), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}

func writeEvents(t *testing.T, sessionDir string, lines []map[string]interface{}) {
	t.Helper()
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	var buf strings.Builder
	for _, line := range lines {
		b, err := json.Marshal(line)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte(buf.String()), 0o644); err != nil {
		t.Fatalf("write jsonl: %v", err)
	}
}

func TestListSessions_FlatSortedByCreatedAt(t *testing.T) {
	root := t.TempDir()
	writeWorkspace(t, filepath.Join(root, "newer"), map[string]interface{}{
		"id":         "newer",
		"summary":    "Newer Session",
		"created_at": "2026-04-26T05:00:00.000Z",
		"updated_at": "2026-04-26T05:30:00.000Z",
	})
	writeEvents(t, filepath.Join(root, "newer"), []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{"content": "hi"}},
	})
	writeWorkspace(t, filepath.Join(root, "older"), map[string]interface{}{
		"id":         "older",
		"summary":    "Older Session",
		"created_at": "2026-04-25T05:00:00.000Z",
		"updated_at": "2026-04-25T05:30:00.000Z",
	})
	writeEvents(t, filepath.Join(root, "older"), []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{"content": "hi"}},
	})

	r := NewReader(root)
	flat, err := r.ListSessions()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(flat) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(flat))
	}
	if flat[0].SessionID != "newer" {
		t.Errorf("expected newer first, got %v", flat[0].SessionID)
	}
}

func TestListSessions_MissingRoot(t *testing.T) {
	r := NewReader(filepath.Join(t.TempDir(), "no-such-dir"))
	flat, err := r.ListSessions()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(flat) != 0 {
		t.Errorf("expected empty, got %d", len(flat))
	}
}

func TestListSessions_MinimumShapeWorkspaceYaml(t *testing.T) {
	root := t.TempDir()
	// Minimum shape per analysis doc: only id + summary_count + created_at + updated_at.
	writeWorkspace(t, filepath.Join(root, "minimal"), map[string]interface{}{
		"id":            "minimal",
		"summary_count": 0,
		"created_at":    "2026-04-26T05:00:00.000Z",
		"updated_at":    "2026-04-26T05:00:00.000Z",
	})
	writeEvents(t, filepath.Join(root, "minimal"), []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{"content": "hi"}},
	})
	r := NewReader(root)
	flat, err := r.ListSessions()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(flat) != 1 {
		t.Fatalf("expected 1 session, got %d", len(flat))
	}
	if flat[0].Summary != "" {
		t.Errorf("Summary should be empty for minimum shape")
	}
}

func TestListSessions_SkipsDirsWithoutManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "no-manifest"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeWorkspace(t, filepath.Join(root, "ok"), map[string]interface{}{
		"id":         "ok",
		"created_at": "2026-04-26T05:00:00.000Z",
	})
	writeEvents(t, filepath.Join(root, "ok"), []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{"content": "hi"}},
	})
	r := NewReader(root)
	flat, _ := r.ListSessions()
	if len(flat) != 1 {
		t.Errorf("expected to skip no-manifest dir, got %d sessions", len(flat))
	}
}

func TestListSessions_SkipsSessionsWithoutEvents(t *testing.T) {
	root := t.TempDir()
	// Session with workspace but no events.jsonl at all.
	writeWorkspace(t, filepath.Join(root, "missing-events"), map[string]interface{}{
		"id":         "missing-events",
		"created_at": "2026-04-26T05:00:00.000Z",
	})
	// Session with workspace but a zero-byte events.jsonl.
	writeWorkspace(t, filepath.Join(root, "empty-events"), map[string]interface{}{
		"id":         "empty-events",
		"created_at": "2026-04-26T05:00:00.000Z",
	})
	if err := os.WriteFile(filepath.Join(root, "empty-events", "events.jsonl"), nil, 0o644); err != nil {
		t.Fatalf("write empty events: %v", err)
	}
	// Session with workspace and at least one event.
	writeWorkspace(t, filepath.Join(root, "with-events"), map[string]interface{}{
		"id":         "with-events",
		"created_at": "2026-04-26T05:00:00.000Z",
	})
	writeEvents(t, filepath.Join(root, "with-events"), []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{"content": "hi"}},
	})

	r := NewReader(root)
	flat, err := r.ListSessions()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(flat) != 1 {
		t.Fatalf("expected only the with-events session, got %d", len(flat))
	}
	if flat[0].SessionID != "with-events" {
		t.Errorf("SessionID = %q", flat[0].SessionID)
	}
	if !flat[0].HasEvents {
		t.Errorf("HasEvents should be true for retained sessions")
	}
}

func TestSessionMeta_DisplayNamePriority(t *testing.T) {
	cases := []struct {
		name string
		ws   map[string]interface{}
		evs  []map[string]interface{}
		want string
	}{
		{
			name: "summary wins",
			ws:   map[string]interface{}{"id": "x", "summary": "Hello", "repository": "owner/repo", "cwd": "F:\\p", "created_at": "t"},
			want: "Hello",
		},
		{
			name: "first user message when no summary",
			ws:   map[string]interface{}{"id": "x", "repository": "owner/repo", "created_at": "t"},
			evs: []map[string]interface{}{
				{"type": "session.start", "data": map[string]interface{}{}},
				{"type": "user.message", "data": map[string]interface{}{"content": "First user message here"}},
			},
			want: "First user message here",
		},
		{
			name: "repository when no summary or events",
			ws:   map[string]interface{}{"id": "x", "repository": "owner/repo", "cwd": "F:\\p", "created_at": "t"},
			want: "owner/repo",
		},
		{
			name: "cwd when no other source",
			ws:   map[string]interface{}{"id": "x", "cwd": "F:\\p", "created_at": "t"},
			want: "F:\\p",
		},
		{
			name: "session id short fallback",
			ws:   map[string]interface{}{"id": "ab12cd34efgh", "created_at": "t"},
			want: "ab12cd34",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			sessionID := "ab12cd34efgh"
			sessionDir := filepath.Join(root, sessionID)
			writeWorkspace(t, sessionDir, tc.ws)
			if len(tc.evs) > 0 {
				writeEvents(t, sessionDir, tc.evs)
			}
			r := NewReader(root)
			meta, err := r.SessionMeta(sessionID)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if meta.DisplayName != tc.want {
				t.Errorf("DisplayName = %q, want %q", meta.DisplayName, tc.want)
			}
		})
	}
}

func TestReadEvents_MissingFileEmpty(t *testing.T) {
	root := t.TempDir()
	writeWorkspace(t, filepath.Join(root, "x"), map[string]interface{}{
		"id": "x", "created_at": "t",
	})
	r := NewReader(root)
	page, err := r.ReadEvents("x", 0, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("Total = %d", page.Total)
	}
	if len(page.Events) != 0 {
		t.Errorf("Events = %d", len(page.Events))
	}
}

func TestReadEvents_LargeEncryptedContentWithinBuffer(t *testing.T) {
	root := t.TempDir()
	sessionID := "big"
	sessionDir := filepath.Join(root, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeWorkspace(t, sessionDir, map[string]interface{}{"id": "big", "created_at": "t"})

	// 6 MiB encryptedContent — well above the 64 KB scanner default and
	// well below the 16 MiB ceiling.
	big := strings.Repeat("a", 6*1024*1024)
	bigEvent := map[string]interface{}{
		"type": "assistant.message",
		"data": map[string]interface{}{
			"messageId":        "m1",
			"content":          "hi",
			"encryptedContent": big,
			"reasoningOpaque":  big,
		},
	}
	b, err := json.Marshal(bigEvent)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	r := NewReader(root)
	page, err := r.ReadEvents(sessionID, 0, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("Total = %d", page.Total)
	}
	ev := page.Events[0]
	data := ev.Raw["data"].(map[string]interface{})
	enc, ok := data["encryptedContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("encryptedContent not redacted: %T", data["encryptedContent"])
	}
	if enc["_redacted"] != "encryptedContent" {
		t.Errorf("redaction marker wrong: %v", enc["_redacted"])
	}
	// Ensure original size preserved
	if size, ok := enc["_size"].(int); !ok || size != len(big) {
		t.Errorf("_size wrong: %v", enc["_size"])
	}
	rop, ok := data["reasoningOpaque"].(map[string]interface{})
	if !ok {
		t.Fatalf("reasoningOpaque not redacted")
	}
	if rop["_redacted"] != "reasoningOpaque" {
		t.Errorf("rop redacted = %v", rop["_redacted"])
	}
}

func TestNormalizeEvent_AssistantReasoning(t *testing.T) {
	root := t.TempDir()
	sessionDir := filepath.Join(root, "x")
	writeWorkspace(t, sessionDir, map[string]interface{}{"id": "x", "created_at": "t"})
	writeEvents(t, sessionDir, []map[string]interface{}{
		{"type": "assistant.reasoning", "data": map[string]interface{}{"reasoningId": "r1", "content": "Planning step 1\nstep 2"}},
	})
	r := NewReader(root)
	page, _ := r.ReadEvents("x", 0, 10)
	ev := page.Events[0]
	if ev.Kind != "assistant" {
		t.Errorf("Kind = %q", ev.Kind)
	}
	if ev.Subtype != "assistant.reasoning" {
		t.Errorf("Subtype = %q", ev.Subtype)
	}
}

func TestNormalizeEvent_ToolStartAndComplete(t *testing.T) {
	root := t.TempDir()
	sessionDir := filepath.Join(root, "x")
	writeWorkspace(t, sessionDir, map[string]interface{}{"id": "x", "created_at": "t"})
	writeEvents(t, sessionDir, []map[string]interface{}{
		{"type": "tool.execution_start", "data": map[string]interface{}{
			"toolCallId": "c1",
			"toolName":   "view",
			"arguments":  map[string]interface{}{"path": "/x"},
		}},
		{"type": "tool.execution_complete", "data": map[string]interface{}{
			"toolCallId": "c1",
			"success":    true,
			"result":     map[string]interface{}{"content": "ok"},
		}},
	})
	r := NewReader(root)
	page, _ := r.ReadEvents("x", 0, 10)
	if page.Events[0].Kind != "tool_use" {
		t.Errorf("start.Kind = %q", page.Events[0].Kind)
	}
	if !strings.Contains(page.Events[0].Summary, "view") {
		t.Errorf("start summary missing tool name: %q", page.Events[0].Summary)
	}
	if page.Events[1].Kind != "tool_result" {
		t.Errorf("complete.Kind = %q", page.Events[1].Kind)
	}
}

func TestNormalizeEvent_AttachmentChildren(t *testing.T) {
	root := t.TempDir()
	sessionDir := filepath.Join(root, "x")
	writeWorkspace(t, sessionDir, map[string]interface{}{"id": "x", "created_at": "t"})
	writeEvents(t, sessionDir, []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{
			"content": "look at this",
			"attachments": []interface{}{
				map[string]interface{}{
					"type":        "file",
					"path":        "C:\\Users\\test.ts",
					"displayName": "@test.ts",
				},
			},
		}},
	})
	r := NewReader(root)
	page, _ := r.ReadEvents("x", 0, 10)
	ev := page.Events[0]
	if len(ev.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(ev.Children))
	}
	if ev.Children[0].Kind != "attachment" {
		t.Errorf("child Kind = %q", ev.Children[0].Kind)
	}
}

func TestReadEvents_IgnoresTransformedContent(t *testing.T) {
	root := t.TempDir()
	sessionDir := filepath.Join(root, "x")
	writeWorkspace(t, sessionDir, map[string]interface{}{"id": "x", "created_at": "t"})
	writeEvents(t, sessionDir, []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{
			"content":            "real prompt",
			"transformedContent": "<system noise>real prompt</system noise>",
		}},
	})
	r := NewReader(root)
	page, _ := r.ReadEvents("x", 0, 10)
	if !strings.Contains(page.Events[0].Summary, "real prompt") {
		t.Errorf("summary should use content, got %q", page.Events[0].Summary)
	}
	if strings.Contains(page.Events[0].Summary, "system noise") {
		t.Errorf("summary leaked transformedContent: %q", page.Events[0].Summary)
	}
}
