"""
Custom HID device support for Camel Pad.

Provides classes for:
- CamelPadKeyboard: Extended keyboard with support for 10 simultaneous keycodes
- CamelPadHostReceiver: Receives messages from the host via OUT reports
"""
import usb_hid
from adafruit_hid import find_device


# Report IDs (must match boot.py)
REPORT_ID_KEYBOARD = 1
REPORT_ID_OUT = 2

# Report sizes
KEYBOARD_REPORT_SIZE = 12  # 1 modifier + 1 reserved + 10 keycodes
OUT_REPORT_SIZE = 64

# Modifier key bits (same as standard HID keyboard)
MODIFIER_LEFT_CTRL = 0x01
MODIFIER_LEFT_SHIFT = 0x02
MODIFIER_LEFT_ALT = 0x04
MODIFIER_LEFT_GUI = 0x08
MODIFIER_RIGHT_CTRL = 0x10
MODIFIER_RIGHT_SHIFT = 0x20
MODIFIER_RIGHT_ALT = 0x40
MODIFIER_RIGHT_GUI = 0x80

# Keycode to modifier mapping
_KEYCODE_TO_MODIFIER = {
    0xE0: MODIFIER_LEFT_CTRL,
    0xE1: MODIFIER_LEFT_SHIFT,
    0xE2: MODIFIER_LEFT_ALT,
    0xE3: MODIFIER_LEFT_GUI,
    0xE4: MODIFIER_RIGHT_CTRL,
    0xE5: MODIFIER_RIGHT_SHIFT,
    0xE6: MODIFIER_RIGHT_ALT,
    0xE7: MODIFIER_RIGHT_GUI,
}


class CamelPadKeyboard:
    """
    Extended keyboard supporting up to 10 simultaneous keycodes.

    This is similar to adafruit_hid.keyboard.Keyboard but supports more keys.
    Compatible with standard Keycode values from adafruit_hid.keycode.

    Usage:
        keyboard = CamelPadKeyboard(usb_hid.devices)
        keyboard.send(Keycode.A, Keycode.B, Keycode.C)  # Up to 10 keys

        # Or use press/release for more control:
        keyboard.press(Keycode.SHIFT, Keycode.A)
        keyboard.release(Keycode.A)
        keyboard.release_all()
    """

    def __init__(self, devices, timeout=None):
        """
        Initialize the extended keyboard.

        Args:
            devices: usb_hid.devices or a list of HID devices
            timeout: Optional timeout for device detection
        """
        self._device = None

        # Try to find our custom Camel Pad device first
        for device in devices:
            if (hasattr(device, 'usage_page') and
                device.usage_page == 0x01 and
                device.usage == 0x06):
                # Check if this might be our custom device
                # We'll try the first keyboard-like device we find
                self._device = device
                break

        if self._device is None:
            # Fallback: try to find any keyboard device
            self._device = find_device(devices, usage_page=0x01, usage=0x06)

        if self._device is None:
            raise RuntimeError("Could not find Camel Pad keyboard device")

        # Initialize the report buffer
        # Report format: [modifier, reserved, key1, key2, ..., key10]
        self._report = bytearray(KEYBOARD_REPORT_SIZE)
        self._report_modifier = memoryview(self._report)[0:1]
        self._report_keys = memoryview(self._report)[2:12]  # 10 keycodes

        # Track pressed keys
        self._pressed_keys = set()

    def send(self, *keycodes):
        """
        Send a key combination and release.

        Args:
            keycodes: Up to 10 keycodes to send simultaneously
        """
        self.press(*keycodes)
        self.release_all()

    def press(self, *keycodes):
        """
        Press one or more keys. They will remain pressed until released.

        Args:
            keycodes: Keycodes to press (up to 10 non-modifier keys)
        """
        for keycode in keycodes:
            self._add_keycode(keycode)
        self._send_report()

    def release(self, *keycodes):
        """
        Release one or more keys.

        Args:
            keycodes: Keycodes to release
        """
        for keycode in keycodes:
            self._remove_keycode(keycode)
        self._send_report()

    def release_all(self):
        """Release all currently pressed keys."""
        self._pressed_keys.clear()
        for i in range(len(self._report)):
            self._report[i] = 0
        self._send_report()

    def _add_keycode(self, keycode):
        """Add a keycode to the pressed set."""
        if keycode in _KEYCODE_TO_MODIFIER:
            # It's a modifier key
            self._report_modifier[0] |= _KEYCODE_TO_MODIFIER[keycode]
        else:
            # Regular key
            self._pressed_keys.add(keycode)
            self._update_key_report()

    def _remove_keycode(self, keycode):
        """Remove a keycode from the pressed set."""
        if keycode in _KEYCODE_TO_MODIFIER:
            # It's a modifier key
            self._report_modifier[0] &= ~_KEYCODE_TO_MODIFIER[keycode]
        else:
            # Regular key
            self._pressed_keys.discard(keycode)
            self._update_key_report()

    def _update_key_report(self):
        """Update the key portion of the report from pressed_keys."""
        # Clear keycodes
        for i in range(10):
            self._report_keys[i] = 0

        # Fill in pressed keys (up to 10)
        for i, keycode in enumerate(list(self._pressed_keys)[:10]):
            self._report_keys[i] = keycode

    def _send_report(self):
        """Send the current report to the host."""
        self._device.send_report(self._report, REPORT_ID_KEYBOARD)


