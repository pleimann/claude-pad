"""
CamelPad main application.
Handles HID communication with host and button/display management.
"""
import time
import board
import traceback

from controller import KeysController
from hid_interface import HIDInterface
from screen import ScreenController

keys_controller = KeysController()
hid_device = HIDInterface()
screen = ScreenController()

print("CamelPad ready")

loop_count = 0
while True:    
    # Check for incoming HID data from host
    try:
        loop_count += 1
        
        print("\n")

        # get_last_received_report returns None if no data
        text = hid_device.get_message_text()
        
        if text and len(text) > 0:
            print(f"Display: {text}")
            screen.set_message(text)

            # ack not implemented yet
            #status = hid_device.send_ack()
            screen.set_status("message recieved")

        key_events = keys_controller.get_key_events()
        for event in key_events:
            hid_device.send_button_press(event['button_index'], event['pressed'])
        
    except Exception as e:
        traceback.print_exception(e)
        pass
    
    # take this out to avoid timing problems with double and long click
    #time.sleep(1)
