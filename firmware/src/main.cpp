#include <Arduino.h>
#include "config.h"
#include "display/display_manager.h"
#include "seesaw/seesaw_manager.h"
#include "comms/serial_comms.h"
#include "sdcard/sdcard_manager.h"

// With ARDUINO_USB_MODE=0 (TinyUSB OTG), Serial = USB CDC.
// The device enumerates as a composite CDC + MSC device.
// Debug prints are suppressed once the bridge connects (to avoid
// mixing text with binary protocol frames on the same serial port).

static DisplayManager display;
static SeesawManager seesaw;
static SerialComms comms;
static SDCardManager sdcard;

// Debug print helper — suppressed when bridge is connected
#define DBG(fmt, ...) do { if (!comms.bridgeConnected()) Serial.printf(fmt "\n", ##__VA_ARGS__); } while(0)

// --- Callbacks ---

static void onButtonChange(uint8_t buttonId, bool pressed) {
    DBG("[btn] id=%d pressed=%d", buttonId, pressed);
    comms.sendButtonEvent(buttonId, pressed);

    // Visual feedback via NeoPixels
    if (pressed) {
        seesaw.setPixelColor(buttonId, 0x004400);  // Green when pressed
    } else {
        seesaw.setPixelColor(buttonId, 0x000000);  // Off when released
    }
    seesaw.showPixels();
}

static void onDisplayText(const char* text, uint16_t len) {
    char buf[512];
    uint16_t copyLen = len < sizeof(buf) - 1 ? len : sizeof(buf) - 1;
    memcpy(buf, text, copyLen);
    buf[copyLen] = '\0';

    display.setNotificationText(buf);
    display.update();
}

static void onStatusText(const char* text, uint16_t len) {
    char buf[128];
    uint16_t copyLen = len < sizeof(buf) - 1 ? len : sizeof(buf) - 1;
    memcpy(buf, text, copyLen);
    buf[copyLen] = '\0';

    display.setStatusText(buf);
    display.update();
}

static void onSetLeds(const uint8_t* data, uint16_t len) {
    for (uint16_t i = 0; i + 3 < len; i += 4) {
        uint8_t pixel = data[i];
        uint32_t color = ((uint32_t)data[i+1] << 16) |
                         ((uint32_t)data[i+2] << 8) |
                         data[i+3];
        seesaw.setPixelColor(pixel, color);
    }
    seesaw.showPixels();
}

static void onBridgeDisconnected() {
    display.setStatusText("DISCONNECTED", 0xff0000);
    display.update();
}

static void onClearDisplay() {
    display.setStatusText("Ready");
    display.setNotificationText("");
    display.setButtonLabels("1", "2", "3", "4");
    display.update();
}

static void onSetButtonLabels(const char* labels[4]) {
    display.setButtonLabels(labels[0], labels[1], labels[2], labels[3]);
    display.update();
}

void setup() {
    Serial.begin(SERIAL_BAUD);
    delay(2000);  // Let TinyUSB CDC enumerate

    // Setup debug prints always go through (bridge can't be connected yet)
    Serial.println("\n=== CamelPad Firmware Starting ===");

    Serial.println("[1/4] Initializing display...");
    display.begin();
    display.setStatusText("Booting...");
    display.update();
    Serial.println("[1/4] Display OK");

    // SD card init must happen after display.begin() because GPIO1/GPIO2 are
    // shared between the ST7701 3-wire SPI init (one-shot) and SDMMC CLK/CMD.
    Serial.println("[2/4] Initializing SD card + USB MSC...");
    bool sdOk = sdcard.begin();
    if (sdOk) {
        display.setStatusText("SD card mounted");
        display.update();
    } else {
        display.setStatusText("No SD card");
        display.update();
    }
    sdcard.beginUSB();  // Register MSC and start USB (CDC + MSC composite)
    Serial.println("[2/4] SD card + USB MSC OK");

    Serial.println("[3/4] Initializing Seesaw...");
    if (!seesaw.begin()) {
        Serial.println("[3/4] Seesaw init FAILED!");
        display.setStatusText("Seesaw init FAILED");
        display.update();
    } else {
        Serial.println("[3/4] Seesaw OK");
        for (int i = 0; i < SEESAW_NEOPIXEL_COUNT; i++) {
            seesaw.setPixelColor(i, 0x001100);
        }
        seesaw.showPixels();
        delay(500);
    }

    seesaw.onButtonChange(onButtonChange);

    Serial.println("[4/4] Initializing comms...");
    comms.begin();
    comms.onDisplayText(onDisplayText);
    comms.onStatusText(onStatusText);
    comms.onSetLeds(onSetLeds);
    comms.onClearDisplay(onClearDisplay);
    comms.onSetButtonLabels(onSetButtonLabels);
    comms.onBridgeDisconnected(onBridgeDisconnected);
    Serial.println("[4/4] Comms OK");

    seesaw.clearPixels();
    seesaw.showPixels();

    display.setStatusText("Ready - Waiting for connection...");
    display.update();
    Serial.println("=== Setup Complete ===");
}

static uint32_t lastHeartbeat = 0;

void loop() {
    comms.poll();
    seesaw.poll();

    // Periodic heartbeat — suppressed when bridge is connected
    if (millis() - lastHeartbeat > 5000) {
        lastHeartbeat = millis();
        DBG("[heartbeat] uptime=%lus", millis() / 1000);
    }

    delay(10);
}
