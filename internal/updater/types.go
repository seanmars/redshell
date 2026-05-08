package updater

import (
	"context"
	"errors"
	"fmt"
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
	// InstallerAssetURL and InstallerAssetName are populated when the release
	// publishes an NSIS installer artifact alongside the portable binary.
	// They are empty for releases that only ship the portable variant; the
	// installer install pathway treats empty values as "asset missing" and
	// emits updater:error with stage installer-download.
	InstallerAssetURL  string `json:"installerAssetUrl,omitempty"`
	InstallerAssetName string `json:"installerAssetName,omitempty"`
	InstallerAssetSize int64  `json:"installerAssetSize,omitempty"`
}

type Provider interface {
	Name() string
	LatestRelease(ctx context.Context) (Release, error)
}

var (
	ErrPlatformUnsupported = errors.New("auto-update is not supported on this platform")
	ErrAssetNotFound       = errors.New("expected asset not found in release")
	ErrChecksumsNotFound   = errors.New("checksums asset not found in release")
	ErrInstallerNotFound   = errors.New("installer asset not found in release")
	ErrUACDeclined         = errors.New("user cancelled elevation")
)

func AssetNameFor(goos, goarch string) string {
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}
	return "redshell-" + goos + "-" + goarch + suffix
}

// InstallerAssetNameFor returns the expected NSIS installer asset filename
// for the given OS/arch combination. The installer pathway is currently
// only supported on Windows AMD64; other combinations return an error so a
// future ARM64 (or other) build is a single-file change here.
func InstallerAssetNameFor(goos, goarch string) (string, error) {
	if goos == "windows" && goarch == "amd64" {
		return "RedShell-amd64-installer.exe", nil
	}
	return "", fmt.Errorf("%w: no installer asset defined for %s/%s", ErrPlatformUnsupported, goos, goarch)
}
