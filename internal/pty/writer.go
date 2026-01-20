package pty

import (
	"time"

	"github.com/mike/claude-pad/internal/action"
)

// Writer wraps a Manager to provide key writing with optional delay
type Writer struct {
	manager  *Manager
	keyDelay time.Duration
}

// NewWriter creates a new PTY writer
func NewWriter(manager *Manager, keyDelay time.Duration) *Writer {
	return &Writer{
		manager:  manager,
		keyDelay: keyDelay,
	}
}

// WriteKey writes a single key press to the PTY
func (w *Writer) WriteKey(key action.KeyPress) error {
	err := w.manager.WriteKey(key)
	if err != nil {
		return err
	}

	// Optional delay between keystrokes for TUIs that need it
	if w.keyDelay > 0 {
		time.Sleep(w.keyDelay)
	}

	return nil
}

// WriteKeys writes multiple key presses to the PTY
func (w *Writer) WriteKeys(keys []action.KeyPress) error {
	for _, key := range keys {
		if err := w.WriteKey(key); err != nil {
			return err
		}
	}
	return nil
}

// WriteString writes a string to the PTY
func (w *Writer) WriteString(s string) error {
	return w.manager.WriteString(s)
}
