package gesture

import (
	"sync"
	"testing"
	"time"
)

func TestDetectorSinglePress(t *testing.T) {
	var mu sync.Mutex
	var received []Gesture

	d := NewDetector(100, 200, func(g Gesture) {
		mu.Lock()
		received = append(received, g)
		mu.Unlock()
	})
	defer d.Stop()

	// Press and release quickly
	d.HandlePress(0)
	time.Sleep(10 * time.Millisecond)
	d.HandleRelease(0)

	// Wait for double-press window to expire
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("received %d gestures, want 1", len(received))
	}
	if received[0].Type != GesturePress {
		t.Errorf("gesture type = %v, want GesturePress", received[0].Type)
	}
	if len(received[0].Buttons) != 1 || received[0].Buttons[0] != 0 {
		t.Errorf("buttons = %v, want [0]", received[0].Buttons)
	}
}

func TestDetectorDoublePress(t *testing.T) {
	var mu sync.Mutex
	var received []Gesture

	d := NewDetector(200, 500, func(g Gesture) {
		mu.Lock()
		received = append(received, g)
		mu.Unlock()
	})
	defer d.Stop()

	// First press and release
	d.HandlePress(1)
	time.Sleep(10 * time.Millisecond)
	d.HandleRelease(1)

	// Second press and release within window
	time.Sleep(50 * time.Millisecond)
	d.HandlePress(1)
	time.Sleep(10 * time.Millisecond)
	d.HandleRelease(1)

	// Wait for any pending timers
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("received %d gestures, want 1", len(received))
	}
	if received[0].Type != GestureDoublePress {
		t.Errorf("gesture type = %v, want GestureDoublePress", received[0].Type)
	}
	if len(received[0].Buttons) != 1 || received[0].Buttons[0] != 1 {
		t.Errorf("buttons = %v, want [1]", received[0].Buttons)
	}
}

func TestDetectorLongPress(t *testing.T) {
	var mu sync.Mutex
	var received []Gesture

	d := NewDetector(100, 150, func(g Gesture) {
		mu.Lock()
		received = append(received, g)
		mu.Unlock()
	})
	defer d.Stop()

	// Press and hold
	d.HandlePress(2)
	time.Sleep(200 * time.Millisecond) // Longer than threshold
	d.HandleRelease(2)

	// Wait a bit for any async operations
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("received %d gestures, want 1", len(received))
	}
	if received[0].Type != GestureLongPress {
		t.Errorf("gesture type = %v, want GestureLongPress", received[0].Type)
	}
	if len(received[0].Buttons) != 1 || received[0].Buttons[0] != 2 {
		t.Errorf("buttons = %v, want [2]", received[0].Buttons)
	}
}

func TestDetectorMultipleButtons(t *testing.T) {
	var mu sync.Mutex
	var received []Gesture

	d := NewDetector(100, 500, func(g Gesture) {
		mu.Lock()
		received = append(received, g)
		mu.Unlock()
	})
	defer d.Stop()

	// Press and release button 0
	d.HandlePress(0)
	time.Sleep(10 * time.Millisecond)
	d.HandleRelease(0)

	// Press and release button 1 (separate gesture)
	time.Sleep(150 * time.Millisecond) // After double-press window
	d.HandlePress(1)
	time.Sleep(10 * time.Millisecond)
	d.HandleRelease(1)

	// Wait for timers
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Fatalf("received %d gestures, want 2", len(received))
	}
	if received[0].Buttons[0] != 0 {
		t.Errorf("first gesture button = %d, want 0", received[0].Buttons[0])
	}
	if received[1].Buttons[0] != 1 {
		t.Errorf("second gesture button = %d, want 1", received[1].Buttons[0])
	}
}

func TestDetectorStop(t *testing.T) {
	callCount := 0
	d := NewDetector(100, 200, func(g Gesture) {
		callCount++
	})

	// Start a long press
	d.HandlePress(0)

	// Stop immediately
	d.Stop()

	// Wait longer than long press threshold
	time.Sleep(300 * time.Millisecond)

	// Should not have received any gesture because timers were stopped
	if callCount != 0 {
		t.Errorf("received %d gestures after Stop(), want 0", callCount)
	}
}
