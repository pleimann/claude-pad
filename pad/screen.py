"""
Controller for managing button inputs and gesture detection.
"""
import time
import board
import displayio
import terminalio
from adafruit_display_text import label


class ScreenController:
    """Manages button inputs, gesture detection, and keyboard output."""

    def __init__(self):
        """
        Initialize the screen layout
        
        """

        # Set up display
        display = board.DISPLAY
        display.auto_refresh = True

        # Create display group
        self.group = displayio.Group()

        # Create text label
        self.status_label = label.Label(
            terminalio.FONT,
            text="Ready",
            color=0x000000,
            background_color=0x00FF00,
            scale=2,
        )
        # Set label position on the display
        self.status_label.anchor_point = (0, 0)
        self.status_label.anchored_position = (0, 0)

        self.group.append(self.status_label)
        
        # Create text label
        self.message_label = label.Label(
            terminalio.FONT,
            text="",
            color=0xFFFFFF,
            scale=4,
        )
        self.message_label.anchor_point = (0, 0)
        self.message_label.anchored_position = (0, 20)

        self.group.append(self.message_label)

        display.root_group = self.group

        print(f"Display: {display.width}x{display.height}")
        

    def set_status(self, status_text):
        self.status_label.text = status_text


    def set_message(self, message):
        self.message_label.text = message

