---
endpoint: ws://localhost:52914
timeout: 60
categories:
  - permission_request
  - task_complete
  - error
  - info
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
