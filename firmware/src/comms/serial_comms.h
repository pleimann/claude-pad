#pragma once

#include <cstdint>
#include "protocol.h"

class SerialComms {
public:
    using TextCallback   = void (*)(const char* text, uint16_t len);
    using LedsCallback   = void (*)(const uint8_t* data, uint16_t len);
    using LabelsCallback = void (*)(const char* labels[4]);
    using VoidCallback   = void (*)();

    void begin();
    void poll();

    void sendButtonEvent(uint8_t buttonId, bool pressed);
    void sendHeartbeat(uint8_t status);

    bool bridgeConnected() const { return _bridgeConnected; }

    void onDisplayText(TextCallback cb)     { _onDisplayText = cb; }
    void onStatusText(TextCallback cb)      { _onStatusText = cb; }
    void onSetLeds(LedsCallback cb)         { _onSetLeds = cb; }
    void onClearDisplay(VoidCallback cb)    { _onClearDisplay = cb; }
    void onSetButtonLabels(LabelsCallback cb) { _onSetLabels = cb; }

private:
    void processMessage(uint8_t msgType, const uint8_t* payload, uint16_t len);
    void sendFrame(uint8_t msgType, const uint8_t* payload, uint16_t len);

    enum ParseState {
        WAIT_START,
        READ_LEN_HI,
        READ_LEN_LO,
        READ_BODY,
        READ_CHECKSUM
    };

    ParseState _state = WAIT_START;
    uint8_t    _buffer[MAX_MSG_LEN];
    uint16_t   _bodyLen = 0;
    uint16_t   _bodyIdx = 0;
    unsigned long _lastByteTime = 0;  // Timeout tracking
    static const unsigned long FRAME_TIMEOUT_MS = 500;  // Reset if no complete frame in 500ms

    bool           _bridgeConnected = false;
    TextCallback   _onDisplayText = nullptr;
    TextCallback   _onStatusText  = nullptr;
    LedsCallback   _onSetLeds     = nullptr;
    VoidCallback   _onClearDisplay = nullptr;
    LabelsCallback _onSetLabels   = nullptr;
};
