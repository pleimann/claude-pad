#!/usr/bin/env node

/**
 * WebSocket client for camel-pad notifications
 *
 * Reads notification JSON from stdin, sends to camel-pad via WebSocket,
 * waits for response, and outputs result JSON to stdout.
 */

const WebSocket = require('ws');
const fs = require('fs');
const path = require('path');
const { randomUUID } = require('crypto');

// Read stdin
let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => input += chunk);
process.stdin.on('end', async () => {
  try {
    await main(JSON.parse(input));
  } catch (err) {
    console.error(JSON.stringify({ error: err.message }));
    process.exit(2);
  }
});

/**
 * Parse YAML frontmatter from .local.md file
 */
function parseConfig(configPath) {
  if (!fs.existsSync(configPath)) {
    return null;
  }

  const content = fs.readFileSync(configPath, 'utf8');
  const match = content.match(/^---\n([\s\S]*?)\n---/);
  if (!match) return null;

  // Simple YAML parsing for our config format
  const yaml = match[1];
  const config = {};

  // Parse endpoint
  const endpointMatch = yaml.match(/endpoint:\s*(.+)/);
  if (endpointMatch) config.endpoint = endpointMatch[1].trim();

  // Parse timeout
  const timeoutMatch = yaml.match(/timeout:\s*(\d+)/);
  if (timeoutMatch) config.timeout = parseInt(timeoutMatch[1], 10);

  // Parse categories
  const categoriesMatch = yaml.match(/categories:\n((?:\s+-\s+.+\n?)+)/);
  if (categoriesMatch) {
    config.categories = categoriesMatch[1]
      .split('\n')
      .map(line => line.replace(/^\s+-\s+/, '').trim())
      .filter(Boolean);
  }

  // Parse keys
  const keysMatch = yaml.match(/keys:\n((?:\s+\w+:\n(?:\s+\w+:\s*.+\n?)+)+)/);
  if (keysMatch) {
    config.keys = {};
    const keysBlock = keysMatch[1];
    const keyMatches = keysBlock.matchAll(/(\w+):\n\s+action:\s*(\w+)\n\s+label:\s*"?([^"\n]+)"?/g);
    for (const m of keyMatches) {
      config.keys[m[1]] = { action: m[2], label: m[3].trim() };
    }
  }

  return config;
}

async function main(hookInput) {
  const projectDir = process.env.CLAUDE_PROJECT_DIR || process.cwd();
  const configPath = path.join(projectDir, '.claude', 'camel-pad.local.md');

  const config = parseConfig(configPath);
  if (!config) {
    // No config, silently pass through (not configured)
    console.log(JSON.stringify({ continue: true, suppressOutput: true }));
    return;
  }

  if (!config.endpoint) {
    console.error(JSON.stringify({ error: 'No endpoint configured' }));
    process.exit(2);
  }

  if (!config.timeout) {
    console.error(JSON.stringify({ error: 'No timeout configured' }));
    process.exit(2);
  }

  // Extract notification info from hook input
  const notificationText = hookInput.notification_text || hookInput.message || '';
  const notificationCategory = hookInput.notification_category || hookInput.category || 'unknown';

  // Check if this category should be forwarded
  if (config.categories && config.categories.length > 0) {
    if (!config.categories.includes(notificationCategory)) {
      // Category not in filter list, pass through silently
      console.log(JSON.stringify({ continue: true, suppressOutput: true }));
      return;
    }
  }

  // Connect to WebSocket and send notification
  const messageId = randomUUID();
  const ws = new WebSocket(config.endpoint);

  const timeoutMs = config.timeout * 1000;
  let timeoutId;

  const result = await new Promise((resolve, reject) => {
    timeoutId = setTimeout(() => {
      ws.close();
      reject(new Error(`Timeout waiting for response after ${config.timeout}s`));
    }, timeoutMs);

    ws.on('open', () => {
      ws.send(JSON.stringify({
        type: 'notification',
        id: messageId,
        text: notificationText,
        category: notificationCategory
      }));
    });

    ws.on('message', (data) => {
      try {
        const response = JSON.parse(data.toString());
        if (response.type === 'response' && response.id === messageId) {
          clearTimeout(timeoutId);
          ws.close();
          resolve(response);
        }
      } catch (e) {
        // Ignore parse errors, wait for valid response
      }
    });

    ws.on('error', (err) => {
      clearTimeout(timeoutId);
      reject(new Error(`WebSocket error: ${err.message}`));
    });

    ws.on('close', () => {
      clearTimeout(timeoutId);
    });
  });

  // Return the response to Claude Code
  console.log(JSON.stringify({
    continue: true,
    systemMessage: `User responded via camel-pad: ${result.action} (${result.label || ''})`,
    hookSpecificOutput: {
      action: result.action,
      label: result.label
    }
  }));
}
