---
description: Use this skill when the user asks to "test camel-pad", "check camel-pad connection", "verify macropad connectivity", or wants to test that the camel-pad bridge is working correctly.
---

# Test camel-pad Bridge Connectivity

Send a test message to the camel-pad device to verify WebSocket connectivity.

## Instructions

1. Read the configuration from `.claude/camel-pad.local.md` to get the endpoint URL

2. If no configuration exists, inform the user:
   - "No configuration found. Run `/camel-pad:configure` first to set up the connection."

3. If configuration exists, use Bash to run the test script:

   ```bash
   node ${CLAUDE_PLUGIN_ROOT}/hooks/scripts/test-connection.js
   ```

4. Report the result:
   - **Success**: "Connected to camel-pad at [endpoint]. Response received: [action]"
   - **Failure**: "Failed to connect to camel-pad: [error message]"

## Tips

- Ensure camel-pad is running with the WebSocket API enabled
- Check that the endpoint URL in your configuration is correct
- Default port is 52914
