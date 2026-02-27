package editor

import (
	"sync"
	"time"

	"github.com/barun-bash/human/internal/llm"
)

// Validator runs background validation of the buffer content.
type Validator struct {
	mu       sync.Mutex
	timer    *time.Timer
	lastErr  string
	debounce time.Duration
	notify   func() // called when validation result changes
}

// NewValidator creates a validator with the given debounce duration.
func NewValidator(debounce time.Duration, notify func()) *Validator {
	return &Validator{
		debounce: debounce,
		notify:   notify,
	}
}

// Schedule queues a validation run. Debounces rapid calls.
func (v *Validator) Schedule(content string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.timer != nil {
		v.timer.Stop()
	}

	v.timer = time.AfterFunc(v.debounce, func() {
		valid, parseErr := llm.ValidateCode(content)
		v.mu.Lock()
		if valid {
			v.lastErr = ""
		} else {
			v.lastErr = parseErr
		}
		v.mu.Unlock()

		if v.notify != nil {
			v.notify()
		}
	})
}

// Error returns the last validation error (empty if valid).
func (v *Validator) Error() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.lastErr
}

// Stop cancels any pending validation.
func (v *Validator) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.timer != nil {
		v.timer.Stop()
	}
}
