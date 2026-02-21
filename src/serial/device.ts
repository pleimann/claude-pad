import { openSync, readSync, writeSync, closeSync, constants } from 'fs';
import { execSync } from 'child_process';
import { EventEmitter } from 'events';
import {
  MSG_BUTTON, MSG_SET_LEDS,
  MSG_DISPLAY_TEXT, MSG_STATUS, MSG_CLEAR, MSG_SET_LABELS, MSG_HEARTBEAT,
  SERIAL_BAUD,
} from '../types.js';
import { buildFrame, FrameParser } from './protocol.js';
import type { ParsedFrame } from './protocol.js';
import { findPort } from './discovery.js';

export interface SerialDeviceConfig {
  port?: string;
  vendorId?: number;
  productId?: number;
}

export interface ButtonEvent {
  buttonId: string;
  pressed: boolean;
}

export class SerialDevice extends EventEmitter {
  private config: SerialDeviceConfig;
  private fd: number | null = null;      // Non-blocking, for both reads and writes
  private portPath: string | null = null;
  private parser = new FrameParser();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pollTimer: ReturnType<typeof setInterval> | null = null;
  private readonly RECONNECT_INTERVAL = 2000;
  private readonly POLL_INTERVAL = 10; // 100Hz polling
  private readonly WRITE_RETRY_MAX = 50;
  private readonly WRITE_RETRY_DELAY_MS = 5;

  constructor(config: SerialDeviceConfig) {
    super();
    this.config = config;
    this.parser.on('frame', (frame: ParsedFrame) => this.handleFrame(frame));
  }

  async connect(): Promise<boolean> {
    let portPath = this.config.port;

    if (!portPath && this.config.vendorId && this.config.productId) {
      portPath = await findPort(this.config.vendorId, this.config.productId);
      if (!portPath) {
        console.error(
          `Device not found: vendor=0x${this.config.vendorId.toString(16)} product=0x${this.config.productId.toString(16)}`
        );
        this.scheduleReconnect();
        return false;
      }
    }

    if (!portPath) {
      console.error('No serial port configured and no vendor/product IDs for auto-discovery');
      return false;
    }

    try {
      // Configure serial port with stty
      this.configurePort(portPath);

      // Open non-blocking fd for both reads and writes
      this.fd = openSync(portPath, constants.O_RDWR | constants.O_NOCTTY | constants.O_NONBLOCK);
      this.portPath = portPath;

      // Start polling for incoming data
      this.startPolling();

      console.log(`Connected to serial port ${portPath}`);
      this.emit('connected');
      return true;
    } catch (err) {
      console.error('Failed to open serial port:', err);
      this.scheduleReconnect();
      return false;
    }
  }

  private configurePort(portPath: string): void {
    try {
      if (process.platform === 'darwin') {
        execSync(`stty -f "${portPath}" ${SERIAL_BAUD} raw clocal -echo -echoctl -echoke -icanon -isig -iexten -opost cs8 -cstopb -parenb`, { timeout: 5000 });
      } else {
        execSync(`stty -F "${portPath}" ${SERIAL_BAUD} raw clocal -echo cs8 -cstopb -parenb`, { timeout: 5000 });
      }
    } catch (err: any) {
      // stty can hang on some USB CDC ports; log but don't fail
      console.warn(`stty configuration warning: ${err.message}`);
    }
  }

  private startPolling(): void {
    if (this.pollTimer) return;

    const buf = Buffer.alloc(1024);
    this.pollTimer = setInterval(() => {
      if (this.fd === null) return;

      try {
        const bytesRead = readSync(this.fd, buf, 0, buf.length, null);
        if (bytesRead > 0) {
          this.parser.parse(buf.subarray(0, bytesRead));
        }
      } catch (err: any) {
        // EAGAIN/EWOULDBLOCK is normal for non-blocking reads with no data
        if (err.code === 'EAGAIN' || err.code === 'EWOULDBLOCK') return;
        // EIO or other errors mean the device was disconnected
        this.handleError(err);
      }
    }, this.POLL_INTERVAL);
  }

