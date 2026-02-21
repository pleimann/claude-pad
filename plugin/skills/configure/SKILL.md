---
description: Use this skill when the user asks to "configure camel-pad", "set up camel-pad", "connect to macropad", "select HID device", or needs to configure the camel-pad bridge settings including device selection, endpoint URL, timeout, notification categories, or key mappings.
---

# Configure camel-pad Bridge

Guide the user through setting up the camel-pad bridge configuration.

## Configuration Files

There are two configuration files:

- **`config.yaml`** - Device settings for the camel-pad TypeScript bridge process
- **`.claude/camel-pad.local.md`** - Plugin settings for the Claude Code integration

## Instructions

1. Read both configuration files to get current values:
   - Read `config.yaml` for current device settings (vendorId, productId)
   - Read `.claude/camel-pad.local.md` for plugin settings if it exists

2. Use AskUserQuestion to gather configuration:

   **HID Device Selection:**
   - First, run the device listing script to get available devices:
     ```bash
     node ${CLAUDE_PLUGIN_ROOT}/hooks/scripts/list-devices.js
     ```
   - Parse the JSON output to get the list of devices
   - Look for a device whose name starts with "CamelPad" - this is the camel-pad device
   - If a CamelPad device is found:
     - Auto-select it and inform the user: "Found CamelPad device: [name] (Vendor: [vendor] Product: [product])"
     - Ask: "Use this device?" with options "Yes (Recommended)", "Choose different device"
     - If "Yes", use those vendorId/productId values
   - If no CamelPad device found, or user chooses "Choose different device":
     - Question: "Which HID device is your camel-pad?"
     - Options: Build from device list, showing: "[name] ([manufacturer]) - Vendor: [vendor] Product: [product]"
     - Add option "Enter manually" for cases where the device isn't detected
   - If "Enter manually", ask for vendorId and productId as hex values (e.g., 0x1234)
   - Store the selected vendorId and productId

   **Endpoint URL:**
   - Question: "What is the WebSocket endpoint for camel-pad?"
   - Options: "ws://localhost:52914" (default), "Custom URL"
   - If custom, ask for the full URL

   **Timeout:**
   - Question: "How many seconds should we wait for a response?"
   - Options: "30 seconds", "60 seconds", "Custom"
   - This is required, there is no default

   **Notification Categories:**
   - Question: "Which notification categories should be forwarded to camel-pad?"
   - Options (multi-select): "permission_request", "task_complete", "error", "info", "All categories"

   **Key Mappings:**
   - Question: "Configure key mappings? (You can edit the config file later for advanced setup)"
   - Options: "Use defaults (key1=approve, key2=deny, key3=skip)", "Configure now", "Skip for now"
   - If "Configure now", ask for each key's action and label

3. Write device settings to `config.yaml`:
   - Use the Edit tool to update the device section in config.yaml
   - Update vendorId and productId values (use uppercase hex format like 0x303A)
   - Example edit - replace:
     ```yaml
     device:
       # USB HID device identifiers
       # Use `camel-pad --list-devices` to find your device
       vendorId: 0x1234
       productId: 0x5678
     ```
     with the new values

4. Write plugin settings to `.claude/camel-pad.local.md`:

```markdown
---
endpoint: [endpoint_url]
timeout: [timeout_seconds]
categories:
  - [category1]
  - [category2]
keys:
  key1:
    action: [action]
    label: "[label]"
  key2:
    action: [action]
    label: "[label]"
  key3:
    action: [action]
    label: "[label]"
---

# camel-pad Bridge Plugin Configuration

This file configures the camel-pad Claude Code plugin.
Edit the YAML frontmatter above to change settings.

## Settings Reference

- `endpoint`: WebSocket URL for the camel-pad bridge server
- `timeout`: Seconds to wait for a response from the device
- `categories`: Which notification types to forward to camel-pad
- `keys`: Button action mappings (action value and display label)

## Note

Device settings (vendorId, productId) are stored in `config.yaml`
which is read by the camel-pad bridge process.
```

5. Confirm: "Configuration saved:
   - Device settings → `config.yaml`
   - Plugin settings → `.claude/camel-pad.local.md`"

6. Suggest: "Run `/camel-pad:test` to verify connectivity."
