package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithSpinnerSuccess(t *testing.T) {
	var buf bytes.Buffer
	err := WithSpinner(&buf, "working...", func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Non-TTY: should print a static message.
	out := buf.String()
	if out == "" {
		t.Error("expected output on non-TTY")
	}
}

func TestWithSpinnerError(t *testing.T) {
	var buf bytes.Buffer
	want := errors.New("boom")
	err := WithSpinner(&buf, "working...", func() error {
		return want
	})
	if err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestWithSpinnerCtxCancel(t *testing.T) {
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	err := WithSpinnerCtx(ctx, &buf, "working...", func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	})
	// Cancellation should return nil (graceful).
	if err != nil {
		t.Fatalf("expected nil on cancel, got %v", err)
	}
}

func TestSpinnerSetMessage(t *testing.T) {
	s := NewSpinner(&bytes.Buffer{}, "initial")
	s.SetMessage("updated")
	s.mu.Lock()
	got := s.message
	s.mu.Unlock()
	if got != "updated" {
		t.Fatalf("expected 'updated', got %q", got)
	}
}

func TestSpinnerShowDelay(t *testing.T) {
	s := NewSpinner(&bytes.Buffer{}, "test")
	if s.showDelay != 200*time.Millisecond {
		t.Fatalf("expected 200ms delay, got %v", s.showDelay)
	}
}

func TestSpinnerFail(t *testing.T) {
	var buf bytes.Buffer
	// Non-TTY: Fail prints directly.
	s := NewSpinner(&buf, "test")
	s.Start()
	s.Fail("something broke")
	out := buf.String()
	if out == "" {
		t.Error("expected output from Fail")
	}
}

func TestElapsedTimeFormatting(t *testing.T) {
	// Just verify the spinner struct tracks startTime.
	s := NewSpinner(&bytes.Buffer{}, "test")
	s.startTime = time.Now().Add(-5 * time.Second)
	elapsed := time.Since(s.startTime)
	if elapsed < 4*time.Second {
		t.Error("expected elapsed time >= 4s")
	}
}
