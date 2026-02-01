import usb_hid
import supervisor

# Disable the terminal output on the built-in display
supervisor.runtime.display.root_group = None

# Vendor-specific HID report descriptor
# UsagePage: 0xFF00 (Vendor Defined)
# Usage: 0x01
# Report size: 64 bytes (for both IN and OUT)
CAMELPAD_REPORT_DESCRIPTOR = bytes((
    0x06, 0x00, 0xFF,  # Usage Page (Vendor Defined 0xFF00)
    0x09, 0x01,        # Usage (0x01)
    0xA1, 0x01,        # Collection (Application)
    0x15, 0x00,        #   Logical Minimum (0)
    0x26, 0xFF, 0x00,  #   Logical Maximum (255)
    0x75, 0x08,        #   Report Size (8 bits)
    0x95, 0x40,        #   Report Count (64)
    0x09, 0x01,        #   Usage (0x01)
    0x81, 0x02,        #   Input (Data, Variable, Absolute) - device to host
    0x09, 0x01,        #   Usage (0x01)
    0x91, 0x02,        #   Output (Data, Variable, Absolute) - host to device
    0xC0,              # End Collection
))

# Create the custom HID device
camelpad_device = usb_hid.Device(
    report_descriptor=CAMELPAD_REPORT_DESCRIPTOR,
    usage_page=0xFF00,
    usage=0x01,
    report_ids=(0,),
    in_report_lengths=(64,),
    out_report_lengths=(64,),
)

# Enable only our custom device (disable default keyboard/mouse/consumer)
usb_hid.enable((camelpad_device,))

print("boot.py: Enabled vendor-specific HID device (usagePage=0xFF00)")
