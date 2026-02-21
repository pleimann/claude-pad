---
description: Send a custom message to the camel-pad display and wait for a button response
---

# Send Message to camel-pad

Send a custom message to the camel-pad device and wait for a response.

## Instructions

1. Determine the message to send:
   - If user provided a message in their request, use that
   - If no message provided, ask: "What message would you like to send to camel-pad?"

2. Read the configuration from `.claude/camel-pad.local.md`
   - If no configuration exists, inform the user to run `/camel-pad:configure` first

3. Send the message using the send script:

   ```bash
   node ${CLAUDE_PLUGIN_ROOT}/hooks/scripts/send-message.js "<message>"
   ```

4. Report the result:
   - **Success**: "Message sent to camel-pad. Response: [action] ([label])"
   - **Failure**: "Failed to send message: [error]"

## Examples

- "Send 'Ready to deploy?' to camel-pad"
- "Display 'Approve migration?' on the macropad"
- "Ask for approval on camel-pad: Continue with build?"
