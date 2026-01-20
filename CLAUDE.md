# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

- `go build ./cmd/camel-pad` - Build the binary
- `go test ./...` - Run all tests
- `go mod tidy` - Update dependencies
- `./camel-pad --list-devices` - List available HID devices
- `./camel-pad --config config.yaml` - Run with specific config file
- `./camel-pad --verbose` - Run with verbose logging
- `./camel-pad set-device` - Interactive device selection and config update
- `./camel-pad set-device 0x1234 0x5678` - Set device IDs directly

## Architecture

```
cmd/camel-pad/           # Entry point, CLI flags
internal/
├── config/               # YAML parsing, validation, hot-reload
│   ├── config.go         # Config types and loading
│   └── watcher.go        # fsnotify-based hot-reload
├── hid/                  # USB HID communication
│   ├── device.go         # HID device connection management
│   ├── discovery.go      # Device enumeration
│   └── protocol.go       # Message encoding/decoding
├── gesture/              # Gesture detection
│   ├── types.go          # Gesture type definitions
│   ├── detector.go       # Timing-based detection (single button)
│   └── engine.go         # State machine orchestration (chords)
├── action/               # Action mapping and execution
│   ├── mapper.go         # Gesture → key sequence lookup
│   └── executor.go       # Key parsing and PTY writing
├── pty/                  # PTY management
│   ├── manager.go        # PTY creation, lifecycle, TUI process
│   └── writer.go         # Key writing with optional delay
└── display/              # OLED display
    ├── manager.go        # Display update orchestration
    ├── renderer.go       # Text → frame buffer rendering
    └── protocol.go       # Frame encoding for HID
```

## Key Patterns

- Event-driven: HID events → gesture engine → action executor → PTY
- Config hot-reload via fsnotify
- Ring buffer for TUI output parsing (status extraction)
- 1-bit packed frame buffer for OLED (row-major, MSB first)
- Gesture state machine handles single/double/long press + chords
- Use charmbracelet libraries (huh, lipglass, bubbletea, etc) to build console UI

## Client Application

The corresponding client application running on the camel pad is in the ./pad directory. It is written in CircuitPython.