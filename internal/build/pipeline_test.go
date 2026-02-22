package build

import "testing"

func TestMatchesGoBackend(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Go", true},
		{"go", true},
		{"Go with Gin", true},
		{"go with fiber", true},
		{"golang", true},
		{"Gin", true},
		{"Node", false},
		{"Python", false},
		{"django", false},
		{"mongodb", false},
		{"", false},
	}

	for _, tt := range tests {
		got := MatchesGoBackend(tt.input)
		if got != tt.want {
			t.Errorf("MatchesGoBackend(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCountFilesEmpty(t *testing.T) {
	dir := t.TempDir()
	count := CountFiles(dir)
	if count != 0 {
		t.Errorf("CountFiles(empty dir) = %d, want 0", count)
	}
}

func TestCountFilesNonExistent(t *testing.T) {
	count := CountFiles("/nonexistent/path/that/does/not/exist")
	if count != 0 {
		t.Errorf("CountFiles(nonexistent) = %d, want 0", count)
	}
}
