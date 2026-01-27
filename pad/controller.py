"""
Controller for managing button inputs and gesture detection.
"""
import time
import keypad
import usb_hid
from adafruit_hid.keyboard import Keyboard

from gesture import ButtonGestureDetector

# Try to import custom HID support
try:
    from custom_hid import CamelPadKeyboard, CamelPadHostReceiver
    CUSTOM_HID_AVAILABLE = True
except ImportError:
    CUSTOM_HID_AVAILABLE = False


class PadController:
    """Manages button inputs, gesture detection, and keyboard output."""

    def __init__(self, button_pins, buttons_config, timing, use_custom_hid=True):
        """
        Initialize the pad controller.

        Args:
            button_pins: List of board pins for buttons
            buttons_config: Dict mapping button index to gesture actions
            timing: Dict with timing configuration for gestures
            use_custom_hid: If True, use custom HID with 10-key support (default: True)
        """
        self.button_pins = button_pins
        self.buttons_config = buttons_config
        self.timing = timing
        self.use_custom_hid = use_custom_hid and CUSTOM_HID_AVAILABLE

        # Set up the keypad with configured button pins
        self.pad = keypad.Keys(button_pins, value_when_pressed=False)

        # Set up the keyboard (try custom HID first, fall back to standard)
        self.host_receiver = None
        if self.use_custom_hid:
            try:
                self.keyboard = CamelPadKeyboard(usb_hid.devices)
                self.host_receiver = CamelPadHostReceiver(usb_hid.devices)
                print("Using custom HID with 10-key support")
            except RuntimeError:
                # Fall back to standard keyboard
                self.keyboard = Keyboard(usb_hid.devices)
                self.use_custom_hid = False
                print("Falling back to standard HID keyboard")
        else:
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

    def get_host_message(self):
        """
        Check for and return any pending message from the host.

        Returns:
            bytes: The message data (up to 64 bytes) or None if no message is available.
                   Only works when using custom HID.
        """
        if self.host_receiver is not None:
            return self.host_receiver.get_out_report()
        return None

    @property
    def button_count(self):
        """Number of configured button pins."""
        return len(self.button_pins)

    @property
    def configured_buttons(self):
        """List of button indices with configured actions."""
        return list(self.buttons_config.keys())

    @property
    def has_custom_hid(self):
        """True if using custom HID with extended features."""
        return self.use_custom_hid

    @property
    def max_simultaneous_keys(self):
        """Maximum number of simultaneous keys supported."""
        return 10 if self.use_custom_hid else 6
