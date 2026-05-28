//go:build windows

package main

import (
	"os"
	"testing"
	"time"
)

// TestWaitForParentExitTimesOut verifies the helper stops at the timeout
// rather than blocking forever when the target process never exits. It waits
// on the live test process itself with a short timeout.
func TestWaitForParentExitTimesOut(t *testing.T) {
	start := time.Now()
	waitForParentExit(os.Getpid(), 50*time.Millisecond)
	elapsed := time.Since(start)

	if elapsed < 40*time.Millisecond {
		t.Fatalf("returned too early (%v); expected to wait ~50ms", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("waited too long (%v); timeout not honored", elapsed)
	}
}

// TestWaitForParentExitMissingProcess verifies the helper returns promptly when
// the target PID cannot be opened (already exited / invalid).
func TestWaitForParentExitMissingProcess(t *testing.T) {
	start := time.Now()
	// PID 0xFFFFFFF0 is effectively never a live process; OpenProcess fails
	// and the helper should return immediately.
	waitForParentExit(0xFFFFFFF0, 5*time.Second)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("did not return promptly for unopenable PID (%v)", elapsed)
	}
}
