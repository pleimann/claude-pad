#include "serial_comms.h"
#include <Arduino.h>

void SerialComms::begin() {
    // Serial is already initialized by Arduino framework when
    // ARDUINO_USB_CDC_ON_BOOT=1
    _state = WAIT_START;
}

void SerialComms::poll() {
    // Reset state machine if timeout waiting for frame completion (prevents state machine from getting stuck)
    if (_state != WAIT_START && (millis() - _lastByteTime) > FRAME_TIMEOUT_MS) {
        _state = WAIT_START;
    }

    while (Serial.available()) {
        uint8_t byte = Serial.read();
        _lastByteTime = millis();  // Track when we received data

        switch (_state) {
        case WAIT_START:
            if (byte == FRAME_START_BYTE) {
                _state = READ_LEN_HI;
            }
            break;

        case READ_LEN_HI:
            _bodyLen = (uint16_t)byte << 8;
            _state = READ_LEN_LO;
            break;

        case READ_LEN_LO:
            _bodyLen |= byte;
            if (_bodyLen == 0 || _bodyLen > MAX_MSG_LEN) {
                _state = WAIT_START;  // Invalid length
            } else {
                _bodyIdx = 0;
                _state = READ_BODY;
            }
            break;

        case READ_BODY:
            _buffer[_bodyIdx++] = byte;
            if (_bodyIdx >= _bodyLen) {
                _state = READ_CHECKSUM;
            }
            break;

        case READ_CHECKSUM: {
            uint8_t expected = protocol::checksum(_buffer, _bodyLen);
            if (byte == expected) {
                processMessage(_buffer[0], _buffer + 1, _bodyLen - 1);
            }
            _state = WAIT_START;
            break;
        }
        }
    }
}

void SerialComms::processMessage(uint8_t msgType, const uint8_t* payload, uint16_t len) {
    _bridgeConnected = true;
    switch (msgType) {
    case MSG_DISPLAY_TEXT:
        if (_onDisplayText && len > 0) {
            _onDisplayText((const char*)payload, len);
        }
        break;

    case MSG_STATUS:
        if (_onStatusText && len > 0) {
            _onStatusText((const char*)payload, len);
        }
        break;

    case MSG_SET_LEDS:
        if (_onSetLeds && len > 0) {
            _onSetLeds(payload, len);
        }
        break;

    case MSG_CLEAR:
        if (_onClearDisplay) {
            _onClearDisplay();
        }
        break;

    case MSG_SET_LABELS: {
        if (_onSetLabels && len > 0) {
            const char* labels[4] = {"", "", "", ""};
            static char labelBufs[4][32];
            int labelIdx = 0;
            uint16_t pos = 0;

            while (pos < len && labelIdx < 4) {
                uint8_t labelLen = payload[pos++];
                if (pos + labelLen > len) break;
                uint8_t copyLen = labelLen < 31 ? labelLen : 31;
                memcpy(labelBufs[labelIdx], payload + pos, copyLen);
                labelBufs[labelIdx][copyLen] = '\0';
                labels[labelIdx] = labelBufs[labelIdx];
                pos += labelLen;
                labelIdx++;
            }
            _onSetLabels(labels);
        }
        break;
    }
    }
}

void SerialComms::sendFrame(uint8_t msgType, const uint8_t* payload, uint16_t len) {
    uint8_t frame[MAX_MSG_LEN + 5];
    uint16_t frameLen = protocol::buildFrame(frame, msgType, payload, len);
    Serial.write(frame, frameLen);
}

void SerialComms::sendButtonEvent(uint8_t buttonId, bool pressed) {
    uint8_t payload[2] = {buttonId, (uint8_t)(pressed ? 1 : 0)};
    sendFrame(MSG_BUTTON, payload, 2);
}

void SerialComms::sendHeartbeat(uint8_t status) {
    sendFrame(MSG_HEARTBEAT, &status, 1);
}
