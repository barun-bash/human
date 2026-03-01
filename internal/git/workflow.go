// Package git provides Git workflow commands integrated with Human project conventions.
package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Feature creates a feature branch following Human conventions.
// Branch name: feature/<kebab-case-name>
func Feature(name string) error {
	if name == "" {
		return fmt.Errorf("feature name is required")
	}

	if err := checkGitRepo(); err != nil {
		return err
	}

	branch := "feature/" + toKebabCase(name)

	if dirty, err := IsWorkingTreeDirty(); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("working tree has uncommitted changes — commit or stash first")
	}

	defaultBranch, err := DefaultBranch()
	if err != nil {
		return fmt.Errorf("could not detect default branch: %w", err)
	}

	// Fetch latest default branch (ignore error if no remote)
	_ = runQuiet("git", "fetch", "origin", defaultBranch)

	// Create and switch to feature branch — try origin/<default> first, fall back to local
	base := "origin/" + defaultBranch
	if err := runQuiet("git", "rev-parse", "--verify", base); err != nil {
		base = defaultBranch // no remote, use local branch
	}
	if err := runQuiet("git", "checkout", "-b", branch, base); err != nil {
		return fmt.Errorf("creating branch %s: %w", branch, err)
	}

	return nil
}

// FeatureFinish merges the current feature branch back to the default branch.
func FeatureFinish(dryRun bool) error {
	if err := checkGitRepo(); err != nil {
		return err
	}

	branch, err := CurrentBranch()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(branch, "feature/") {
		return fmt.Errorf("not on a feature branch (current: %s)", branch)
	}

	defaultBranch, err := DefaultBranch()
	if err != nil {
		return fmt.Errorf("could not detect default branch: %w", err)
	}

	if dryRun {
		fmt.Printf("Would merge %s into %s and delete branch\n", branch, defaultBranch)
		return nil
	}

	// Switch to default branch and pull
	if err := runQuiet("git", "checkout", defaultBranch); err != nil {
		return fmt.Errorf("switching to %s: %w", defaultBranch, err)
	}
	_ = runQuiet("git", "pull", "origin", defaultBranch)

	// Merge feature branch with no-ff and explicit message (avoids opening editor)
	msg := fmt.Sprintf("Merge branch '%s'", branch)
	if err := runQuiet("git", "merge", "--no-ff", "-m", msg, branch); err != nil {
		// Merge failed (likely conflict) — abort and return to feature branch
		_ = runQuiet("git", "merge", "--abort")
		_ = runQuiet("git", "checkout", branch)
		return fmt.Errorf("merge conflict — resolve manually with: git checkout %s && git merge --no-ff %s", defaultBranch, branch)
	}

	// Only delete after successful merge
	if err := runQuiet("git", "branch", "-d", branch); err != nil {
		return fmt.Errorf("deleting branch %s: %w", branch, err)
	}

	return nil
}

// Release tags a release with an annotated tag.
func Release(version string, dryRun bool) error {
	if err := checkGitRepo(); err != nil {
		return err
	}

	if !IsValidVersion(version) {
		return fmt.Errorf("invalid version %q (expected vX.Y.Z or X.Y.Z)", version)
	}

	// Normalize to v-prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	branch, err := CurrentBranch()
	if err != nil {
		return err
	}

	defaultBranch, err := DefaultBranch()
	if err != nil {
		return fmt.Errorf("could not detect default branch: %w", err)
	}

	if branch != defaultBranch {
		return fmt.Errorf("releases must be tagged on %s (current: %s)", defaultBranch, branch)
	}

	if dirty, err := IsWorkingTreeDirty(); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("working tree has uncommitted changes — commit first")
	}

	// Check tag doesn't exist
	if err := runQuiet("git", "rev-parse", version); err == nil {
		return fmt.Errorf("tag %s already exists", version)
	}

	if dryRun {
		fmt.Printf("Would create annotated tag %s on %s\n", version, defaultBranch)
		return nil
	}

	msg := fmt.Sprintf("Release %s", version)
	if err := runQuiet("git", "tag", "-a", version, "-m", msg); err != nil {
		return fmt.Errorf("creating tag: %w", err)
	}

	return nil
}

