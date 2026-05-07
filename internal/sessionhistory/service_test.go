package sessionhistory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupRoots(t *testing.T) (claudeRoot, copilotRoot string) {
	t.Helper()
	root := t.TempDir()
	claudeRoot = filepath.Join(root, "claude")
	copilotRoot = filepath.Join(root, "copilot")
	if err := os.MkdirAll(claudeRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(copilotRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return
}

func TestService_UnknownAgentRejected(t *testing.T) {
	cr, cp := setupRoots(t)
	svc := NewServiceWithRoots(map[string]string{"claude": cr, "copilot": cp})

	_, err := svc.ListSessions("vscode-copilot")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("expected ErrUnknownAgent, got %v", err)
	}

	_, err = svc.ListEvents("nope", "session-1", 0, 10)
	if !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("expected ErrUnknownAgent, got %v", err)
	}
}

func TestService_PathTraversalRejected(t *testing.T) {
	cr, cp := setupRoots(t)
	svc := NewServiceWithRoots(map[string]string{"claude": cr, "copilot": cp})

	cases := []struct {
		agent     string
		sessionID string
	}{
		{"claude", "../escape"},
		{"claude", "F--ws/../../../escape"},
		{"copilot", ".."},
		{"copilot", "sess/../.."},
	}
	for _, tc := range cases {
		_, err := svc.ListEvents(tc.agent, tc.sessionID, 0, 10)
		if err == nil {
			t.Errorf("[%s/%s] expected error, got nil", tc.agent, tc.sessionID)
			continue
		}
		if !errors.Is(err, ErrInvalidSessionID) {
			t.Errorf("[%s/%s] expected ErrInvalidSessionID, got %v", tc.agent, tc.sessionID, err)
		}
	}
}

func TestService_EmptySessionIDRejected(t *testing.T) {
	cr, cp := setupRoots(t)
	svc := NewServiceWithRoots(map[string]string{"claude": cr, "copilot": cp})

	_, err := svc.SessionMeta("claude", "")
	if !errors.Is(err, ErrInvalidSessionID) {
		t.Errorf("expected ErrInvalidSessionID for empty session id, got %v", err)
	}
}

func TestService_RoutesToBothReaders(t *testing.T) {
	cr, cp := setupRoots(t)
	// Seed minimum claude session: encoded-cwd dir + jsonl with one user.
	jsonl := `{"type":"user","cwd":"F:\\p","message":{"role":"user","content":"hi"}}` + "\n"
	if err := os.MkdirAll(filepath.Join(cr, "F--p"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cr, "F--p", "abc.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	// Seed minimum copilot session: dir with workspace.yaml.
	if err := os.MkdirAll(filepath.Join(cp, "sess1"), 0o755); err != nil {
		t.Fatal(err)
	}
	yamlContent := "id: sess1\nsummary: Hello\ncreated_at: \"2026-04-26T00:00:00Z\"\n"
	if err := os.WriteFile(filepath.Join(cp, "sess1", "workspace.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	copilotEvents := `{"type":"user.message","data":{"content":"hi"}}` + "\n"
	if err := os.WriteFile(filepath.Join(cp, "sess1", "events.jsonl"), []byte(copilotEvents), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewServiceWithRoots(map[string]string{"claude": cr, "copilot": cp})

	claudeListing, err := svc.ListSessions("claude")
	if err != nil {
		t.Fatalf("claude ListSessions: %v", err)
	}
	if claudeListing.Kind != "groups" {
		t.Errorf("claude listing kind = %q", claudeListing.Kind)
	}
	if len(claudeListing.Groups) != 1 {
		t.Fatalf("claude groups: %d", len(claudeListing.Groups))
	}

	copilotListing, err := svc.ListSessions("copilot")
	if err != nil {
		t.Fatalf("copilot ListSessions: %v", err)
	}
	if copilotListing.Kind != "groups" {
		t.Errorf("copilot kind = %q, want groups", copilotListing.Kind)
	}
	if len(copilotListing.Flat) != 0 {
		t.Errorf("copilot flat should be empty after grouping, got %d", len(copilotListing.Flat))
	}
	if len(copilotListing.Groups) != 1 {
		t.Fatalf("copilot groups: %d", len(copilotListing.Groups))
	}
	if len(copilotListing.Groups[0].Sessions) != 1 {
		t.Fatalf("copilot group sessions: %d", len(copilotListing.Groups[0].Sessions))
	}
	if copilotListing.Groups[0].Sessions[0].Summary != "Hello" {
		t.Errorf("copilot summary: %q", copilotListing.Groups[0].Sessions[0].Summary)
	}
}

func TestService_ListEventsAppliesDefaults(t *testing.T) {
	cr, cp := setupRoots(t)
	if err := os.MkdirAll(filepath.Join(cr, "F--p"), 0o755); err != nil {
		t.Fatal(err)
	}
	var lines []string
	for i := 0; i < 5; i++ {
		lines = append(lines, `{"type":"user","message":{"role":"user","content":"x"}}`)
	}
	if err := os.WriteFile(filepath.Join(cr, "F--p", "z.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewServiceWithRoots(map[string]string{"claude": cr, "copilot": cp})

	page, err := svc.ListEvents("claude", "F--p/z", -1, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if page.Offset != 0 {
		t.Errorf("Offset = %d, want 0 after negative input", page.Offset)
	}
	if page.Limit != 200 {
		t.Errorf("Limit = %d, want 200 default", page.Limit)
	}
	if page.AgentID != "claude" || page.SessionID != "F--p/z" {
		t.Errorf("agent/session not propagated: %+v", page)
	}
}
