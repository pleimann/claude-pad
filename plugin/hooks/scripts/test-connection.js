#!/usr/bin/env node

/**
 * Test WebSocket connectivity to camel-pad
 */

const WebSocket = require('ws');
const fs = require('fs');
const path = require('path');
const { randomUUID } = require('crypto');
const yaml = require('yaml');

const projectDir = process.env.CLAUDE_PROJECT_DIR || process.cwd();
const configPath = path.join(projectDir, 'config.yaml');

function parseConfig(configPath) {
  if (!fs.existsSync(configPath)) {
    return null;
  }

  try {
    const content = fs.readFileSync(configPath, 'utf8');
    const config = yaml.parse(content);

    if (!config || !config.server) {
      return null;
    }

    // Extract settings needed for WebSocket connection
    const endpoint = `ws://${config.server.host || 'localhost'}:${config.server.port || 52914}`;
    const timeout = config.defaults?.timeoutMs ? Math.floor(config.defaults.timeoutMs / 1000) : 30;

    return { endpoint, timeout };
  } catch (err) {
    console.error('Error parsing config:', err.message);
    return null;
  }
}

async function main() {
  const config = parseConfig(configPath);

  if (!config) {
    console.log(JSON.stringify({
      success: false,
      error: 'No configuration found. Run /camel-pad:configure first.'
    }));
    process.exit(1);
  }

  if (!config.endpoint) {
    console.log(JSON.stringify({
      success: false,
      error: 'No endpoint configured'
    }));
    process.exit(1);
  }

  const timeout = (config.timeout || 10) * 1000;
  const messageId = randomUUID();

  try {
    const ws = new WebSocket(config.endpoint);

    const result = await new Promise((resolve, reject) => {
      const timeoutId = setTimeout(() => {
        ws.close();
        reject(new Error('Connection timeout'));
      }, timeout);

      ws.on('open', () => {
        ws.send(JSON.stringify({
          type: 'test',
          id: messageId,
          text: 'Test message from Claude Code',
          category: 'test'
        }));
      });

      ws.on('message', (data) => {
        try {
          const response = JSON.parse(data.toString());
          if (response.id === messageId) {
            clearTimeout(timeoutId);
            ws.close();
            resolve(response);
          }
        } catch (e) {
          // Ignore parse errors
        }
      });

      ws.on('error', (err) => {
        clearTimeout(timeoutId);
        reject(err);
      });
    });

    console.log(JSON.stringify({
      success: true,
      endpoint: config.endpoint,
      response: result
    }));

  } catch (err) {
    console.log(JSON.stringify({
      success: false,
      endpoint: config.endpoint,
      error: err.message
    }));
    process.exit(1);
  }
}

main();
