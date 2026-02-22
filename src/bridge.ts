import { SerialDevice } from './serial/device.js';
import { GestureDetector } from './gesture/detector.js';
import { ConfigWatcher } from './config/watcher.js';
import { NotificationServer } from './websocket/server.js';
import { validateConfig } from './config/loader.js';
import type { NotificationMessage } from './types.js';

export interface BridgeStatus {
  connected: boolean;
  portPath: string | null;
  pendingCount: number;
}

export interface BridgeHandle {
  shutdown(): void;
  getStatus(): BridgeStatus;
  onStatusChange(cb: (status: BridgeStatus) => void): void;
  sendText(text: string): boolean;
  sendStatus(text: string): boolean;
  clearDisplay(): boolean;
  sendLeds(leds: Array<{ index: number; r: number; g: number; b: number }>): boolean;
  sendLabels(labels: string[]): boolean;
}

function remapButtonIndex(i: number, h: 'left' | 'right'): number {
  return h === 'right' ? 3 - i : i;
}

export async function startBridge(configPath: string): Promise<BridgeHandle> {
  const configWatcher = new ConfigWatcher(configPath);
  const config = configWatcher.getConfig();

  const errors = validateConfig(config);
  if (errors.length > 0) {
    throw new Error(`Configuration errors:\n${errors.map(e => `  - ${e}`).join('\n')}`);
  }

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

  const statusListeners: Array<(status: BridgeStatus) => void> = [];
  let connected = false;
  let portPath: string | null = null;
  let pingInterval: ReturnType<typeof setInterval> | null = null;
  let handedness = config.handedness;

  function emitStatus() {
    const status: BridgeStatus = {
      connected,
      portPath,
      pendingCount: notificationServer.hasPending() ? 1 : 0,
    };
    for (const cb of statusListeners) cb(status);
  }

  // Serial button events → Gesture detector (with handedness remapping)
  serialDevice.on('button', ({ buttonId, pressed }) => {
    const num = parseInt(buttonId.replace('key', ''));
    const remapped = `key${remapButtonIndex(num, handedness)}`;
    gestureDetector.handleButton(remapped, pressed);
  });

  serialDevice.on('connected', () => {
    connected = true;
    portPath = config.device.port ?? null;
    emitStatus();
    serialDevice.sendStatus('Connected');
    pingInterval = setInterval(() => serialDevice.sendPing(), 5000);
  });

  serialDevice.on('disconnected', () => {
    connected = false;
    portPath = null;
    if (pingInterval) { clearInterval(pingInterval); pingInterval = null; }
    emitStatus();
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
    handedness = newConfig.handedness;
    gestureDetector.updateConfig({
      longPressMs: newConfig.gestures.longPressMs,
      doublePressMs: newConfig.gestures.doublePressMs,
    });
    notificationServer.updateConfig(newConfig);
  });

  // Start services
  configWatcher.start();
  notificationServer.start();
  serialDevice.connect();

  return {
    shutdown() {
      if (pingInterval) { clearInterval(pingInterval); pingInterval = null; }
      configWatcher.stop();
      notificationServer.stop();
      serialDevice.disconnect();
      gestureDetector.reset();
    },
    getStatus(): BridgeStatus {
      return { connected, portPath, pendingCount: notificationServer.hasPending() ? 1 : 0 };
    },
    onStatusChange(cb: (status: BridgeStatus) => void) {
      statusListeners.push(cb);
    },
    sendText(text: string): boolean {
      return serialDevice.sendText(text);
    },
    sendStatus(text: string): boolean {
      return serialDevice.sendStatus(text);
    },
    clearDisplay(): boolean {
      return serialDevice.clearDisplay();
    },
    sendLeds(leds: Array<{ index: number; r: number; g: number; b: number }>): boolean {
      const remapped = leds.map(led => ({ ...led, index: remapButtonIndex(led.index, handedness) }));
      return serialDevice.sendLeds(remapped);
    },
    sendLabels(labels: string[]): boolean {
      const reordered = handedness === 'right' ? [...labels].reverse() : labels;
      return serialDevice.sendLabels(reordered);
    },
  };
}
