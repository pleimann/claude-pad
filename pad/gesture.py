"""
Gesture detection for button inputs.

Implements a state machine to detect single press, double press, and long press gestures.
"""
import time


class GestureState:
    """State machine states for gesture detection."""
    IDLE = 0
    PRESSED = 1
    WAIT_DOUBLE = 2
    DOUBLE_PRESSED = 3


class ButtonGestureDetector:
    """Detects single press, double press, and long press gestures for a button."""

    def __init__(self, button_index, config, keyboard, timing):
        self.button_index = button_index
        self.config = config
        self.keyboard = keyboard

        # Timing configuration (convert ms to seconds)
        self.double_press_window = timing["double_press_window_ms"] / 1000.0
        self.long_press_threshold = timing["long_press_threshold_ms"] / 1000.0

        # State machine
        self.state = GestureState.IDLE
        self.is_pressed = False
        self.press_time = 0
        self.release_time = 0
        self.long_press_fired = False

    def handle_event(self, pressed, current_time):
        """Handle a button press or release event."""
        self.is_pressed = pressed

        if pressed:
            self._on_press(current_time)
        else:
            self._on_release(current_time)

    def _on_press(self, current_time):
        """Handle button press event."""
        if self.state == GestureState.IDLE:
            # First press
            self.state = GestureState.PRESSED
            self.press_time = current_time
            self.long_press_fired = False

        elif self.state == GestureState.WAIT_DOUBLE:
            # Second press - potential double press
            self.state = GestureState.DOUBLE_PRESSED
            self.press_time = current_time

    def _on_release(self, current_time):
        """Handle button release event."""
        if self.state == GestureState.PRESSED:
            if self.long_press_fired:
                # Long press already fired, go back to idle
                self.state = GestureState.IDLE
            else:
                # Short press - wait for potential double press
                self.state = GestureState.WAIT_DOUBLE
                self.release_time = current_time

        elif self.state == GestureState.DOUBLE_PRESSED:
            # Released after second press - fire double press
            self._fire_gesture("double_press")
            self.state = GestureState.IDLE

    def update(self, current_time):
        """
        Update the gesture state machine. Call this every loop iteration
        to handle time-based state transitions (long press, double-press timeout).
        """
        if self.state == GestureState.PRESSED:
            # Check for long press while button is held
            if self.is_pressed and not self.long_press_fired:
                hold_duration = current_time - self.press_time
                if hold_duration >= self.long_press_threshold:
                    self._fire_gesture("long_press")
                    self.long_press_fired = True

        elif self.state == GestureState.WAIT_DOUBLE:
            # Check for double-press timeout
            wait_duration = current_time - self.release_time
            if wait_duration >= self.double_press_window:
                # Timeout - fire single press
                self._fire_gesture("press")
                self.state = GestureState.IDLE

    def _fire_gesture(self, gesture_type):
        """Execute the action for a gesture type."""
        if gesture_type not in self.config:
            return

        keys = self.config[gesture_type]
        if not keys:
            return

        # Check if it's a sequence (list of lists) or a single combo (flat list)
        if keys and isinstance(keys[0], list):
            # Sequence of key combos
            for combo in keys:
                self._send_keys(combo)
                time.sleep(0.05)  # Small delay between sequence items
        else:
            # Single key combo
            self._send_keys(keys)

    def _send_keys(self, keycodes):
        """Send a key combination."""
        if keycodes:
            self.keyboard.send(*keycodes)

