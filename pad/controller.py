"""
Controller for managing button inputs and gesture detection.
"""
import time
import keypad
import usb_hid
from adafruit_hid.keyboard import Keyboard

from gesture import ButtonGestureDetector


class PadController:
    """Manages button inputs, gesture detection, and keyboard output."""

    def __init__(self, button_pins, buttons_config, timing):
        """
        Initialize the pad controller.

        Args:
            button_pins: List of board pins for buttons
            buttons_config: Dict mapping button index to gesture actions
            timing: Dict with timing configuration for gestures
        """
        self.button_pins = button_pins
        self.buttons_config = buttons_config
        self.timing = timing

        # Set up the keypad with configured button pins
        self.pad = keypad.Keys(button_pins, value_when_pressed=False)

        # Set up the keyboard
        self.keyboard = Keyboard(usb_hid.devices)

        # Create gesture detectors for configured buttons
        self.detectors = {}
        for button_index, button_config in buttons_config.items():
            if button_index < len(button_pins):
                self.detectors[button_index] = ButtonGestureDetector(
                    button_index, button_config, self.keyboard, timing
                )

    def update(self):
        """
        Process button events and update gesture detectors.
        Call this every loop iteration.
        """
        current_time = time.monotonic()
        
        # Process button events
        while (event := self.pad.events.get()) is not None:
            button_index = event.key_number
            if button_index in self.detectors:
                self.detectors[button_index].handle_event(event.pressed, current_time)

        # Update all detectors for time-based state transitions
        for detector in self.detectors.values():
            detector.update(current_time)

    @property
    def button_count(self):
        """Number of configured button pins."""
        return len(self.button_pins)

    @property
    def configured_buttons(self):
        """List of button indices with configured actions."""
        return list(self.buttons_config.keys())

