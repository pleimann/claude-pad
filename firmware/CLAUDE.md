# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
.venv/bin/platformio run                    # Build
.venv/bin/platformio run --target upload    # Flash to device
.venv/bin/platformio device monitor         # Serial monitor (115200 baud)
.venv/bin/platformio run --target clean     # Clean build
```

Note: Uses Python 3.13 venv because ESP-IDF doesn't support Python 3.14+.

## Architecture

```
src/
├── main.cpp              # Component wiring, callbacks, 10ms poll loop
├── config.h              # All pin definitions and protocol constants
├── display/
│   ├── display_config.h  # LGFX_CamelPad class: ST7701 3-wire SPI init + RGB bus config
│   ├── display_manager.h/.cpp  # UI layout (status bar, notification, button labels)
├── seesaw/
│   └── seesaw_manager.h/.cpp   # Button debounce (50ms + 3-read stability), NeoPixel control
└── comms/
    ├── protocol.h        # Frame builder, checksum calculation
    └── serial_comms.h/.cpp     # State machine parser, callback dispatch
```

## Key Patterns

**Callback Pattern**: All managers register callbacks with `comms`/`seesaw` for decoupling.

**Sprite Double Buffering**: Full-screen `LGFX_Sprite` in PSRAM (~525KB). Draw to sprite, push to display to avoid flicker.

**Dirty Flag**: Display only redraws when `_dirty=true`.

**State Machine Parsing**: SerialComms uses 5-state machine for frame parsing:

```
WAIT_START → READ_LEN_HI → READ_LEN_LO → READ_BODY → READ_CHECKSUM → dispatch
```

**Button Debounce**: 3 consecutive consistent reads required over 50ms window.

## Display Configuration

The display uses a custom ST7701 initialization via 3-wire SPI bit-bang, then LovyanGFX handles the RGB parallel bus. The init sequence in `display_config.h` was extracted from Waveshare's LVGL example (`../ESP32-S3-LCD-3.16-Demo/Arduino/examples/08_LVGL_V9_Test/lvgl_port.c`).

Display is 320x820 native, rotated 90° to 820x320 landscape. Layout:

- Status bar: 30px top
- Notification: 220px middle
- Button labels: 70px bottom (4 zones, 205px each)

## Serial Protocol

Length-prefixed binary framing over USB CDC ACM:

```
[0xAA] [LEN_HI] [LEN_LO] [MSG_TYPE] [PAYLOAD...] [CHECKSUM_XOR]
```

Message types defined in `config.h`:

- `0x01` DISPLAY_TEXT: UTF-8 notification text
- `0x02` BUTTON: `[button_id, pressed]` (device→host)
- `0x03` SET_LEDS: `[idx, R, G, B]` repeated
- `0x04` STATUS: UTF-8 status bar text
- `0x05` CLEAR: Clear display to idle
- `0x06` SET_LABELS: `[len, label...]` x 4
- `0x07` HEARTBEAT: `[status]` (device→host)

## Hardware Reference

**Board**: Waveshare ESP32-S3-LCD-3.16 (custom board definition in `boards/`)

| Interface              | Pins                                    |
| ---------------------- | --------------------------------------- |
| ST7701 SPI (init only) | CLK=GPIO2, MOSI=GPIO1, CS=GPIO0         |
| RGB Parallel (16-bit)  | See `display_config.h` for full mapping |
| I2C (Seesaw)           | SDA=GPIO15, SCL=GPIO7, addr=0x49        |
| Backlight              | GPIO6 (PWM, inverted)                   |

Seesaw pins: Buttons=11-14, NeoPixels=pin 2 (4x WS2812B)
