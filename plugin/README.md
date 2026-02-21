# camel-pad

Bridge Claude Code notifications to the camel-pad device, displaying messages on the OLED screen and receiving responses via physical key presses.

## Features

- Forward Claude Code notifications to camel-pad's OLED display
- Receive user responses via macropad key presses
- Configurable notification filtering by category
- Customizable key-to-action mappings
- WebSocket-based communication with camel-pad

## Prerequisites

- Node.js 18+ (for WebSocket communication)
- camel-pad running with HTTP/WebSocket API enabled on port 52914

## Installation

1. Enable the plugin in Claude Code:

   ```bash
   claude --plugin-dir /path/to/camel-pad
   ```

2. Configure the plugin:
   ```bash
   /camel-pad:configure
   ```

## Configuration

Settings are stored in `.claude/camel-pad.local.md` with YAML frontmatter:

```yaml
---
endpoint: ws://localhost:52914
timeout: 30
categories:
  - permission_request
  - task_complete
  - error
keys:
  key1:
    action: approve
    label: "Yes"
  key2:
    action: deny
    label: "No"
  key3:
    action: skip
    label: "Skip"
---
```

### Configuration Options

| Option       | Description                             |
| ------------ | --------------------------------------- |
| `endpoint`   | WebSocket URL for camel-pad API         |
| `timeout`    | Seconds to wait for response (required) |
| `categories` | Notification categories to forward      |
| `keys`       | Key-to-action mappings                  |

## Skills (Commands)

The plugin provides skills that can be invoked as commands or triggered contextually:

| Command                | Description                                                                                    |
| ---------------------- | ---------------------------------------------------------------------------------------------- |
| `/camel-pad:configure` | Interactive configuration - select HID device, set endpoint, timeout, categories, key mappings |
| `/camel-pad:test`      | Test connectivity with camel-pad device                                                        |
| `/camel-pad:send`      | Send a custom message to the camel-pad display                                                 |

Skills are also triggered when you ask things like:

- "Configure camel-pad" or "Set up the macropad"
- "Test camel-pad connection"
- "Send a message to camel-pad"

## WebSocket Protocol

Messages sent to camel-pad:

```json
{ "type": "notification", "id": "uuid", "text": "...", "category": "..." }
```

Responses from camel-pad:

```json
{ "type": "response", "id": "uuid", "action": "approve", "label": "Yes" }
```

## Development

This plugin is part of the camel-pad project. The camel-pad application must expose a WebSocket API that accepts notification messages and returns user responses.
