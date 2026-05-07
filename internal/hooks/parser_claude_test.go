package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read fixture %q: %v", rel, err)
	}
	return data
}

func TestParseClaudeFile_FullFixture(t *testing.T) {
	res, err := parseClaudeFile(loadFixture(t, "claude/full.json"), "src-user")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if res.DisableAll {
		t.Errorf("DisableAll should be false")
	}

	wantEvents := map[string]int{
		"PreToolUse":       3,
		"UserPromptSubmit": 1,
		"SessionStart":     1,
	}
	got := map[string]int{}
	for _, h := range res.Hooks {
		got[h.Event]++
	}
	for ev, want := range wantEvents {
		if got[ev] != want {
			t.Errorf("event %q: got %d hooks, want %d", ev, got[ev], want)
		}
	}

	var http, mcp, prompt, agent *Hook
	for i := range res.Hooks {
		h := &res.Hooks[i]
		switch h.Type {
		case "http":
			http = h
		case "mcp_tool":
			mcp = h
		case "prompt":
			prompt = h
		case "agent":
			agent = h
		}
	}
	if http == nil || http.Summary != "https://example.com/hook" {
		t.Errorf("http summary: %+v", http)
	}
	if mcp == nil || mcp.Summary != "memory:remember" {
		t.Errorf("mcp summary: %+v", mcp)
	}
	if prompt == nil || prompt.Summary != "Audit $ARGUMENTS for secrets" {
		t.Errorf("prompt summary: %+v", prompt)
	}
	if agent == nil || agent.Summary != "Summarise project state" {
		t.Errorf("agent summary: %+v", agent)
	}

	for _, h := range res.Hooks {
		if h.SourceID != "src-user" {
			t.Errorf("hook %q: SourceID = %q, want src-user", h.ID, h.SourceID)
		}
		if h.ID == "" {
			t.Errorf("hook on event %q has empty ID", h.Event)
		}
	}
}

func TestParseClaudeFile_DisableAllHooks(t *testing.T) {
	res, err := parseClaudeFile(loadFixture(t, "claude/disable_all.json"), "src-user")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !res.DisableAll {
		t.Errorf("DisableAll should be true")
	}
	if len(res.Hooks) != 1 {
		t.Errorf("hooks count: got %d, want 1", len(res.Hooks))
	}
}

func TestParseClaudeFile_UnknownTypePreserved(t *testing.T) {
	res, err := parseClaudeFile(loadFixture(t, "claude/unknown_type.json"), "src-user")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(res.Hooks) != 1 {
		t.Fatalf("hooks count: got %d, want 1", len(res.Hooks))
	}
	h := res.Hooks[0]
	if h.Type != "future_shape" {
		t.Errorf("type = %q, want future_shape", h.Type)
	}
	if h.Summary != "" {
		t.Errorf("unknown type should produce empty summary, got %q", h.Summary)
	}
	if h.Raw["extra"] == nil {
		t.Errorf("Raw should preserve unknown 'extra' field")
	}
}

func TestParseClaudeFile_MissingMatcher(t *testing.T) {
	res, err := parseClaudeFile(loadFixture(t, "claude/full.json"), "src-user")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var sessionStart *Hook
	for i := range res.Hooks {
		if res.Hooks[i].Event == "SessionStart" {
			sessionStart = &res.Hooks[i]
		}
	}
	if sessionStart == nil {
		t.Fatalf("SessionStart hook not present")
	}
	if sessionStart.Matcher != "" {
		t.Errorf("missing matcher should produce empty string, got %q", sessionStart.Matcher)
	}
}

func TestParseClaudeFile_RegexMatcher(t *testing.T) {
	res, _ := parseClaudeFile(loadFixture(t, "claude/full.json"), "src-user")
	for _, h := range res.Hooks {
		if h.Type == "mcp_tool" && h.Matcher != "^Notebook" {
			t.Errorf("regex matcher: got %q, want ^Notebook", h.Matcher)
		}
	}
}

func TestParseClaudeFile_PipeMatcher(t *testing.T) {
	res, _ := parseClaudeFile(loadFixture(t, "claude/disable_all.json"), "src-user")
	if len(res.Hooks) == 0 || res.Hooks[0].Matcher != "Edit|Write" {
		t.Errorf("pipe matcher not preserved: %+v", res.Hooks)
	}
}

func TestParseClaudeFile_Malformed(t *testing.T) {
	_, err := parseClaudeFile(loadFixture(t, "claude/malformed.json"), "src-user")
	if err == nil {
		t.Errorf("malformed JSON should error")
	}
}

func TestDedupKey_TypeNormalization(t *testing.T) {
	cases := []struct {
		hook Hook
		want string
	}{
		{Hook{Type: "command", Summary: "echo hi"}, "command:echo hi"},
		{Hook{Type: "http", Summary: "https://x"}, "http:https://x"},
		{Hook{Type: "mcp_tool", Summary: "memory:remember"}, "mcp_tool:memory:remember"},
		{Hook{Type: "prompt", Summary: "Audit"}, "prompt:Audit"},
		{Hook{Type: "future_shape", Summary: ""}, "future_shape:"},
		{Hook{Type: "", Summary: ""}, ""},
	}
	for _, c := range cases {
		if got := dedupKey(c.hook); got != c.want {
			t.Errorf("dedupKey(%+v) = %q, want %q", c.hook, got, c.want)
		}
	}
}

func TestHookID_StableForSameCoords(t *testing.T) {
	a := hookID("src-x", "PreToolUse", 0, 1)
	b := hookID("src-x", "PreToolUse", 0, 1)
	c := hookID("src-x", "PreToolUse", 0, 2)
	if a != b {
		t.Errorf("same coords should produce same id: %q vs %q", a, b)
	}
	if a == c {
		t.Errorf("different handler index should differ: %q", a)
	}
}
