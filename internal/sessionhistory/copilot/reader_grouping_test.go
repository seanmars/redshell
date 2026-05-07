package copilot

import (
	"path/filepath"
	"testing"
)

func sessionWith(t *testing.T, root, id string, ws map[string]interface{}) {
	t.Helper()
	writeWorkspace(t, filepath.Join(root, id), ws)
	writeEvents(t, filepath.Join(root, id), []map[string]interface{}{
		{"type": "user.message", "data": map[string]interface{}{"content": "hi"}},
	})
}

func findGroup(groups []SessionGroup, key string) *SessionGroup {
	for i := range groups {
		if groups[i].Cwd == key {
			return &groups[i]
		}
	}
	return nil
}

func TestListSessionGroups_TwoSessionsSameCwdOneOther(t *testing.T) {
	root := t.TempDir()
	sessionWith(t, root, "s1", map[string]interface{}{
		"id":         "s1",
		"cwd":        "F:/work/redshell",
		"created_at": "2026-04-26T10:00:00Z",
		"updated_at": "2026-04-26T10:00:00Z",
	})
	sessionWith(t, root, "s2", map[string]interface{}{
		"id":         "s2",
		"cwd":        "F:/work/redshell",
		"created_at": "2026-04-26T11:00:00Z",
		"updated_at": "2026-04-26T11:00:00Z",
	})
	sessionWith(t, root, "s3", map[string]interface{}{
		"id":         "s3",
		"cwd":        "F:/work/other",
		"created_at": "2026-04-26T09:00:00Z",
		"updated_at": "2026-04-26T09:00:00Z",
	})

	groups, err := NewReader(root).ListSessionGroups()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d (%+v)", len(groups), groups)
	}
	rs := findGroup(groups, "F:/work/redshell")
	if rs == nil || len(rs.Sessions) != 2 {
		t.Fatalf("redshell group missing or wrong size: %+v", rs)
	}
	other := findGroup(groups, "F:/work/other")
	if other == nil || len(other.Sessions) != 1 {
		t.Fatalf("other group missing or wrong size: %+v", other)
	}
}

func TestListSessionGroups_RepositoryFallbackWhenCwdAndGitRootEmpty(t *testing.T) {
	root := t.TempDir()
	sessionWith(t, root, "s1", map[string]interface{}{
		"id":         "s1",
		"repository": "owner/only-repo",
		"created_at": "2026-04-26T10:00:00Z",
		"updated_at": "2026-04-26T10:00:00Z",
	})

	groups, err := NewReader(root).ListSessionGroups()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Cwd != "owner/only-repo" {
		t.Errorf("group key = %q, want repository value as third-tier fallback", groups[0].Cwd)
	}
}

func TestListSessionGroups_FractionalSecondTimestamps(t *testing.T) {
	// Production Copilot writes `"2026-04-26T05:00:00.000Z"` (RFC3339Nano);
	// guard against time.RFC3339 silently dropping the fractional component
	// and producing equal recencies.
	root := t.TempDir()
	sessionWith(t, root, "earlier", map[string]interface{}{
		"id":         "earlier",
		"cwd":        "F:/work/aaa",
		"created_at": "2026-04-26T05:00:00.000Z",
		"updated_at": "2026-04-26T05:00:00.000Z",
	})
	sessionWith(t, root, "later", map[string]interface{}{
		"id":         "later",
		"cwd":        "F:/work/bbb",
		"created_at": "2026-04-26T05:00:00.500Z",
		"updated_at": "2026-04-26T05:00:00.500Z",
	})

	groups, err := NewReader(root).ListSessionGroups()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Cwd != "F:/work/bbb" {
		t.Errorf("group[0].Cwd = %q, want F:/work/bbb (the .500Z group)", groups[0].Cwd)
	}
}

func TestListSessionGroups_GitRootFallbackWhenCwdEmpty(t *testing.T) {
	root := t.TempDir()
	sessionWith(t, root, "s1", map[string]interface{}{
		"id":         "s1",
		"git_root":   "F:/work/repo-x",
		"repository": "owner/repo-x",
		"created_at": "2026-04-26T10:00:00Z",
		"updated_at": "2026-04-26T10:00:00Z",
	})

	groups, err := NewReader(root).ListSessionGroups()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Cwd != "F:/work/repo-x" {
		t.Errorf("group key = %q, want git_root value", groups[0].Cwd)
	}
}

func TestListSessionGroups_UnknownBucketWhenAllFallbacksEmpty(t *testing.T) {
	root := t.TempDir()
	sessionWith(t, root, "s1", map[string]interface{}{
		"id":         "s1",
		"created_at": "2026-04-26T10:00:00Z",
		"updated_at": "2026-04-26T10:00:00Z",
	})

	groups, err := NewReader(root).ListSessionGroups()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Cwd != UnknownCwd {
		t.Errorf("group key = %q, want %q", groups[0].Cwd, UnknownCwd)
	}
	if len(groups[0].Sessions) != 1 {
		t.Errorf("expected the session to land in the unknown bucket")
	}
}

func TestListSessionGroups_GroupSortByMaxRecency(t *testing.T) {
	root := t.TempDir()
	// Group A's newest is older than group B's newest.
	sessionWith(t, root, "a-old", map[string]interface{}{
		"id":         "a-old",
		"cwd":        "F:/work/aaa",
		"created_at": "2026-04-20T10:00:00Z",
		"updated_at": "2026-04-20T10:00:00Z",
	})
	sessionWith(t, root, "a-mid", map[string]interface{}{
		"id":         "a-mid",
		"cwd":        "F:/work/aaa",
		"created_at": "2026-04-22T10:00:00Z",
		"updated_at": "2026-04-22T10:00:00Z",
	})
	sessionWith(t, root, "b-newest", map[string]interface{}{
		"id":         "b-newest",
		"cwd":        "F:/work/bbb",
		"created_at": "2026-04-25T10:00:00Z",
		"updated_at": "2026-04-25T10:00:00Z",
	})

	groups, err := NewReader(root).ListSessionGroups()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Cwd != "F:/work/bbb" {
		t.Errorf("group[0].Cwd = %q, want bbb (newer max)", groups[0].Cwd)
	}
	if groups[1].Cwd != "F:/work/aaa" {
		t.Errorf("group[1].Cwd = %q, want aaa", groups[1].Cwd)
	}
}

func TestListSessionGroups_WithinGroupCreatedAtDesc(t *testing.T) {
	root := t.TempDir()
	sessionWith(t, root, "old", map[string]interface{}{
		"id":         "old",
		"cwd":        "F:/work/x",
		"created_at": "2026-04-20T10:00:00Z",
	})
	sessionWith(t, root, "new", map[string]interface{}{
		"id":         "new",
		"cwd":        "F:/work/x",
		"created_at": "2026-04-26T10:00:00Z",
	})
	sessionWith(t, root, "mid", map[string]interface{}{
		"id":         "mid",
		"cwd":        "F:/work/x",
		"created_at": "2026-04-23T10:00:00Z",
	})

	groups, err := NewReader(root).ListSessionGroups()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	g := groups[0]
	if len(g.Sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(g.Sessions))
	}
	if g.Sessions[0].SessionID != "new" || g.Sessions[1].SessionID != "mid" || g.Sessions[2].SessionID != "old" {
		t.Errorf("intra-group order = [%s, %s, %s], want [new, mid, old]",
			g.Sessions[0].SessionID, g.Sessions[1].SessionID, g.Sessions[2].SessionID)
	}
}
