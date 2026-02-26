package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  SemVer
		err   bool
	}{
		{"0.4.0", SemVer{0, 4, 0}, false},
		{"1.2.3", SemVer{1, 2, 3}, false},
		{"v0.4.0", SemVer{0, 4, 0}, false},
		{"v1.0.0-beta", SemVer{1, 0, 0}, false},
		{"10.20.30", SemVer{10, 20, 30}, false},
		{"bad", SemVer{}, true},
		{"1.2", SemVer{}, true},
		{"1.2.x", SemVer{}, true},
		{"", SemVer{}, true},
	}

	for _, tt := range tests {
		got, err := Parse(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("Parse(%q) expected error, got %v", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSemVer_String(t *testing.T) {
	v := SemVer{1, 2, 3}
	if s := v.String(); s != "1.2.3" {
		t.Errorf("String() = %q, want %q", s, "1.2.3")
	}
}

func TestSemVer_Compare(t *testing.T) {
	tests := []struct {
		a, b SemVer
		want int
	}{
		{SemVer{1, 0, 0}, SemVer{1, 0, 0}, 0},
		{SemVer{2, 0, 0}, SemVer{1, 0, 0}, 1},
		{SemVer{1, 0, 0}, SemVer{2, 0, 0}, -1},
		{SemVer{1, 2, 0}, SemVer{1, 1, 0}, 1},
		{SemVer{1, 1, 0}, SemVer{1, 2, 0}, -1},
		{SemVer{1, 1, 2}, SemVer{1, 1, 1}, 1},
		{SemVer{1, 1, 1}, SemVer{1, 1, 2}, -1},
		{SemVer{0, 4, 0}, SemVer{0, 4, 1}, -1},
	}

	for _, tt := range tests {
		got := tt.a.Compare(tt.b)
		if got != tt.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsNewerThan(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"0.5.0", "0.4.0", true},
		{"0.4.0", "0.4.0", false},
		{"0.3.0", "0.4.0", false},
		{"v1.0.0", "0.4.0", true},
		{"v0.4.1", "v0.4.0", true},
		{"bad", "0.4.0", false},
		{"0.4.0", "bad", false},
	}

	for _, tt := range tests {
		got := IsNewerThan(tt.latest, tt.current)
		if got != tt.want {
			t.Errorf("IsNewerThan(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestInfo_Dev(t *testing.T) {
	// Save and restore package-level vars.
	origSHA, origDate := CommitSHA, BuildDate
	defer func() { CommitSHA, BuildDate = origSHA, origDate }()

	CommitSHA = "dev"
	BuildDate = "unknown"
	if info := Info(); info != Version {
		t.Errorf("Info() = %q, want %q", info, Version)
	}
}

func TestInfo_Release(t *testing.T) {
	origSHA, origDate := CommitSHA, BuildDate
	defer func() { CommitSHA, BuildDate = origSHA, origDate }()

	CommitSHA = "abc1234"
	BuildDate = "2026-02-26"
	want := Version + " (abc1234, 2026-02-26)"
	if info := Info(); info != want {
		t.Errorf("Info() = %q, want %q", info, want)
	}
}
