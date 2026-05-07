package osopen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"redshell/internal/sysproc"
)

type dispatcher func(absPath string) error

var dispatch dispatcher = osDispatch

func OpenPath(path string) error {
	expanded, err := expandHome(path)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", path, err)
	}
	if _, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", abs)
		}
		return fmt.Errorf("stat %q: %w", abs, err)
	}
	return dispatch(abs)
}

func expandHome(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func osDispatch(absPath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", absPath)
	case "darwin":
		cmd = exec.Command("open", absPath)
	default:
		cmd = exec.Command("xdg-open", absPath)
	}
	cmd.SysProcAttr = sysproc.Hidden()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %q: %w", absPath, err)
	}
	return cmd.Process.Release()
}
