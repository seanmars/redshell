package updater

import (
	"strings"

	"golang.org/x/mod/semver"
)

func canonicalSemver(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

func IsValidVersion(v string) bool {
	return semver.IsValid(canonicalSemver(v))
}

func Compare(a, b string) int {
	ca, cb := canonicalSemver(a), canonicalSemver(b)
	if !semver.IsValid(ca) || !semver.IsValid(cb) {
		switch {
		case !semver.IsValid(ca) && !semver.IsValid(cb):
			return 0
		case !semver.IsValid(ca):
			return -1
		default:
			return 1
		}
	}
	return semver.Compare(ca, cb)
}
