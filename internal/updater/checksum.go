package updater

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func ParseChecksums(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("checksums line %d: expected '<hash>  <filename>', got %q", lineNo, line)
		}
		hash := strings.ToLower(fields[0])
		if !isHex64(hash) {
			return nil, fmt.Errorf("checksums line %d: %q is not a 64-char hex sha256", lineNo, fields[0])
		}
		name := strings.Join(fields[1:], " ")
		name = strings.TrimPrefix(name, "*")
		out[name] = hash
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}
	if len(out) == 0 {
		return nil, errors.New("checksums file contains no entries")
	}
	return out, nil
}

func isHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}
