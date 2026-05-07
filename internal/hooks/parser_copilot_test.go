package hooks

import (
	"runtime"
	"testing"
)

func TestParseCopilotFile_AllSixEvents(t *testing.T) {
	hooks, err := parseCopilotFile(loadFixture(t, "copilot/full.json"), "src-cp")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	wantEvents := []string{
		"errorOccurred", "postToolUse", "preToolUse",
		"sessionEnd", "sessionStart", "userPromptSubmitted",
	}
	got := map[string]int{}
	for _, h := range hooks {
		got[h.Event]++
	}
	for _, ev := range wantEvents {
		if got[ev] != 1 {
			t.Errorf("event %q: got %d, want 1", ev, got[ev])
		}
	}
}

func TestParseCopilotFile_MatcherIsAlwaysEmpty(t *testing.T) {
	hooks, _ := parseCopilotFile(loadFixture(t, "copilot/full.json"), "src-cp")
	for _, h := range hooks {
		if h.Matcher != "" {
			t.Errorf("event %q: Matcher should be empty for Copilot, got %q", h.Event, h.Matcher)
		}
	}
}

func TestParseCopilotFile_OSPreferenceForBothScripts(t *testing.T) {
	hooks, _ := parseCopilotFile(loadFixture(t, "copilot/full.json"), "src-cp")
	var sessionStart Hook
	for _, h := range hooks {
		if h.Event == "sessionStart" {
			sessionStart = h
		}
	}
	if runtime.GOOS == "windows" {
		if sessionStart.Summary != "scripts/start.ps1" {
			t.Errorf("windows: summary should prefer powershell, got %q", sessionStart.Summary)
		}
	} else {
		if sessionStart.Summary != "scripts/start.sh" {
			t.Errorf("non-windows: summary should prefer bash, got %q", sessionStart.Summary)
		}
	}
}

func TestParseCopilotFile_SinglePlatformFallsBack(t *testing.T) {
	hooks, _ := parseCopilotFile(loadFixture(t, "copilot/full.json"), "src-cp")
	for _, h := range hooks {
		switch h.Event {
		case "userPromptSubmitted":
			if h.Summary != "scripts/audit.ps1" {
				t.Errorf("powershell-only: got %q", h.Summary)
			}
		case "preToolUse":
			if h.Summary != "scripts/gate.sh" {
				t.Errorf("bash-only: got %q", h.Summary)
			}
		}
	}
}

func TestParseCopilotFile_RawPreservesMetadata(t *testing.T) {
	hooks, _ := parseCopilotFile(loadFixture(t, "copilot/full.json"), "src-cp")
	for _, h := range hooks {
		if h.Event == "sessionEnd" {
			if h.Raw["comment"] != "cleanup" {
				t.Errorf("Raw should preserve comment: got %v", h.Raw["comment"])
			}
		}
	}
}

func TestParseCopilotFile_Malformed(t *testing.T) {
	if _, err := parseCopilotFile(loadFixture(t, "copilot/malformed.json"), "src-cp"); err == nil {
		t.Errorf("malformed JSON should error")
	}
}
