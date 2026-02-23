#include "sdcard_manager.h"
#include "config.h"

#include <Arduino.h>
#include <USB.h>
#include <USBMSC.h>
#include <SD_MMC.h>

static USBMSC msc;

// --- USB MSC callbacks ---

static int32_t onRead(uint32_t lba, uint32_t offset, void* buffer, uint32_t bufsize) {
    uint32_t secSize = SD_MMC.sectorSize();
    if (!secSize) return -1;
    for (uint32_t x = 0; x < bufsize / secSize; x++) {
        if (!SD_MMC.readRAW((uint8_t*)buffer + (x * secSize), lba + x)) return -1;
    }
    return (int32_t)bufsize;
}

static int32_t onWrite(uint32_t lba, uint32_t offset, uint8_t* buffer, uint32_t bufsize) {
    uint32_t secSize = SD_MMC.sectorSize();
    if (!secSize) return -1;
    for (uint32_t x = 0; x < bufsize / secSize; x++) {
        uint8_t blk[secSize];
        memcpy(blk, buffer + secSize * x, secSize);
        if (!SD_MMC.writeRAW(blk, lba + x)) return -1;
    }
    return (int32_t)bufsize;
}

static bool onStartStop(uint8_t power_condition, bool start, bool load_eject) {
    return true;
}

// --- SDCardManager ---

bool SDCardManager::begin() {
    // Configure SDMMC pins (1-bit mode).
    // GPIO1/GPIO2 are shared with the ST7701 SPI init, which is complete by now.
    SD_MMC.setPins(PIN_SD_CLK, PIN_SD_CMD, PIN_SD_D0);

    // Mount in 1-bit mode (oneWire=true).
    if (!SD_MMC.begin("/sdcard", /*oneWire=*/true)) {
        Serial.println("[sdcard] Mount failed — no card or wiring issue");
        _mounted = false;
        return false;
    }

    _mounted = true;
    Serial.printf("[sdcard] Mounted: %.2f GB (%llu sectors x %u bytes)\n",
        (double)SD_MMC.totalBytes() / 1024 / 1024 / 1024,
        (unsigned long long)SD_MMC.numSectors(),
        SD_MMC.sectorSize());
    return true;
}

void SDCardManager::beginUSB() {
    msc.vendorID("ESP32");
    msc.productID("CamelPad SD");
    msc.productRevision("1.0");
    msc.onRead(onRead);
    msc.onWrite(onWrite);
    msc.onStartStop(onStartStop);
    msc.mediaPresent(_mounted);

    if (_mounted) {
        msc.begin(SD_MMC.numSectors(), SD_MMC.sectorSize());
    } else {
        msc.begin(0, 512);
    }

    USB.begin();
    Serial.println("[sdcard] USB MSC started");
}

uint64_t SDCardManager::totalBytes() const {
    return _mounted ? SD_MMC.totalBytes() : 0;
}

uint64_t SDCardManager::usedBytes() const {
    return _mounted ? SD_MMC.usedBytes() : 0;
}
