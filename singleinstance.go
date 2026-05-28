package main

import (
	"strconv"
	"strings"

	"redshell/internal/updater"
)

// parseWaitParentPID scans args for the updater relaunch handshake flag
// (updater.WaitParentPIDFlag, "--wait-parent-pid=<pid>") and returns the PID.
// ok is false when the flag is absent or its value is not a positive integer.
func parseWaitParentPID(args []string) (pid int, ok bool) {
	prefix := updater.WaitParentPIDFlag + "="
	for _, a := range args {
		if !strings.HasPrefix(a, prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(a, prefix))
		if err != nil || n <= 0 {
			return 0, false
		}
		return n, true
	}
	return 0, false
}
