"""
Camel Pad boot.py - Custom USB HID Configuration

This file runs before code.py and configures the USB HID devices.
It creates a custom HID device with:
- Extended keyboard IN report (10 keycodes)
- Host message OUT report (64 bytes) for receiving commands from host
"""
import usb_hid

# Custom HID Report Descriptor for Camel Pad
#
# Report ID 1: Extended Keyboard (IN report - device to host)
#   - 1 byte modifier keys (Ctrl, Shift, Alt, GUI)
#   - 1 byte reserved
#   - 10 bytes for keycodes (supports 10 simultaneous keys)
#
# Report ID 2: Host Message (OUT report - host to device)
#   - 64 bytes for custom messages from host
#
CAMEL_PAD_REPORT_DESCRIPTOR = bytes([
    # ============================================
    # Extended Keyboard (Report ID 1)
    # ============================================
    0x05, 0x01,        # Usage Page (Generic Desktop)
    0x09, 0x06,        # Usage (Keyboard)
    0xA1, 0x01,        # Collection (Application)
    0x85, 0x01,        #   Report ID (1)

    # Modifier byte (8 bits for modifier keys)
    0x05, 0x07,        #   Usage Page (Key Codes)
    0x19, 0xE0,        #   Usage Minimum (224) - Left Control
    0x29, 0xE7,        #   Usage Maximum (231) - Right GUI
    0x15, 0x00,        #   Logical Minimum (0)
    0x25, 0x01,        #   Logical Maximum (1)
    0x75, 0x01,        #   Report Size (1 bit)
    0x95, 0x08,        #   Report Count (8 bits)
    0x81, 0x02,        #   Input (Data, Variable, Absolute)

    # Reserved byte
    0x75, 0x08,        #   Report Size (8 bits)
    0x95, 0x01,        #   Report Count (1)
    0x81, 0x01,        #   Input (Constant) - Reserved byte

    # LED output report (standard keyboard LEDs)
    0x05, 0x08,        #   Usage Page (LEDs)
    0x19, 0x01,        #   Usage Minimum (1)
    0x29, 0x05,        #   Usage Maximum (5)
    0x75, 0x01,        #   Report Size (1 bit)
    0x95, 0x05,        #   Report Count (5)
    0x91, 0x02,        #   Output (Data, Variable, Absolute) - LED states
    0x75, 0x03,        #   Report Size (3 bits)
    0x95, 0x01,        #   Report Count (1)
    0x91, 0x01,        #   Output (Constant) - Padding

    # Keycodes (10 bytes for up to 10 simultaneous keys)
    0x05, 0x07,        #   Usage Page (Key Codes)
    0x19, 0x00,        #   Usage Minimum (0)
    0x29, 0xFF,        #   Usage Maximum (255)
    0x15, 0x00,        #   Logical Minimum (0)
    0x26, 0xFF, 0x00,  #   Logical Maximum (255)
    0x75, 0x08,        #   Report Size (8 bits)
    0x95, 0x0A,        #   Report Count (10) - 10 keycodes
    0x81, 0x00,        #   Input (Data, Array)

    0xC0,              # End Collection

    # ============================================
    # Host Message (Report ID 2) - OUT Report
    # For receiving data from host to device
    # ============================================
    0x06, 0x00, 0xFF,  # Usage Page (Vendor Defined 0xFF00)
    0x09, 0x01,        # Usage (Vendor Usage 1)
    0xA1, 0x01,        # Collection (Application)
    0x85, 0x02,        #   Report ID (2)

    # OUT report data (64 bytes for efficient USB transfers)
    0x09, 0x02,        #   Usage (Vendor Usage 2)
    0x15, 0x00,        #   Logical Minimum (0)
    0x26, 0xFF, 0x00,  #   Logical Maximum (255)
    0x75, 0x08,        #   Report Size (8 bits)
    0x95, 0x40,        #   Report Count (64) - 64 bytes per packet
    0x91, 0x02,        #   Output (Data, Variable, Absolute)

    0xC0,              # End Collection
])

# Create the custom Camel Pad HID device
camel_pad_device = usb_hid.Device(
    report_descriptor=CAMEL_PAD_REPORT_DESCRIPTOR,
    usage_page=0x01,           # Generic Desktop (for keyboard)
    usage=0x06,                # Keyboard
    report_ids=(1, 2),         # Report IDs: 1=keyboard, 2=out
    in_report_lengths=(12, 0),       # Report ID 1: 12 bytes IN (1 modifier + 1 reserved + 10 keys)
    out_report_lengths=(1, 64),      # Report ID 1: 1 byte OUT (LEDs), Report ID 2: 64 bytes
)

# Enable the custom device
# We keep the default devices and add our custom device
# This allows both standard keyboard functionality and custom extended features
usb_hid.enable(
    (
        usb_hid.Device.KEYBOARD,    # Standard keyboard for compatibility
        usb_hid.Device.CONSUMER_CONTROL,  # Media controls
        camel_pad_device,           # Our custom extended device
    )
)

print("Camel Pad custom HID enabled")
