package gesture

import (
	"time"
)

// buttonState tracks the state of a single button for gesture detection
type buttonState struct {
	button          int
	pressTime       time.Time
	releaseTime     time.Time
	isPressed       bool
	waitingDouble   bool
	inDoublePress   bool // True when we're in the second press of a double-press
	longPressTimer  *time.Timer
	doubleTimer     *time.Timer
}

// Detector handles timing-based gesture detection for individual buttons
type Detector struct {
	doublePressWindow  time.Duration
	longPressThreshold time.Duration
	onGesture          func(Gesture)
	states             map[int]*buttonState
}

// NewDetector creates a new gesture detector
func NewDetector(doublePressWindowMs, longPressThresholdMs int, onGesture func(Gesture)) *Detector {
	return &Detector{
		doublePressWindow:  time.Duration(doublePressWindowMs) * time.Millisecond,
		longPressThreshold: time.Duration(longPressThresholdMs) * time.Millisecond,
		onGesture:          onGesture,
		states:             make(map[int]*buttonState),
	}
}

func (d *Detector) getState(button int) *buttonState {
	if s, ok := d.states[button]; ok {
		return s
	}
	s := &buttonState{button: button}
	d.states[button] = s
	return s
}

// HandlePress processes a button press event
func (d *Detector) HandlePress(button int) {
	state := d.getState(button)
	now := time.Now()

	// If we were waiting for a double press and got one in time
	if state.waitingDouble {
		if state.doubleTimer != nil {
			state.doubleTimer.Stop()
			state.doubleTimer = nil
		}
		state.waitingDouble = false
		state.inDoublePress = true // Mark that this is the second press
		state.isPressed = true
		state.pressTime = now
		// Don't start long press timer for second press of double-press
		// Emit double press on release
		return
	}

	state.isPressed = true
	state.pressTime = now

	// Start long press timer
	state.longPressTimer = time.AfterFunc(d.longPressThreshold, func() {
		// Only emit if still pressed
		if state.isPressed {
			d.onGesture(NewLongPressGesture(button))
			// Mark that we've handled this press as long press
			state.longPressTimer = nil
		}
	})
}

// HandleRelease processes a button release event
func (d *Detector) HandleRelease(button int) {
	state := d.getState(button)
	now := time.Now()

	if !state.isPressed {
		return
	}

	state.isPressed = false
	state.releaseTime = now
	pressDuration := now.Sub(state.pressTime)

	// Stop long press timer if running
	if state.longPressTimer != nil {
		state.longPressTimer.Stop()
		state.longPressTimer = nil
	}

	// If this was a long press, it was already emitted
	if pressDuration >= d.longPressThreshold {
		return
	}

	// If this is the release of the second press of a double-press
	if state.inDoublePress {
		state.inDoublePress = false
		d.onGesture(NewDoublePressGesture(button))
		return
	}

	// Start waiting for potential double press
	state.waitingDouble = true
	state.doubleTimer = time.AfterFunc(d.doublePressWindow, func() {
		// Timeout - emit single press
		if state.waitingDouble {
			state.waitingDouble = false
			d.onGesture(NewPressGesture(button))
		}
	})
}

// Stop stops all pending timers
func (d *Detector) Stop() {
	for _, state := range d.states {
		if state.longPressTimer != nil {
			state.longPressTimer.Stop()
		}
		if state.doubleTimer != nil {
			state.doubleTimer.Stop()
		}
	}
}
