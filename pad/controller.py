"""
Controller for managing button inputs and gesture detection.
"""
import time
import board
import keypad
import usb_hid
from adafruit_hid.keyboard import Keyboard


# Button pins (adjust for your hardware)
BUTTON_PINS = [board.BOOT]

class KeysController:
    """Manages button inputs, gesture detection, and keyboard output."""

    def __init__(self):
        """
        Initialize the pad controller.

        Args:
            button_pins: List of board pins for buttons
            buttons_config: Dict mapping button index to gesture actions
            timing: Dict with timing configuration for gestures
        """
        # Set up button keypad
        self.buttons = keypad.Keys(BUTTON_PINS, value_when_pressed=False)
        print(f"Buttons: {len(BUTTON_PINS)} configured")
        

    def get_key_events(self):
        """
        Process button events
        Call this every loop iteration.
        """
        print("Getting key events")
        
        # Process button events
        key_events = []
        while (event := self.buttons.events.get()) is not None:
            key_event = {
                "button_index": event.key_number,
                "pressed": event.pressed,
                "timestamp": event.timestamp
            }
            
            key_events.append(key_event)
                        
        print(f"Found {len(key_events)} key events")
        
        return key_events


    @property
    def button_count(self):
        """Number of configured button pins."""
        return len(self.button_pins)

    @property
    def configured_buttons(self):
        """List of button indices with configured actions."""
        return list(self.buttons_config.keys())

