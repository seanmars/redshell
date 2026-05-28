//go:build production

package main

import "github.com/wailsapp/wails/v2/pkg/options"

// newSingleInstanceLock returns the configured single-instance lock for
// production builds. Wails terminates the duplicate process itself; onSecond
// runs in the already-running primary to raise its window.
func newSingleInstanceLock(onSecond func(options.SecondInstanceData)) *options.SingleInstanceLock {
	return &options.SingleInstanceLock{
		UniqueId:               singleInstanceUniqueID,
		OnSecondInstanceLaunch: onSecond,
	}
}
