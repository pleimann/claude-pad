---
description: Use this skill when the user asks to "configure camel-pad", "set up camel-pad", "connect to macropad", or needs to configure the camel-pad bridge settings including device, server port, timeout, or key mappings.
---

# Configure camel-pad Bridge

Guide the user through setting up the camel-pad bridge in `config.yaml`.

## Configuration File

All settings are stored in the platform-specific config file which is read by the camel-pad bridge process:
- macOS: `~/Library/Application Support/camel-pad/config.yaml`
- Windows: `%APPDATA%\camel-pad\config.yaml`
- Linux: `~/.config/camel-pad/config.yaml`

## Instructions

1. Get the config path by running:
   ```bash
   node plugin/hooks/scripts/config-path.js
   ```
   Then read that file to get current values

2. Use AskUserQuestion to gather configuration:

   **Device Settings:**
   - Show the user the current `device.vendorId` and `device.productId`
   - Question: "The bridge uses these device identifiers to find the serial port. The defaults (0x303A / 0x1001) work for the Waveshare ESP32-S3-LCD-3.16. Do you need to change them?"
   - Options: "Use defaults (Recommended)", "Enter custom VID/PID"
   - If "Enter custom VID/PID", ask for vendorId and productId as hex values (e.g., 0x1234)

   **Server Settings:**
   - Question: "What port should the WebSocket server listen on?"
   - Options: "52914 (Recommended)", "Custom port"
   - If custom, ask for the port number

   **Timeout:**
   - Question: "How many milliseconds should we wait for a response?"
   - Options: "30000 (30 seconds)", "60000 (60 seconds)", "Custom"

   **Key Mappings (for Claude Code notifications):**
   - Question: "Configure key1 press action for responding to notifications?"
   - Options: "approve/Yes (Recommended)", "deny/No", "skip/Skip", "Custom"
   - Repeat for key2 and key3

3. Update the config file at the path obtained in step 1:
   - Update device.vendorId and device.productId if changed
   - Update server.port if changed
   - Update defaults.timeoutMs
   - Update keys.key1.press, keys.key2.press, keys.key3.press with action/label

4. Confirm: "Configuration saved to [config_path]. Run `/camel-pad:test` to verify connectivity."
