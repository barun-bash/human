package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version, CommitSHA, and BuildDate are set via ldflags at build time.
// Example: go build -ldflags "-X .../version.Version=1.0.0 -X .../version.CommitSHA=abc1234 -X .../version.BuildDate=2026-02-26"
var (
	Version  = "0.4.0"
	CommitSHA = "dev"
	BuildDate = "unknown"
)

// Info returns a human-readable version string.
// For dev builds: "0.4.0"
// For release builds: "0.4.0 (abc1234, 2026-02-26)"
func Info() string {
	v := strings.TrimPrefix(Version, "v")
	if CommitSHA == "dev" || CommitSHA == "" {
		return v
	}
	return fmt.Sprintf("%s (%s, %s)", v, CommitSHA, BuildDate)
}

// SemVer represents a parsed semantic version (major.minor.patch).
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// Parse parses a version string like "0.4.0" or "v0.4.0" into a SemVer.
// Returns an error if the string is not a valid semver.
func Parse(s string) (SemVer, error) {
	s = strings.TrimPrefix(s, "v")

	parts := strings.SplitN(s, "-", 2) // strip pre-release suffix
	s = parts[0]

	segments := strings.Split(s, ".")
	if len(segments) != 3 {
		return SemVer{}, fmt.Errorf("invalid version %q: expected major.minor.patch", s)
	}

	major, err := strconv.Atoi(segments[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version %q: %w", segments[0], err)
	}
	minor, err := strconv.Atoi(segments[1])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid minor version %q: %w", segments[1], err)
	}
	patch, err := strconv.Atoi(segments[2])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid patch version %q: %w", segments[2], err)
	}

	return SemVer{Major: major, Minor: minor, Patch: patch}, nil
}

// String returns the version as "major.minor.patch".
func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1, 0, or 1 depending on whether v is less than, equal to,
// or greater than other.
func (v SemVer) Compare(other SemVer) int {
	if v.Major != other.Major {
		return cmpInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return cmpInt(v.Minor, other.Minor)
	}
	return cmpInt(v.Patch, other.Patch)
}

// IsNewerThan returns true if latest is a newer version than current.
// Returns false on parse errors or if versions are equal.
func IsNewerThan(latest, current string) bool {
	l, err := Parse(latest)
	if err != nil {
		return false
	}
	c, err := Parse(current)
	if err != nil {
		return false
	}
	return l.Compare(c) > 0
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
