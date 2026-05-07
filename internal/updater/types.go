package updater

import (
	"context"
	"errors"
	"time"
)

const ChecksumsAssetName = "checksums.txt"

type Release struct {
	Version      string    `json:"version"`
	PublishedAt  time.Time `json:"publishedAt"`
	Notes        string    `json:"notes"`
	AssetURL     string    `json:"assetUrl"`
	AssetName    string    `json:"assetName"`
	AssetSize    int64     `json:"assetSize"`
	ChecksumsURL string    `json:"checksumsUrl"`
}

type Provider interface {
	Name() string
	LatestRelease(ctx context.Context) (Release, error)
}

var (
	ErrPlatformUnsupported = errors.New("auto-update is not supported on this platform")
	ErrAssetNotFound       = errors.New("expected asset not found in release")
	ErrChecksumsNotFound   = errors.New("checksums asset not found in release")
)

func AssetNameFor(goos, goarch string) string {
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}
	return "redshell-" + goos + "-" + goarch + suffix
}
