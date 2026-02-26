package repl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/version"
)

// UpdateInfo holds the result of a background version check.
type UpdateInfo struct {
	Available     bool
	LatestVersion string
	CurrentVersion string
}

// githubAPIURL is the endpoint for fetching the latest release.
// Package-level var so tests can override it with httptest.NewServer.
var githubAPIURL = "https://api.github.com/repos/barun-bash/human/releases/latest"

// updateCheckInterval is the minimum time between GitHub API checks.
const updateCheckInterval = 24 * time.Hour

// checkUpdateBackground starts a goroutine that checks for a newer version.
// Results are stored in r.updateInfo; r.updateDone is closed when complete.
func (r *REPL) checkUpdateBackground() {
	r.updateDone = make(chan struct{})

	go func() {
		defer close(r.updateDone)

		info := checkForUpdate(r.settings)

		r.updateMu.Lock()
		r.updateInfo = info
		r.updateMu.Unlock()
	}()
}

// showUpdateNotification waits briefly for the background check and displays
// a notification if an update is available. If the check hasn't finished in
// 500ms, it skips (the cached result will be shown next time).
func (r *REPL) showUpdateNotification() {
	if r.updateDone == nil {
		return
	}

	select {
	case <-r.updateDone:
		// Check finished in time.
	case <-time.After(500 * time.Millisecond):
		// Too slow — skip this time.
		return
	}

	r.updateMu.Lock()
	info := r.updateInfo
	r.updateMu.Unlock()

	if info == nil || !info.Available {
		return
	}

	fmt.Fprintln(r.out)
	fmt.Fprintf(r.out, "  %s Update available: %s %s %s — run %s to upgrade\n",
		cli.Warn(""),
		info.CurrentVersion,
		cli.Muted("→"),
		cli.Accent(info.LatestVersion),
		cli.Accent("/update"),
	)
}

// checkForUpdate checks whether a newer version exists. It uses a 24h cache
// stored in GlobalSettings to avoid hitting the GitHub API on every startup.
func checkForUpdate(settings *config.GlobalSettings) *UpdateInfo {
	current := version.Version

	// Check cache first.
	if settings.LastUpdateCheck != "" && settings.LatestVersion != "" {
		lastCheck, err := time.Parse(time.RFC3339, settings.LastUpdateCheck)
		if err == nil && time.Since(lastCheck) < updateCheckInterval {
			return &UpdateInfo{
				Available:      version.IsNewerThan(settings.LatestVersion, current),
				LatestVersion:  settings.LatestVersion,
				CurrentVersion: current,
			}
		}
	}

	// Fetch from GitHub.
	latest, err := fetchLatestRelease()
	if err != nil {
		return nil
	}

	// Update cache.
	settings.LastUpdateCheck = time.Now().UTC().Format(time.RFC3339)
	settings.LatestVersion = latest
	_ = config.SaveGlobal(settings) // non-fatal

	return &UpdateInfo{
		Available:      version.IsNewerThan(latest, current),
		LatestVersion:  latest,
		CurrentVersion: current,
	}
}

// githubRelease is the subset of the GitHub API response we need.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// fetchLatestRelease queries the GitHub releases API for the latest tag.
func fetchLatestRelease() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "human-cli/"+version.Version)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	tag := strings.TrimPrefix(release.TagName, "v")
	if tag == "" {
		return "", fmt.Errorf("empty tag_name in GitHub response")
	}

	return tag, nil
}

// cmdUpdate handles the /update REPL command.
func cmdUpdate(r *REPL, args []string) {
	fmt.Fprintln(r.out, cli.Info("Checking for updates..."))

	// Force-fetch (bypass cache).
	latest, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not check for updates: %v", err)))
		return
	}

	current := version.Version
	if !version.IsNewerThan(latest, current) {
		fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("You're on the latest version (%s).", current)))
		return
	}

	fmt.Fprintf(r.out, "  Update available: %s → %s\n\n", current, cli.Accent(latest))

	// Detect install method.
	method := detectInstallMethod(r.settings)

	switch method {
	case "source":
		updateFromSource(r)
	case "go_install":
		updateViaGoInstall(r)
	default:
		showManualUpdateInstructions(r, latest)
	}
}

