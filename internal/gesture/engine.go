package gesture

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/mike/claude-pad/internal/config"
	"github.com/mike/claude-pad/internal/hid"
)

// Engine orchestrates gesture detection including chords
type Engine struct {
	timing       config.TimingConfig
	onGesture    func(Gesture)
	detector     *Detector
	chordWindow  time.Duration
	mu           sync.Mutex
	pressedBtns  map[int]time.Time // Button -> press timestamp
	chordPending bool
	chordTimer   *time.Timer
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewEngine creates a new gesture engine
func NewEngine(timing config.TimingConfig, onGesture func(Gesture)) *Engine {
	e := &Engine{
		timing:      timing,
		onGesture:   onGesture,
		chordWindow: time.Duration(timing.ChordWindowMs) * time.Millisecond,
		pressedBtns: make(map[int]time.Time),
	}

	// Create detector that routes single-button gestures through the engine
	e.detector = NewDetector(
		timing.DoublePressWindowMs,
		timing.LongPressThresholdMs,
		onGesture,
	)

	return e
}

// Start starts the gesture engine
func (e *Engine) Start(ctx context.Context) {
	e.ctx, e.cancel = context.WithCancel(ctx)
}

// Stop stops the gesture engine and cleans up resources
func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.chordTimer != nil {
		e.chordTimer.Stop()
	}
	e.detector.Stop()
}

// ProcessEvent processes a HID event and detects gestures
func (e *Engine) ProcessEvent(event hid.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	buttons := event.PressedButtons()

	switch event.Type {
	case hid.Press:
		e.handlePress(buttons)
	case hid.Release:
		e.handleRelease(buttons)
	}
}

func (e *Engine) handlePress(buttons []int) {
	now := time.Now()

	// Track all pressed buttons
	for _, btn := range buttons {
		if _, exists := e.pressedBtns[btn]; !exists {
			e.pressedBtns[btn] = now
		}
	}

	// Check for potential chord (multiple buttons pressed)
	if len(e.pressedBtns) > 1 {
		e.chordPending = true

		// Reset chord timer
		if e.chordTimer != nil {
			e.chordTimer.Stop()
		}

		// Wait for chord window to settle
		e.chordTimer = time.AfterFunc(e.chordWindow, func() {
			e.mu.Lock()
			defer e.mu.Unlock()
			e.checkChord()
		})
	} else if len(buttons) == 1 && len(e.pressedBtns) == 1 {
		// Single button press - start individual gesture detection
		// but wait for chord window first
		btn := buttons[0]
		time.AfterFunc(e.chordWindow, func() {
			e.mu.Lock()
			isChord := e.chordPending || len(e.pressedBtns) > 1
			e.mu.Unlock()

			if !isChord {
				e.detector.HandlePress(btn)
			}
		})
	}
}

func (e *Engine) handleRelease(buttons []int) {
	// Determine which buttons were released
	releasedBtns := e.findReleasedButtons(buttons)

	// If we were building a chord and buttons are released
	if e.chordPending && len(e.pressedBtns) > 0 {
		// Check if all chord buttons released
		if len(releasedBtns) > 0 {
			e.checkChord()
		}
	}

	// Update pressed state
	for _, btn := range releasedBtns {
		delete(e.pressedBtns, btn)
	}

	// Route single button releases to detector if not part of chord
	if !e.chordPending {
		for _, btn := range releasedBtns {
			e.detector.HandleRelease(btn)
		}
	}

	// Reset chord state when all buttons released
	if len(e.pressedBtns) == 0 {
		e.chordPending = false
	}
}

func (e *Engine) findReleasedButtons(currentPressed []int) []int {
	currentSet := make(map[int]bool)
	for _, btn := range currentPressed {
		currentSet[btn] = true
	}

	var released []int
	for btn := range e.pressedBtns {
		if !currentSet[btn] {
			released = append(released, btn)
		}
	}
	return released
}

func (e *Engine) checkChord() {
	if !e.chordPending {
		return
	}

	if len(e.pressedBtns) >= 2 {
		// Collect all pressed buttons
		buttons := make([]int, 0, len(e.pressedBtns))
		for btn := range e.pressedBtns {
			buttons = append(buttons, btn)
		}
		sort.Ints(buttons)

		// Emit chord gesture
		e.chordPending = false
		e.onGesture(NewChordGesture(buttons))

		// Clear pressed buttons since chord was handled
		e.pressedBtns = make(map[int]time.Time)
	}
}