class CamelPadHostReceiver:
    """
    Receives messages from the host via OUT reports.

    Usage:
        receiver = CamelPadHostReceiver(usb_hid.devices)

        # Check for OUT report (64 bytes)
        data = receiver.get_out_report()
        if data:
            print(f"Received data: {data}")
    """

    def __init__(self, devices):
        """
        Initialize the host receiver.

        Args:
            devices: usb_hid.devices or a list of HID devices
        """
        self._device = None

        # Find our custom device
        for device in devices:
            if (hasattr(device, 'usage_page') and
                device.usage_page == 0x01 and
                device.usage == 0x06):
                self._device = device
                break

        if self._device is None:
            raise RuntimeError("Could not find Camel Pad HID device")

        # Buffer for receiving data
        self._out_buffer = bytearray(OUT_REPORT_SIZE)

    def get_out_report(self):
        """
        Get the last OUT report data from the host.

        Returns:
            bytes: The OUT report data (up to 64 bytes) or None if no new data
        """
        try:
            # Try to read the OUT report
            count = self._device.get_last_received_report(self._out_buffer, REPORT_ID_OUT)
            if count > 0:
                return bytes(self._out_buffer[:count])
        except (AttributeError, RuntimeError):
            # Device doesn't support get_last_received_report or no data available
            pass
        return None

    @property
    def out_report_size(self):
        """Size of OUT report in bytes."""
        return OUT_REPORT_SIZE


class CamelPadDevice:
    """
    Unified interface for the Camel Pad custom HID device.

    Combines keyboard and host receiver functionality.

    Usage:
        pad = CamelPadDevice(usb_hid.devices)

        # Send keys
        pad.keyboard.send(Keycode.A)

        # Check for host messages
        msg = pad.get_host_message()
        if msg:
            print(f"Host says: {msg}")
    """

    def __init__(self, devices):
        """
        Initialize the Camel Pad device.

        Args:
            devices: usb_hid.devices or a list of HID devices
        """
        self.keyboard = CamelPadKeyboard(devices)
        self.host_receiver = CamelPadHostReceiver(devices)

    def get_host_message(self):
        """
        Get any pending message from the host.

        Returns:
            bytes: Message data or None if no message
        """
        return self.host_receiver.get_out_report()

    def send_keys(self, *keycodes):
        """
        Send a key combination.

        Args:
            keycodes: Up to 10 keycodes to send
        """
        self.keyboard.send(*keycodes)