  private stopPolling(): void {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      console.log('Attempting to reconnect...');
      this.connect();
    }, this.RECONNECT_INTERVAL);
  }

  private sleepMs(ms: number): void {
    const start = Date.now();
    while (Date.now() - start < ms) {
      // Busy-wait for minimal latency
    }
  }

  private handleFrame(frame: ParsedFrame): void {
    switch (frame.msgType) {
      case MSG_BUTTON: {
        if (frame.payload.length >= 2) {
          const buttonId = `key${frame.payload[0]}`;
          const pressed = frame.payload[1] === 1;
          this.emit('button', { buttonId, pressed } as ButtonEvent);
        }
        break;
      }
      case MSG_HEARTBEAT:
        this.emit('heartbeat', frame.payload[0]);
        break;
    }
  }

  private handleError(err: Error): void {
    console.error('Serial error:', err.message);
    this.emit('error', err);
    this.disconnect();
    this.scheduleReconnect();
  }

  sendText(text: string): boolean {
    return this.sendMessage(MSG_DISPLAY_TEXT, Buffer.from(text, 'utf8'));
  }

  sendStatus(text: string): boolean {
    return this.sendMessage(MSG_STATUS, Buffer.from(text, 'utf8'));
  }

  sendLabels(labels: string[]): boolean {
    const parts: Buffer[] = [];
    for (const label of labels) {
      const encoded = Buffer.from(label, 'utf8');
      const lenBuf = Buffer.from([encoded.length]);
      parts.push(lenBuf, encoded);
    }
    return this.sendMessage(MSG_SET_LABELS, Buffer.concat(parts));
  }

  sendLeds(leds: Array<{ index: number; r: number; g: number; b: number }>): boolean {
    const buf = Buffer.alloc(leds.length * 4);
    for (let i = 0; i < leds.length; i++) {
      const led = leds[i];
      buf[i * 4] = led.index;
      buf[i * 4 + 1] = led.r;
      buf[i * 4 + 2] = led.g;
      buf[i * 4 + 3] = led.b;
    }
    return this.sendMessage(MSG_SET_LEDS, buf);
  }

  clearDisplay(): boolean {
    return this.sendMessage(MSG_CLEAR);
  }

  private sendMessage(msgType: number, payload?: Buffer): boolean {
    if (this.fd === null) {
      console.error('Failed to send: fd is null');
      return false;
    }

    try {
      const frame = buildFrame(msgType, payload);
      let written = 0;
      let retries = 0;

      // Handle non-blocking writes with retry on EAGAIN
      while (written < frame.length && retries < this.WRITE_RETRY_MAX) {
        try {
          const bytesWritten = writeSync(this.fd, frame, written);
          if (bytesWritten === 0 && written < frame.length) {
            // No bytes written but no error; delay and retry
            retries++;
            this.sleepMs(this.WRITE_RETRY_DELAY_MS);
          } else {
            written += bytesWritten;
            retries = 0; // Reset retries on successful write
          }
        } catch (err: any) {
          if (err.code === 'EAGAIN' || err.code === 'EWOULDBLOCK') {
            retries++;
            if (retries >= this.WRITE_RETRY_MAX) {
              console.error(`Write timeout after ${retries} retries`);
              return false;
            }
            // Delay before retry to allow kernel buffer to drain
            this.sleepMs(this.WRITE_RETRY_DELAY_MS);
          } else {
            console.error(`Write error: ${err.code || err.message}`, err);
            throw err;
          }
        }
      }

      if (written !== frame.length) {
        console.error(`Partial write: ${written}/${frame.length} bytes`);
        return false;
      }

      return true;
    } catch (err: any) {
      console.error('Failed to send:', err.message || err);
      return false;
    }
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    this.stopPolling();

    if (this.fd !== null) {
      try { closeSync(this.fd); } catch { /* ignore */ }
      this.fd = null;
    }
    if (this.portPath !== null) {
      this.portPath = null;
      this.parser.reset();
      this.emit('disconnected');
    }
  }

  isConnected(): boolean {
    return this.fd !== null;
  }
}
