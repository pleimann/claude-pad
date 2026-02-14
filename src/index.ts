#!/usr/bin/env bun

import { resolve } from 'path';
import { SerialDevice } from './serial/device.js';
import { listPorts } from './serial/discovery.js';
import { GestureDetector } from './gesture/detector.js';
import { ConfigWatcher } from './config/watcher.js';
import { NotificationServer } from './websocket/server.js';
import { validateConfig } from './config/loader.js';
import type { NotificationMessage } from './types.js';

// Parse command line arguments
const args = process.argv.slice(2);
const command = args[0];

// Handle commands
if (command === 'list-devices') {
  console.log('Available serial ports:');
  const ports = await listPorts();
  for (const port of ports) {
    const vid = port.vendorId ? `0x${port.vendorId}` : '----';
    const pid = port.productId ? `0x${port.productId}` : '----';
    console.log(`  ${port.path}  Vendor: ${vid}  Product: ${pid}  ${port.manufacturer || ''}`);
  }
  process.exit(0);
}

const configPath = resolve(command || 'config.yaml');

// Load and validate config
const configWatcher = new ConfigWatcher(configPath);
const config = configWatcher.getConfig();

const errors = validateConfig(config);
if (errors.length > 0) {
  console.error('Configuration errors:');
  for (const error of errors) {
    console.error(`  - ${error}`);
  }
  process.exit(1);
}

console.log('camel-pad starting...');
console.log(`Config: ${configPath}`);
if (config.device.port) {
  console.log(`Device: port=${config.device.port}`);
} else {
  console.log(`Device: vendor=0x${config.device.vendorId?.toString(16)} product=0x${config.device.productId?.toString(16)}`);
}
console.log(`Server: ws://${config.server.host}:${config.server.port}`);

// Initialize components
const serialDevice = new SerialDevice({
  port: config.device.port,
  vendorId: config.device.vendorId,
  productId: config.device.productId,
});

const gestureDetector = new GestureDetector({
  longPressMs: config.gestures.longPressMs,
  doublePressMs: config.gestures.doublePressMs,
});

const notificationServer = new NotificationServer(config);

// Wire up events

// Serial button events → Gesture detector
serialDevice.on('button', ({ buttonId, pressed }) => {
  gestureDetector.handleButton(buttonId, pressed);
});

// Gesture events → Notification server
gestureDetector.on('gesture', ({ buttonId, gesture }) => {
  console.log(`Gesture: ${buttonId} ${gesture}`);
  const handled = notificationServer.handleGesture(buttonId, gesture);
  if (!handled && !notificationServer.hasPending()) {
    console.log('No pending notifications');
  }
});

// Notification events → Serial display
notificationServer.on('notification', (message: NotificationMessage) => {
  console.log(`Notification: ${message.text}`);
  serialDevice.sendText(message.text);
});

// Config reload events
configWatcher.on('reload', (newConfig) => {
  console.log('Applying new configuration...');

  gestureDetector.updateConfig({
    longPressMs: newConfig.gestures.longPressMs,
    doublePressMs: newConfig.gestures.doublePressMs,
  });

  notificationServer.updateConfig(newConfig);
});

// Graceful shutdown
function shutdown(): void {
  console.log('\nShutting down...');
  configWatcher.stop();
  notificationServer.stop();
  serialDevice.disconnect();
  gestureDetector.reset();
  process.exit(0);
}

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

// Start services
configWatcher.start();
notificationServer.start();
serialDevice.connect();

console.log('Ready. Press Ctrl+C to exit.');
