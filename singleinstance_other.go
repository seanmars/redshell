//go:build !production

package main

import "github.com/wailsapp/wails/v2/pkg/options"

// newSingleInstanceLock returns nil for non-production builds (dev, test, vet,
// plain go build), leaving multiple instances allowed.
func newSingleInstanceLock(func(options.SecondInstanceData)) *options.SingleInstanceLock {
	return nil
}
