#!/usr/bin/env node

/**
 * Get the platform-specific config path for camel-pad
 * Matches the logic in src/tray/config-store.ts
 */

const os = require('os');
const path = require('path');
const fs = require('fs');

function getConfigPath() {
  let base;

  if (process.platform === 'darwin') {
    base = path.join(os.homedir(), 'Library', 'Application Support', 'camel-pad');
  } else if (process.platform === 'win32') {
    base = path.join(process.env.APPDATA || path.join(os.homedir(), 'AppData', 'Roaming'), 'camel-pad');
  } else {
    base = path.join(os.homedir(), '.config', 'camel-pad');
  }

  // Create directory if it doesn't exist
  fs.mkdirSync(base, { recursive: true });

  return path.join(base, 'config.yaml');
}

module.exports = { getConfigPath };

// If run directly, print the path
if (require.main === module) {
  console.log(getConfigPath());
}
