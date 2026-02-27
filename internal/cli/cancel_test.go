package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestSetupSignalHandler(t *testing.T) {
	ctx, cancel := SetupSignalHandler()
	defer cancel()

	// Context should be active.
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled yet")
	default:
	}

	// Cancel manually.
	cancel()
	<-ctx.Done()
}

func TestRunCancellableSuccess(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	err := RunCancellable(ctx, &buf, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRunCancellableError(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	want := errors.New("fail")
	err := RunCancellable(ctx, &buf, func(ctx context.Context) error {
		return want
	})
	if err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestRunCancellableContextCancel(t *testing.T) {
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	err := RunCancellable(ctx, &buf, func(ctx context.Context) error {
		cancel()
		<-ctx.Done()
		return ctx.Err()
	})
	// Graceful cancellation returns nil.
	if err != nil {
		t.Fatalf("expected nil on cancel, got %v", err)
	}
}

func TestCancelled(t *testing.T) {
	var buf bytes.Buffer
	Cancelled(&buf)
	out := buf.String()
	if out == "" {
		t.Error("expected output from Cancelled")
	}
}
