package updater

// BuildKind identifies which install pathway the running binary corresponds
// to. It is set at link time via -ldflags "-X 'redshell/internal/updater.BuildKind=installer'"
// when building the NSIS-installable variant; the portable build leaves it at
// the zero/default value.
//
// Values:
//   - "portable" (default): the rename-trick swap path (Swap in rename_windows.go)
//     handles updates. Requires a writable install directory.
//   - "installer": the elevated silent-install path (SpawnInstaller in
//     installer_install_windows.go) handles updates. Triggers UAC; the install
//     directory does NOT need to be writable by the running user.
//
// Any other value is treated as "portable" so a misconfigured ldflag falls
// back to the safer pathway rather than attempting the elevated install.
var BuildKind = "portable"

const (
	BuildKindPortable  = "portable"
	BuildKindInstaller = "installer"
)

// IsInstaller returns true when this binary was built as the NSIS installer
// variant. Use this rather than comparing BuildKind directly so the switch
// happens in one place.
func IsInstaller() bool {
	return BuildKind == BuildKindInstaller
}

// IsPortable returns true for the default portable build, including any
// unrecognized BuildKind value (treated as portable for safety).
func IsPortable() bool {
	return !IsInstaller()
}
