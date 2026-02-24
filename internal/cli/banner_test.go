package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildLogoLines(t *testing.T) {
	// Building with 1 letter should only show H.
	lines := buildLogoLines(1)
	for i, line := range lines {
		if line == "" {
			t.Errorf("row %d is empty for 1-letter logo", i)
		}
	}

	// Building all 5 letters should produce wider lines.
	full := buildLogoLines(5)
	for i := range full {
		if len(full[i]) <= len(lines[i]) {
			t.Errorf("full logo row %d should be wider than single-letter", i)
		}
	}
}

func TestPrintStaticLogo(t *testing.T) {
	var buf bytes.Buffer
	ColorEnabled = false

	printStaticLogo(&buf)

	output := buf.String()
	if !strings.Contains(output, "_") {
		t.Error("static logo should contain underscore")
	}

	// Should have 6 lines (the block art).
	lineCount := strings.Count(output, "\n")
	if lineCount != logoRows {
		t.Errorf("expected %d lines, got %d", logoRows, lineCount)
	}
}

func TestPrintBannerStatic(t *testing.T) {
	var buf bytes.Buffer
	ColorEnabled = false

	info := &BannerInfo{
		ProjectFile: "app.human",
		ProjectName: "myapp",
	}
	// animate=false so it always prints static.
	PrintBanner(&buf, "0.5.0", false, info)

	output := buf.String()
	if !strings.Contains(output, "0.5.0") {
		t.Error("banner should contain version")
	}
	if !strings.Contains(output, "myapp") {
		t.Error("banner should contain project name")
	}
	if !strings.Contains(output, "app.human") {
		t.Error("banner should contain project file")
	}
	if !strings.Contains(output, "Tip:") {
		t.Error("banner should contain a tip")
	}
}

func TestPrintBannerNoProject(t *testing.T) {
	var buf bytes.Buffer
	ColorEnabled = false

	PrintBanner(&buf, "0.5.0", false, &BannerInfo{})

	output := buf.String()
	if !strings.Contains(output, "No project") {
		t.Error("banner should indicate no project loaded")
	}
}

func TestPrintBannerFirstRun(t *testing.T) {
	var buf bytes.Buffer
	ColorEnabled = false

	info := &BannerInfo{FirstRun: true}
	PrintBanner(&buf, "0.5.0", false, info)

	output := buf.String()
	if !strings.Contains(output, "Welcome to Human") {
		t.Error("first-run banner should contain welcome message")
	}
	if !strings.Contains(output, "/connect") {
		t.Error("first-run banner should mention /connect")
	}
	if !strings.Contains(output, "/new") {
		t.Error("first-run banner should mention /new")
	}
}

func TestPrintBannerNilInfo(t *testing.T) {
	var buf bytes.Buffer
	ColorEnabled = false

	// Should not panic with nil info.
	PrintBanner(&buf, "0.5.0", false, nil)

	output := buf.String()
	if !strings.Contains(output, "Version:") {
		t.Error("banner should contain version label")
	}
}

func TestPrintBannerLLMStatus(t *testing.T) {
	var buf bytes.Buffer
	ColorEnabled = false

	info := &BannerInfo{LLMStatus: "anthropic (claude-sonnet-4.5)"}
	PrintBanner(&buf, "0.5.0", false, info)

	output := buf.String()
	if !strings.Contains(output, "anthropic") {
		t.Error("banner should show LLM status")
	}
}

func TestRandomTipNotEmpty(t *testing.T) {
	tip := RandomTip()
	if tip == "" {
		t.Error("RandomTip should not return empty string")
	}
}

func TestIsTTYBuffer(t *testing.T) {
	var buf bytes.Buffer
	if isTTY(&buf) {
		t.Error("bytes.Buffer should not be a TTY")
	}
}

func TestAnimateSkippedForNonTTY(t *testing.T) {
	var buf bytes.Buffer
	ColorEnabled = false

	// animate=true but writer is not a TTY, so static logo should be printed.
	PrintBanner(&buf, "0.5.0", true, &BannerInfo{})

	output := buf.String()
	// Static logo should be present (no ANSI escape sequences for cursor movement).
	if strings.Contains(output, "\033[2J") {
		t.Error("non-TTY output should not contain screen clear escape")
	}
	if !strings.Contains(output, "_") {
		t.Error("non-TTY output should still contain underscore")
	}
}