// detectInstallMethod figures out how the CLI was installed.
func detectInstallMethod(settings *config.GlobalSettings) string {
	// Check saved preference first.
	if settings.InstallMethod != "" {
		return settings.InstallMethod
	}

	// Check if we're in a git repo with the right remote.
	if settings.SourceDir != "" {
		if _, err := os.Stat(filepath.Join(settings.SourceDir, ".git")); err == nil {
			return "source"
		}
	}

	// Check if the binary is in GOPATH/bin or GOBIN.
	exe, err := os.Executable()
	if err == nil {
		gobin := os.Getenv("GOBIN")
		if gobin == "" {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				home, _ := os.UserHomeDir()
				gopath = filepath.Join(home, "go")
			}
			gobin = filepath.Join(gopath, "bin")
		}
		if strings.HasPrefix(exe, gobin) {
			return "go_install"
		}
	}

	return "binary"
}

// updateFromSource runs git pull + make build + make install in the source directory.
func updateFromSource(r *REPL) {
	srcDir := r.settings.SourceDir
	if srcDir == "" {
		fmt.Fprintln(r.errOut, cli.Error("Source directory not configured. Set it with /config set source_dir <path>"))
		return
	}

	fmt.Fprintln(r.out, cli.Info("Updating from source..."))

	// git pull --ff-only
	fmt.Fprintln(r.out, cli.Info("  Step 1/3: git pull"))
	cmd := exec.Command("git", "pull", "--ff-only")
	cmd.Dir = srcDir
	cmd.Stdout = r.out
	cmd.Stderr = r.errOut
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("git pull failed: %v", err)))
		fmt.Fprintln(r.errOut, cli.Info("Try updating manually: cd "+srcDir+" && git pull && make build && make install"))
		return
	}

	// make build
	fmt.Fprintln(r.out, cli.Info("  Step 2/3: make build"))
	cmd = exec.Command("make", "build")
	cmd.Dir = srcDir
	cmd.Stdout = r.out
	cmd.Stderr = r.errOut
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("make build failed: %v", err)))
		return
	}

	// make install
	fmt.Fprintln(r.out, cli.Info("  Step 3/3: make install"))
	cmd = exec.Command("make", "install")
	cmd.Dir = srcDir
	cmd.Stdout = r.out
	cmd.Stderr = r.errOut
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("make install failed: %v", err)))
		fmt.Fprintln(r.errOut, cli.Info("You may need sudo: cd "+srcDir+" && sudo make install"))
		return
	}

	fmt.Fprintln(r.out, cli.Success("Updated successfully. Restart the REPL to use the new version."))
}

// updateViaGoInstall runs go install for the latest version.
func updateViaGoInstall(r *REPL) {
	fmt.Fprintln(r.out, cli.Info("Updating via go install..."))

	cmd := exec.Command("go", "install", "github.com/barun-bash/human/cmd/human@latest")
	cmd.Stdout = r.out
	cmd.Stderr = r.errOut
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("go install failed: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success("Updated successfully. Restart the REPL to use the new version."))
}

// showManualUpdateInstructions displays how to update manually.
func showManualUpdateInstructions(r *REPL, latest string) {
	fmt.Fprintln(r.out, cli.Info("To update, choose one of:"))
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "  From source:")
	fmt.Fprintln(r.out, "    git clone https://github.com/barun-bash/human && cd human")
	fmt.Fprintln(r.out, "    make build && make install")
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "  Via go install:")
	fmt.Fprintln(r.out, "    go install github.com/barun-bash/human/cmd/human@latest")
	fmt.Fprintln(r.out)
	fmt.Fprintf(r.out, "  Download binary: %s\n", cli.Muted(fmt.Sprintf("https://github.com/barun-bash/human/releases/tag/v%s", latest)))
}

// updateMu and updateInfo/updateDone fields are added to the REPL struct.
// This file uses them but they're declared in repl.go.

// Ensure the mutex type is available.
var _ sync.Mutex
