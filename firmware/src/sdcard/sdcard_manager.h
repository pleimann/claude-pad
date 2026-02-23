#pragma once

#include <stdbool.h>
#include <stdint.h>

// SDCardManager initializes the SD card over SDMMC and exposes it as a USB
// Mass Storage Class (MSC) device via TinyUSB.
//
// Call begin() after display.begin() — the SD card shares GPIO1/GPIO2 with
// the ST7701 3-wire SPI init interface, which is one-shot and done by the
// time begin() is called.
//
// Call beginUSB() after the SD card is mounted to register the USBMSC
// callbacks and start the USB stack.
class SDCardManager {
public:
    // Mount the SD card. Returns true if a card was found and mounted.
    bool begin();

    // Register USBMSC read/write callbacks and start USB.
    // Call this after begin(). Safe to call even if begin() returned false
    // (host will see "no media present").
    void beginUSB();

    bool isMounted() const { return _mounted; }
    uint64_t totalBytes() const;
    uint64_t usedBytes() const;

private:
    bool _mounted = false;
};
