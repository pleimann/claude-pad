# Macropad Middleware Implementation Plan

## Overview

A Go middleware application that bridges a custom USB HID macropad with a TUI application via PTY. Supports configurable gesture detection (single/double/long-press, chords) and bidirectional communication for OLED status display.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Middleware (Go)                              │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────────┐  │
│  │   HID    │──►│ Gesture  │──►│  Action  │──►│   PTY Manager    │  │
│  │  Driver  │   │  Engine  │   │  Mapper  │   │ (spawns TUI)     │  │
│  └──────────┘   └──────────┘   └──────────┘   └──────────────────┘  │
│       ▲                                              │               │
│       │              ┌──────────┐                    ▼               │
│       └──────────────│ Display  │◄───────────── TUI stdout          │
│      OLED updates    │ Manager  │    (parsed for status)            │
│                      └──────────┘                                    │
│                           ▲                                          │
│                      ┌────┴─────┐                                    │
│                      │  Config  │◄─── config.toml (hot-reload)       │
│                      │  Loader  │                                    │
│                      └──────────┘                                    │
└─────────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
claude-pad/
├── cmd/
│   └── claude-pad/
│       └── main.go              # Entry point, CLI flags
├── internal/
│   ├── config/
│   │   ├── config.go            # TOML parsing, validation
│   │   └── watcher.go           # Hot-reload via fsnotify
│   ├── hid/
│   │   ├── device.go            # HID connection management
│   │   ├── protocol.go          # Message encoding/decoding
│   │   └── discovery.go         # Device enumeration
│   ├── gesture/
│   │   ├── engine.go            # Gesture state machine
│   │   ├── detector.go          # Timing-based detection
│   │   └── types.go             # Gesture types (press, double, long, chord)
│   ├── action/
│   │   ├── mapper.go            # Gesture → action lookup
│   │   └── executor.go          # Execute key sequences
│   ├── pty/
│   │   ├── manager.go           # PTY creation, lifecycle
│   │   └── writer.go            # Write keystrokes to PTY
│   └── display/
│       ├── manager.go           # Display update orchestration
│       ├── renderer.go          # Status → frame buffer
│       └── protocol.go          # Frame encoding for OLED
├── config.example.toml
├── go.mod
└── CLAUDE.md
```

## Component Details

### 1. Config Schema (TOML)

```toml
[device]
vendor_id = 0x1234
product_id = 0x5678
poll_interval_ms = 10

[timing]
double_press_window_ms = 300
long_press_threshold_ms = 500
chord_window_ms = 50

[tui]
command = "my-tui-app"
args = ["--flag", "value"]
working_dir = "/path/to/dir"  # optional

# Button mappings - index is 0-based button number
[[buttons]]
index = 0
name = "btn_a"  # optional, for display/logging

  [buttons.press]
  keys = ["ctrl+c"]

  [buttons.double_press]
  keys = ["ctrl+z"]

  [buttons.long_press]
  keys = ["q", "enter"]  # sequence of keys

[[buttons]]
index = 1
name = "btn_b"

  [buttons.press]
  keys = ["down", "down", "down"]

# Chord mappings - multiple buttons pressed together
[[chords]]
buttons = [0, 1]  # btn_a + btn_b together
keys = ["ctrl+alt+delete"]

# Display configuration
[display]
width = 128
height = 64
update_interval_ms = 100

[[display.regions]]
name = "status"
x = 0
y = 0
width = 128
height = 32
source = "tui_status"  # or "static", "system"

[[display.regions]]
name = "mode"
x = 0
y = 32
width = 128
height = 32
source = "static"
content = "Ready"
```

### 2. HID Protocol (Device ↔ Host)

**Device → Host (Button Events):**
```
Byte 0: Report ID (0x01)
Byte 1: Event type (0x01=press, 0x02=release)
Byte 2-3: Button bitmask (16 buttons max, little-endian)
Byte 4-7: Timestamp (ms since boot, little-endian u32)
```

**Host → Device (Display Updates):**
```
Byte 0: Report ID (0x02)
Byte 1: Command (0x01=full frame, 0x02=partial, 0x03=clear)
Byte 2-3: X offset (for partial)
Byte 4-5: Y offset (for partial)
Byte 6-7: Width
Byte 8-9: Height
Byte 10+: Pixel data (1-bit packed, row-major)
```

### 3. Gesture Detection State Machine

```
                    ┌─────────────────────────────────────┐
                    │                                     │
                    ▼                                     │
