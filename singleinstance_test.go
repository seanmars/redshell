package main

import (
	"testing"

	"redshell/internal/updater"
)

func TestParseWaitParentPID(t *testing.T) {
	flag := updater.WaitParentPIDFlag
	tests := []struct {
		name    string
		args    []string
		wantPID int
		wantOK  bool
	}{
		{"present", []string{flag + "=1234"}, 1234, true},
		{"present among others", []string{"--other", flag + "=42", "extra"}, 42, true},
		{"absent", []string{"--other", "extra"}, 0, false},
		{"empty", nil, 0, false},
		{"malformed non-numeric", []string{flag + "=abc"}, 0, false},
		{"malformed empty value", []string{flag + "="}, 0, false},
		{"zero rejected", []string{flag + "=0"}, 0, false},
		{"negative rejected", []string{flag + "=-5"}, 0, false},
		{"bare flag without value", []string{flag}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pid, ok := parseWaitParentPID(tt.args)
			if pid != tt.wantPID || ok != tt.wantOK {
				t.Fatalf("parseWaitParentPID(%v) = (%d, %v), want (%d, %v)", tt.args, pid, ok, tt.wantPID, tt.wantOK)
			}
		})
	}
}
