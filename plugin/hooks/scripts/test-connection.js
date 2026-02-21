#!/usr/bin/env node

/**
 * Test WebSocket connectivity to camel-pad
 */

const WebSocket = require('ws');
const fs = require('fs');
const path = require('path');
const { randomUUID } = require('crypto');

const projectDir = process.env.CLAUDE_PROJECT_DIR || process.cwd();
const configPath = path.join(projectDir, '.claude', 'camel-pad.local.md');

function parseConfig(configPath) {
  if (!fs.existsSync(configPath)) {
    return null;
  }

  const content = fs.readFileSync(configPath, 'utf8');
  const match = content.match(/^---\n([\s\S]*?)\n---/);
  if (!match) return null;

  const yaml = match[1];
  const config = {};

  const endpointMatch = yaml.match(/endpoint:\s*(.+)/);
  if (endpointMatch) config.endpoint = endpointMatch[1].trim();

  const timeoutMatch = yaml.match(/timeout:\s*(\d+)/);
  if (timeoutMatch) config.timeout = parseInt(timeoutMatch[1], 10);

  return config;
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