┌──────┐  press  ┌──────────┐  release  ┌─────────────┐  │ timeout
│ IDLE │────────►│ PRESSED  │──────────►│ WAIT_DOUBLE │──┤
└──────┘         └──────────┘           └─────────────┘  │
    ▲                │                        │          │
    │                │ hold > threshold       │ press    │
    │                ▼                        ▼          │
    │           ┌──────────┐           ┌─────────────┐   │
    │           │LONG_PRESS│           │DOUBLE_PRESS │   │
    │           └──────────┘           └─────────────┘   │
    │                │                        │          │
    └────────────────┴────────────────────────┴──────────┘
                         (emit gesture)
```

**Chord Detection:**
- Track all button states in a sliding window
- If multiple buttons are pressed within `chord_window_ms`, treat as chord
- Match against configured chord combinations

### 4. Key Sequence Format

Keys in config use a simple format:
- Modifiers: `ctrl+`, `alt+`, `shift+`, `meta+`
- Special keys: `enter`, `tab`, `esc`, `up`, `down`, `left`, `right`, `backspace`, `delete`, `home`, `end`, `pageup`, `pagedown`, `f1`-`f12`
- Printable: single character like `a`, `1`, `/`
- Examples: `ctrl+c`, `ctrl+shift+z`, `alt+f4`, `enter`

### 5. PTY Manager

- Uses `creack/pty` package to create pseudo-terminal
- Spawns TUI command with PTY as stdin/stdout/stderr
- Maintains goroutines for:
  - Reading TUI output (for status extraction)
  - Writing keystrokes from action executor
- Handles TUI process lifecycle (restart on crash, clean shutdown)

### 6. Display Manager

- Parses TUI stdout for status markers (configurable regex/prefix)
- Alternative: TUI writes to a separate status file that middleware watches
- Renders status text to 1-bit frame buffer
- Sends frames to device via HID at configured interval
- Supports regions with different update sources

## Dependencies

```go
require (
    github.com/BurntSushi/toml v1.3.0      // TOML parsing
    github.com/creack/pty v1.1.21          // PTY management
    github.com/fsnotify/fsnotify v1.7.0    // Config hot-reload
    github.com/karalabe/hid v1.0.0         // USB HID communication
    golang.org/x/image v0.15.0             // Font rendering for display
)
```

## Implementation Order

### Phase 1: Core Infrastructure
1. `cmd/claude-pad/main.go` - CLI entry point with flags
2. `internal/config/config.go` - TOML parsing and types
3. `internal/hid/device.go` - Basic HID connection
4. `internal/hid/protocol.go` - Message types

### Phase 2: Input Pipeline
5. `internal/gesture/types.go` - Gesture type definitions
6. `internal/gesture/engine.go` - State machine implementation
7. `internal/gesture/detector.go` - Timing logic
8. `internal/action/mapper.go` - Config-based gesture→action lookup

### Phase 3: Output Pipeline
9. `internal/pty/manager.go` - PTY spawn and lifecycle
10. `internal/pty/writer.go` - Key sequence writing
11. `internal/action/executor.go` - Execute mapped actions

### Phase 4: Display
12. `internal/display/renderer.go` - Text to frame buffer
13. `internal/display/protocol.go` - Frame encoding
14. `internal/display/manager.go` - Orchestration and status parsing

### Phase 5: Polish
15. `internal/config/watcher.go` - Hot-reload support
16. `internal/hid/discovery.go` - Device enumeration/reconnection
17. Logging, error handling, graceful shutdown

## Verification

1. **Unit tests**: Gesture detection timing, config parsing, protocol encoding
2. **Integration test**: Mock HID device → gesture → PTY write
3. **Manual testing**:
   - Connect macropad, verify device discovery
   - Press buttons, verify TUI receives keystrokes
   - Verify gesture detection (double-press, long-press, chords)
   - Modify config, verify hot-reload
   - Check OLED displays status updates
