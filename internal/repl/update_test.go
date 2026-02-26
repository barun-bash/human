package repl

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/version"
)

func TestFetchLatestRelease(t *testing.T) {
	// Set up a fake GitHub API server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"tag_name": "v99.0.0"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Override the API URL.
	orig := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = orig }()

	tag, err := fetchLatestRelease()
	if err != nil {
		t.Fatalf("fetchLatestRelease() error: %v", err)
	}
	if tag != "99.0.0" {
		t.Errorf("fetchLatestRelease() = %q, want %q", tag, "99.0.0")
	}
}

func TestFetchLatestRelease_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	orig := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = orig }()

	_, err := fetchLatestRelease()
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestCheckForUpdate_Cached(t *testing.T) {
	settings := &config.GlobalSettings{
		LastUpdateCheck: "2099-01-01T00:00:00Z",
		LatestVersion:   "99.0.0",
	}

	info := checkForUpdate(settings)
	if info == nil {
		t.Fatal("expected non-nil info from cache")
	}
	if !info.Available {
		t.Error("expected update to be available")
	}
	if info.LatestVersion != "99.0.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "99.0.0")
	}
}

func TestCheckForUpdate_CacheNotAvailable(t *testing.T) {
	settings := &config.GlobalSettings{
		LastUpdateCheck: "2099-01-01T00:00:00Z",
		LatestVersion:   "0.0.1",
	}

	info := checkForUpdate(settings)
	if info == nil {
		t.Fatal("expected non-nil info from cache")
	}
	if info.Available {
		t.Error("expected update not to be available when cached version is older")
	}
}

func TestDetectInstallMethod_Default(t *testing.T) {
	settings := &config.GlobalSettings{}
	method := detectInstallMethod(settings)
	// Without any hints, falls through to "binary" (or "go_install" if
	// running from GOPATH). Just check it doesn't panic.
	if method != "binary" && method != "go_install" {
		t.Errorf("unexpected method: %q", method)
	}
}

func TestDetectInstallMethod_Saved(t *testing.T) {
	settings := &config.GlobalSettings{InstallMethod: "source"}
	method := detectInstallMethod(settings)
	if method != "source" {
		t.Errorf("method = %q, want %q", method, "source")
	}
}

func TestShowUpdateNotification_NoUpdate(t *testing.T) {
	cli.ColorEnabled = false
	out := &bytes.Buffer{}
	r := &REPL{
		out:    out,
		errOut: &bytes.Buffer{},
	}
	r.updateDone = make(chan struct{})
	r.updateInfo = &UpdateInfo{Available: false}
	close(r.updateDone)

	r.showUpdateNotification()
	if out.Len() != 0 {
		t.Errorf("expected no output when no update available, got: %s", out.String())
	}
}

func TestShowUpdateNotification_UpdateAvailable(t *testing.T) {
	cli.ColorEnabled = false
	out := &bytes.Buffer{}
	r := &REPL{
		out:    out,
		errOut: &bytes.Buffer{},
	}
	r.updateDone = make(chan struct{})
	r.updateInfo = &UpdateInfo{
		Available:      true,
		LatestVersion:  "1.0.0",
		CurrentVersion: "0.4.0",
	}
	close(r.updateDone)

	r.showUpdateNotification()
	output := out.String()
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected notification to contain new version, got: %s", output)
	}
	if !strings.Contains(output, "/update") {
		t.Errorf("expected notification to mention /update, got: %s", output)
	}
}

func TestCmdUpdate_AlreadyCurrent(t *testing.T) {
	cli.ColorEnabled = false

	// Mock server that returns the current version.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"tag_name": "v" + version.Version}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	orig := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = orig }()

	out := &bytes.Buffer{}
	r := &REPL{
		out:      out,
		errOut:   &bytes.Buffer{},
		settings: &config.GlobalSettings{},
	}

	cmdUpdate(r, nil)
	output := out.String()
	if !strings.Contains(output, "latest version") {
		t.Errorf("expected 'latest version' message, got: %s", output)
	}
}

func TestCmdUpdate_NewVersionManual(t *testing.T) {
	cli.ColorEnabled = false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"tag_name": "v99.0.0"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	orig := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = orig }()

	out := &bytes.Buffer{}
	r := &REPL{
		out:      out,
		errOut:   &bytes.Buffer{},
		settings: &config.GlobalSettings{InstallMethod: "binary"},
	}

	cmdUpdate(r, nil)
	output := out.String()
	if !strings.Contains(output, "99.0.0") {
		t.Errorf("expected version in output, got: %s", output)
	}
	if !strings.Contains(output, "go install") {
		t.Errorf("expected manual instructions, got: %s", output)
	}
}
