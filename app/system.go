package app

import (
	"context"

	"redshell/internal/osopen"
)

type SystemApp struct {
	ctx     context.Context
	version string
}

func NewSystemApp(version string) *SystemApp {
	return &SystemApp{version: version}
}

func (a *SystemApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *SystemApp) OpenPath(path string) error {
	return osopen.OpenPath(path)
}

func (a *SystemApp) GetAppVersion() string {
	return a.version
}
