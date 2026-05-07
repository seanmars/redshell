package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProbeVersion_ClaudeFormat(t *testing.T) {
	svc := NewServiceWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("2.1.119 (Claude Code)\n"), nil
	})
	got := svc.probeVersion(context.Background(), "claude")
	if got != "2.1.119" {
		t.Fatalf("expected 2.1.119, got %q", got)
	}
}

func TestProbeVersion_CopilotFormat(t *testing.T) {
	svc := NewServiceWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("GitHub Copilot CLI 1.0.34.\nRun 'copilot update' to check for updates.\n"), nil
	})
	got := svc.probeVersion(context.Background(), "copilot")
	if got != "1.0.34" {
		t.Fatalf("expected 1.0.34, got %q", got)
	}
}

func TestProbeVersion_EmptyOutput(t *testing.T) {
	svc := NewServiceWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(""), nil
	})
	got := svc.probeVersion(context.Background(), "anything")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestProbeVersion_NonZeroExit(t *testing.T) {
	svc := NewServiceWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, errors.New("exec: not found")
	})
	got := svc.probeVersion(context.Background(), "missing")
	if got != "" {
		t.Fatalf("expected empty string for failed exec, got %q", got)
	}
}

func TestProbeVersion_TimeoutReturnsEmpty(t *testing.T) {
	svc := NewServiceWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	got := svc.probeVersion(ctx, "slow")
	if got != "" {
		t.Fatalf("expected empty string on timeout, got %q", got)
	}
}

func TestProbeVersion_NoSemverInOutput(t *testing.T) {
	svc := NewServiceWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("unknown command\n"), nil
	})
	got := svc.probeVersion(context.Background(), "weird")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestListAgents_PopulatesShape(t *testing.T) {
	svc := NewServiceWithRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "claude":
			return []byte("2.1.119 (Claude Code)\n"), nil
		case "copilot":
			return []byte("GitHub Copilot CLI 1.0.34.\n"), nil
		}
		return nil, errors.New("unknown bin")
	})
	agents := svc.ListAgents()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	want := map[string]struct{ version, settingsFile, configDir string }{
		"claude":  {"2.1.119", "~/.claude/settings.json", "~/.claude"},
		"copilot": {"1.0.34", "~/.copilot/config.json", "~/.copilot"},
	}
	for _, a := range agents {
		w, ok := want[a.ID]
		if !ok {
			t.Fatalf("unexpected agent id %q", a.ID)
		}
		if a.Version != w.version {
			t.Errorf("%s version: want %q, got %q", a.ID, w.version, a.Version)
		}
		if a.SettingsFile != w.settingsFile {
			t.Errorf("%s settingsFile: want %q, got %q", a.ID, w.settingsFile, a.SettingsFile)
		}
		if a.ConfigDir != w.configDir {
			t.Errorf("%s configDir: want %q, got %q", a.ID, w.configDir, a.ConfigDir)
		}
	}
}