// ReleaseNotes generates a changelog from git commits since the last tag.
func ReleaseNotes() (string, error) {
	if err := checkGitRepo(); err != nil {
		return "", err
	}

	// Find last tag
	lastTag, err := runOutput("git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		// No tags — use all commits
		lastTag = ""
	}

	// Get commits since last tag
	var logArgs []string
	if lastTag != "" {
		logArgs = []string{"log", lastTag + "..HEAD", "--oneline", "--no-merges"}
	} else {
		logArgs = []string{"log", "--oneline", "--no-merges"}
	}
	raw, err := runOutput("git", logArgs...)
	if err != nil {
		return "", fmt.Errorf("reading git log: %w", err)
	}

	if strings.TrimSpace(raw) == "" {
		return "No changes since last release.\n", nil
	}

	// Group by conventional commit type
	groups := map[string][]string{
		"Features":      {},
		"Bug Fixes":     {},
		"Documentation": {},
		"Refactoring":   {},
		"Other Changes": {},
	}
	order := []string{"Features", "Bug Fixes", "Documentation", "Refactoring", "Other Changes"}

	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		// Strip commit hash prefix
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		msg := parts[1]

		switch {
		case strings.HasPrefix(msg, "feat:") || strings.HasPrefix(msg, "feat("):
			groups["Features"] = append(groups["Features"], cleanCommitMsg(msg))
		case strings.HasPrefix(msg, "fix:") || strings.HasPrefix(msg, "fix("):
			groups["Bug Fixes"] = append(groups["Bug Fixes"], cleanCommitMsg(msg))
		case strings.HasPrefix(msg, "docs:") || strings.HasPrefix(msg, "docs("):
			groups["Documentation"] = append(groups["Documentation"], cleanCommitMsg(msg))
		case strings.HasPrefix(msg, "refactor:") || strings.HasPrefix(msg, "refactor("):
			groups["Refactoring"] = append(groups["Refactoring"], cleanCommitMsg(msg))
		default:
			groups["Other Changes"] = append(groups["Other Changes"], msg)
		}
	}

	// Format as Markdown
	var sb strings.Builder
	date := time.Now().Format("2006-01-02")
	if lastTag != "" {
		sb.WriteString(fmt.Sprintf("# Changelog (since %s) — %s\n\n", lastTag, date))
	} else {
		sb.WriteString(fmt.Sprintf("# Changelog — %s\n\n", date))
	}

	for _, section := range order {
		items := groups[section]
		if len(items) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n", section))
		sort.Strings(items)
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// ── Helpers ──

func checkGitRepo() error {
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return fmt.Errorf("not a git repository")
	}
	return nil
}

// CurrentBranch returns the current git branch name.
func CurrentBranch() (string, error) {
	return runOutput("git", "rev-parse", "--abbrev-ref", "HEAD")
}

// DefaultBranch detects the default branch (main or master).
func DefaultBranch() (string, error) {
	// Try symbolic ref first (works if remote is set up)
	ref, err := runOutput("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		// refs/remotes/origin/main → main
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fall back: check if main or master exists
	if err := runQuiet("git", "rev-parse", "--verify", "refs/heads/main"); err == nil {
		return "main", nil
	}
	if err := runQuiet("git", "rev-parse", "--verify", "refs/heads/master"); err == nil {
		return "master", nil
	}

	return "main", nil // default assumption
}

// IsWorkingTreeDirty checks if the working tree has uncommitted changes.
func IsWorkingTreeDirty() (bool, error) {
	out, err := runOutput("git", "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// IsValidVersion checks if a string is a valid semver tag (vX.Y.Z or X.Y.Z).
func IsValidVersion(v string) bool {
	v = strings.TrimPrefix(v, "v")
	return regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(v)
}

// runQuiet executes a git command silently, returning only an error.
func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// runOutput executes a git command and returns its trimmed stdout.
func runOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

func toKebabCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '-')
		}
		if r == ' ' || r == '_' {
			result = append(result, '-')
		} else {
			result = append(result, unicode.ToLower(r))
		}
	}
	// Collapse multiple hyphens
	out := string(result)
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}

func cleanCommitMsg(msg string) string {
	// Remove prefix like "feat: " or "fix(scope): "
	idx := strings.Index(msg, ":")
	if idx >= 0 && idx < len(msg)-1 {
		msg = strings.TrimSpace(msg[idx+1:])
	}
	return msg
}
