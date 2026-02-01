import usb_hid
import traceback

# HID message types (must match host protocol)
MSG_DISPLAY_TEXT = 0x01
MSG_BUTTON = 0x02
MSG_ACKNOWLEDGE = 0x03

# Report size
REPORT_SIZE = 64


class HIDInterface:
    """Manages communications with host"""

    def __init__(self):
        # Find the custom HID device
        self.hid_device = None
        for device in usb_hid.devices:
            # Look for vendor-specific usage page (0xFF00)
            if device.usage_page == 0xFF00 and device.usage == 0x01:
                self.hid_device = device
        
        if not self.hid_device:
            print("ERROR: Vendor HID device not found. Check boot.py")
            return

        print(f"HID device found: usage_page=0x{self.hid_device.usage_page:04x}")
        
        # Pre-allocate buffers
        self.out_report = bytearray(REPORT_SIZE)
        self.in_buffer = bytearray(REPORT_SIZE)
    
    
    def get_message_text(self):
        print("Getting message")

        data = self.hid_device.get_last_received_report()
        if data and len(data) > 0:
            print(f"Received message!")
            
            msg_type = data[0]

            if msg_type == MSG_DISPLAY_TEXT:
                text = None
        
                # Extract text (bytes 1-63, null-terminated)
                text_bytes = bytes(data[1:])
                # Find null terminator or use full length
                try:
                    null_pos = text_bytes.index(b'\x00')
                    text = text_bytes[:null_pos].decode('utf-8')
                    
                except ValueError as e:
                    traceback.print_exception(e)
                    text = text_bytes.decode('utf-8', errors='replace')
                
                return text
            
            return f"unknown message {msg_type} received"
    
    def send_ack(self) -> string:
        self.out_report[0] = MSG_ACKNOWLEDGE
        self._zero_report(self.out_report, 1)
        
        try:
            self.hid_device.send_report(self.out_report)
            
            return "Acknowledgement sent"
            
        except Exception as e:
            return f"HID send error: {e}"
    
    
    def send_button_press(self, button_id: int, pressed: bool) -> str:
        # Send button event to host
        self.out_report[0] = MSG_BUTTON
        self.out_report[1] = button_id
        self.out_report[2] = pressed
        
        self._zero_report(self.out_report, 3)

        try:
            self.hid_device.send_report(self.out_report)
            return f"Button {button_id} {'pressed' if pressed else 'released'}"
            
        except Exception as e:
            return f"HID send error: {e}"
    
            
    def _zero_report(self, report, start_index):
        # Clear rest of report
        for i in range(start_index, REPORT_SIZE):
            self.out_report[i] = 0
        