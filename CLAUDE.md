# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

- `bun install` - Install dependencies
- `bun run start` - Run the application
- `bun run dev` - Run with watch mode (auto-restart on changes)
- `bun run build` - Build for production
- `bun run src/index.ts list-devices` - List available serial devices
- `bun run src/index.ts config.yaml` - Run with specific config file

## Architecture

```
src/
├── index.ts              # Entry point, CLI, wiring
├── types.ts              # Shared type definitions, protocol constants
├── serial/
│   ├── device.ts         # Serial connection, read/write, reconnection
│   ├── discovery.ts      # Serial port enumeration by vendor/product ID
│   └── protocol.ts       # Binary frame building/parsing, checksum
├── gesture/
│   ├── types.ts          # Gesture type definitions
│   └── detector.ts       # Timing-based state machine (press/double/long)
├── websocket/
│   └── server.ts         # WebSocket server, notification queue, responses
└── config/
    ├── loader.ts         # YAML parsing, validation, defaults
    └── watcher.ts        # chokidar-based hot-reload
```

## Key Patterns

- Event-driven: Serial button events → gesture detector → notification server → response
- Config hot-reload via chokidar file watching
- Gesture state machine: idle → pressed → (longPress | waitDouble → (press | doublePressed → doublePress))
- WebSocket notification queue with oldest-first response matching
- Automatic serial reconnection on disconnect

## Data Flow

```
┌─────────────────┐   Serial     ┌─────────────────┐    WebSocket    ┌─────────────────┐
│   Macropad FW   │◄────────────►│   camel-pad     │◄───────────────►│  Claude Code    │
│    (Arduino)    │  (CDC ACM)   │  (TypeScript)   │                 │  Plugin         │
└─────────────────┘              └─────────────────┘                 └─────────────────┘
     Buttons                          Bridge                          Notifications
     Display                       Gesture detection
                                   Key mapping
                                   Config (hot-reload)
```

## Communication Protocol

The firmware uses USB CDC ACM (serial) via TinyUSB with length-prefixed binary framing:

```
[0xAA] [LEN_HI] [LEN_LO] [MSG_TYPE] [PAYLOAD...] [CHECKSUM_XOR]
```

Message types:

- `0x01` Host→Device: Display text (UTF-8 payload)
- `0x02` Device→Host: Button event (`[button_id, pressed]`)
- `0x03` Host→Device: Set LEDs (`[idx, R, G, B]` repeated)
- `0x04` Host→Device: Status text (UTF-8 payload)
- `0x05` Host→Device: Clear display
- `0x06` Host→Device: Set button labels (`[len, label...]` x 4)
- `0x07` Device→Host: Heartbeat (`[status]`)

The bridge uses raw file I/O (`fs.openSync` + `stty`) for serial communication — no native N-API dependencies, compatible with Bun.

## WebSocket Protocol

- Notification: `{"type": "notification", "id": "uuid", "text": "...", "category": "..."}`
- Response: `{"type": "response", "id": "uuid", "action": "approve", "label": "Yes"}`
- Error: `{"type": "error", "id": "uuid", "error": "Timeout"}`

## Device

### Waveshare ESP32-S3-LCD-3.16

- **MCU**: ESP32-S3 with 16MB flash, octal PSRAM at 80MHz
- **Communication**: USB CDC ACM
- **Display**: Waveshare 3.16" 320x820 MIPI RGB (rotated 90°)
- **Input**: ATtiny1616 Adafruit Seesaw (I2C 0x49) with 4 buttons (pins 11-14)
- **LEDs**: 4 NeoPixels via Seesaw (pin 2)
- **USB**: TinyUSB CDC ACM for serial communication

### Firmware

The firmware is in `./firmware/`, built with PlatformIO (Arduino framework on ESP-IDF).

**Build**: `cd firmware && .venv/bin/platformio run`
**Flash**: `cd firmware && .venv/bin/platformio run --target upload`
**Monitor**: `cd firmware && .venv/bin/platformio device monitor`

Note: Uses a Python 3.13 venv (`.venv/`) because ESP-IDF doesn't support Python 3.14+.

Key dependencies:

- **LVGL v9** — UI framework (software-rotated 820x320 landscape)
- **ESP-IDF native `esp_lcd`** — RGB panel driver with bounce buffers
- **Adafruit Seesaw Library** — Button input and NeoPixel control over I2C

```
firmware/src/
├── main.cpp                    # setup/loop, wiring
├── config.h                    # Pin definitions, protocol constants
├── lv_conf.h                   # LVGL configuration
├── display/
│   ├── display_config.h        # ST7701 init sequence (from Waveshare examples)
│   ├── display_manager.h       # Display manager API
│   └── display_manager.cpp     # LVGL-based UI layout and rendering
├── seesaw/
│   ├── seesaw_manager.h        # Seesaw manager API
│   └── seesaw_manager.cpp      # Button polling, NeoPixel control
├── comms/
│   ├── protocol.h              # Frame format, checksum
│   ├── serial_comms.h          # Serial communication API
│   └── serial_comms.cpp        # USB CDC message parsing/sending
└── vendor/
    ├── st7701_bsp/             # ST7701 panel driver (esp_lcd_new_panel_st7701)
    └── io_additions/           # 3-wire SPI IO for ST7701 init commands
```

The hardware device is a `Waveshare ESP32-S3-LCD-3.16`. Documentation: https://www.waveshare.com/wiki/ESP32-S3-LCD-3.16

Manufacturer examples are in `../ESP32-S3-LCD-3.16-Demo`. The ST7701 init sequence in `display_config.h` was extracted from those examples.

### Pin Mapping

| Interface            | Pins                    |
| -------------------- | ----------------------- |
| SPI (Display Config) | CLK: GPIO2, MOSI: GPIO1 |
| I2C (Seesaw)         | SDA: GPIO15, SCL: GPIO7 |

### Seesaw Pins

- Pin 2: NeoPixels (x4)
- Pin 11-14: Key01-Key04

## Claude Code Plugin

The camel-pad-bridge plugin in ./camel-pad-bridge integrates with Claude Code to forward notifications to the application.
