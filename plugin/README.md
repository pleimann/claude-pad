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

All settings are stored in `config.yaml` which is read by both the camel-pad bridge and the plugin.

The plugin uses these settings from `config.yaml`:
- **server.host / server.port**: WebSocket endpoint (default: localhost:52914)
- **defaults.timeoutMs**: Response timeout in milliseconds (default: 30000)
- **keys.keyN.press**: Button mappings with action/label for notifications

Example `config.yaml` (see main README for full config):
```yaml
server:
  host: localhost
  port: 52914

defaults:
  timeoutMs: 30000

keys:
  key1:
    press:
      action: approve
      label: "Yes"
  key2:
    press:
      action: deny
      label: "No"
  key3:
    press:
      action: skip
      label: "Skip"
```

## Skills (Commands)

The plugin provides skills that can be invoked as commands or triggered contextually:

| Command                | Description                                                                       |
| ---------------------- | --------------------------------------------------------------------------------- |
| `/camel-pad:configure` | Interactive configuration - set device IDs, server port, timeout, key mappings    |
| `/camel-pad:test`      | Test connectivity with camel-pad device                                           |
| `/camel-pad:send`      | Send a custom message to the camel-pad display                                    |

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
