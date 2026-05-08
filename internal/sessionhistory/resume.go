package sessionhistory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrTerminalUnsupported is returned by ResumeSession on platforms where
// launching a new terminal window is not implemented.
var ErrTerminalUnsupported = errors.New("resume terminal not supported on this platform")

// ErrProjectCwdMissing is returned when ResumeSession is called with a
// non-empty cwd that does not resolve to an existing directory (the path is
// not absolute, does not exist, or is a file rather than a directory). The
// frontend surfaces the wrapped message in a toast so the user knows which
// path is missing before retrying.
var ErrProjectCwdMissing = errors.New("project directory does not exist")

// validBasename matches the shape of every Claude/Copilot session id basename
// observed on disk. Restricting to this set before interpolating into a shell
// command prevents injection via the session id argument.
var validBasename = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// agentCLI maps an agentID to the CLI binary name used to resume a session.
// Both binaries are expected on PATH; if missing, the spawned terminal
// surfaces the "command not found" error to the user.
var agentCLI = map[string]string{
	"claude":  "claude",
	"copilot": "copilot",
}

// launchResumeTerminal is provided by terminal_<os>.go. Tests override it.
var launchResumeTerminal = defaultLaunchResumeTerminal

// ResumeSession opens a new terminal window running `<cli> --resume <id>`
// with its working directory set to `cwd` (the session's project directory,
// not the session-file directory). The sessionID accepted here may be the
// path-prefixed Claude shape `<encoded-cwd>/<uuid>` or the bare Copilot UUID;
// the basename is extracted before invocation. The basename is then strictly
// validated before being interpolated into the shell command.
//
// `cwd` is treated as follows:
//   - empty: launch in the spawning process's default cwd, no error.
//   - non-empty and resolves to an existing directory: forward to the
//     launcher.
//   - non-empty and invalid (not absolute, missing, or not a directory):
//     return ErrProjectCwdMissing wrapped with the offending path; the
//     terminal SHALL NOT be launched.
func (s *Service) ResumeSession(agentID, sessionID, cwd string) error {
	cli, ok := agentCLI[agentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownAgent, agentID)
	}
	if sessionID == "" {
		return fmt.Errorf("%w: empty", ErrInvalidSessionID)
	}
	short := sessionID
	if i := strings.LastIndex(short, "/"); i >= 0 {
		short = short[i+1:]
	}
	if !validBasename.MatchString(short) {
		return fmt.Errorf("%w: bad characters", ErrInvalidSessionID)
	}
	resolved, err := resolveCwd(cwd)
	if err != nil {
		return err
	}
	return launchResumeTerminal(cli, short, resolved)
}

// resolveCwd validates the cwd carried in a Resume request.
//
// An empty cwd is allowed and yields ("", nil) so the launcher inherits the
// spawning process's cwd. A non-empty cwd MUST be absolute and resolve to an
// existing directory; otherwise this returns ErrProjectCwdMissing wrapped
// with the offending path so the frontend can show it in the failure toast.
func resolveCwd(cwd string) (string, error) {
	if cwd == "" {
		return "", nil
	}
	if !filepath.IsAbs(cwd) {
		return "", fmt.Errorf("%w: %s", ErrProjectCwdMissing, cwd)
	}
	clean := filepath.Clean(cwd)
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrProjectCwdMissing, cwd)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrProjectCwdMissing, cwd)
	}
	return clean, nil
}
