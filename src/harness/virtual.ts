import { EventEmitter } from 'events';
import { readSync, writeSync, closeSync } from 'fs';
import { dlopen, FFIType, ptr } from 'bun:ffi';
import { FrameParser, buildFrame } from '../serial/protocol';
import { MSG_BUTTON, MSG_HEARTBEAT } from '../types';

const POLL_INTERVAL = 10; // ms
const READ_BUFFER_SIZE = 1024;

export class VirtualDevice extends EventEmitter {
  private masterFd: number | null = null;
  public slavePath: string = '';
  public mode: 'virtual' = 'virtual';
  private parser: FrameParser | null = null;
  private pollTimer: NodeJS.Timeout | null = null;

  async connect(): Promise<{ slavePath: string }> {
    const { masterFd, slavePath } = await this.createPty();
    this.masterFd = masterFd;
    this.slavePath = slavePath;

    // Set non-blocking
    this.setNonBlocking(masterFd);

    // Create parser
    this.parser = new FrameParser();
    this.parser.on('frame', (frame) => {
      this.emit('frame', frame);
    });

    // Start polling
    this.startPolling();

    return { slavePath };
  }

  disconnect(): void {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
    if (this.masterFd !== null) {
      try {
        closeSync(this.masterFd);
      } catch (err) {
        // ignore
      }
      this.masterFd = null;
    }
    this.parser = null;
  }

  sendText(text: string): boolean {
    if (this.masterFd === null) return false;
    try {
      const frame = buildFrame(0x01, Buffer.from(text, 'utf8'));
      writeSync(this.masterFd, frame);
      return true;
    } catch (err) {
      return false;
    }
  }

  sendStatus(text: string): boolean {
    if (this.masterFd === null) return false;
    try {
      const frame = buildFrame(0x04, Buffer.from(text, 'utf8'));
      writeSync(this.masterFd, frame);
      return true;
    } catch (err) {
      return false;
    }
  }

  clear(): boolean {
    if (this.masterFd === null) return false;
    try {
      const frame = buildFrame(0x05);
      writeSync(this.masterFd, frame);
      return true;
    } catch (err) {
      return false;
    }
  }

  sendLeds(leds: Array<{ index: number; r: number; g: number; b: number }>): boolean {
    if (this.masterFd === null) return false;
    try {
      const payload = Buffer.alloc(leds.length * 4);
      let offset = 0;
      for (const led of leds) {
        payload[offset++] = led.index;
        payload[offset++] = led.r;
        payload[offset++] = led.g;
        payload[offset++] = led.b;
      }
      const frame = buildFrame(0x03, payload);
      writeSync(this.masterFd, frame);
      return true;
    } catch (err) {
      return false;
    }
  }

  sendLabels(labels: string[]): boolean {
    if (this.masterFd === null) return false;
    try {
      const parts: Buffer[] = [];
      for (const label of labels) {
        const utf8 = Buffer.from(label, 'utf8');
        const len = Buffer.alloc(1);
        len[0] = utf8.length;
        parts.push(len, utf8);
      }
      const payload = Buffer.concat(parts);
      const frame = buildFrame(0x06, payload);
      writeSync(this.masterFd, frame);
      return true;
    } catch (err) {
      return false;
    }
  }

  async sendButtonEvent(buttonId: number, holdMs: number = 50): Promise<void> {
    if (this.masterFd === null) throw new Error('not connected');

    // Press
    const pressFrame = buildFrame(MSG_BUTTON, Buffer.from([buttonId, 1]));
    writeSync(this.masterFd, pressFrame);
    this.emit('frame', { msgType: MSG_BUTTON, payload: Buffer.from([buttonId, 1]) });

    // Wait
    await new Promise((resolve) => setTimeout(resolve, holdMs));

    // Release
    const releaseFrame = buildFrame(MSG_BUTTON, Buffer.from([buttonId, 0]));
    writeSync(this.masterFd, releaseFrame);
    this.emit('frame', { msgType: MSG_BUTTON, payload: Buffer.from([buttonId, 0]) });
  }

  sendHeartbeat(status: number): void {
    if (this.masterFd === null) return;
    try {
      const frame = buildFrame(MSG_HEARTBEAT, Buffer.from([status]));
      writeSync(this.masterFd, frame);
    } catch (err) {
      // ignore
    }
  }

  // Private methods

  private createPty(): Promise<{ masterFd: number; slavePath: string }> {
    // Try libutil first (macOS ≤12), then libSystem.B (macOS ≥13)
    let lib;
    try {
      lib = dlopen('libutil.dylib', {
        openpty: {
          args: [FFIType.ptr, FFIType.ptr, FFIType.ptr, FFIType.ptr, FFIType.ptr],
          returns: FFIType.int,
        },
      });
    } catch (err) {
      lib = dlopen('libSystem.B.dylib', {
        openpty: {
          args: [FFIType.ptr, FFIType.ptr, FFIType.ptr, FFIType.ptr, FFIType.ptr],
          returns: FFIType.int,
        },
      });
    }

    const masterBuf = new Int32Array(1);
    const slaveBuf = new Int32Array(1);
    const nameBuf = new Uint8Array(256); // PATH_MAX

    const rc = lib.symbols.openpty(ptr(masterBuf), ptr(slaveBuf), ptr(nameBuf), 0, 0);

    if (rc !== 0) {
      throw new Error('openpty() failed');
    }

    const masterFd = masterBuf[0];
    const slavePath = Buffer.from(nameBuf).toString('utf8').replace(/\0.*/, '');

    return Promise.resolve({ masterFd, slavePath });
  }

  private setNonBlocking(fd: number): void {
    // fcntl(fd, F_SETFL, O_NONBLOCK)
    const libc = dlopen('libSystem.B.dylib', {
      fcntl: {
        args: [FFIType.int, FFIType.int, FFIType.int],
        returns: FFIType.int,
      },
    });

    const F_SETFL = 4;
    const O_NONBLOCK = 0x0004;

    libc.symbols.fcntl(fd, F_SETFL, O_NONBLOCK);
  }

  private startPolling(): void {
    const buf = Buffer.alloc(READ_BUFFER_SIZE);

    this.pollTimer = setInterval(() => {
      if (this.masterFd === null || !this.parser) return;

      try {
        const n = readSync(this.masterFd, buf, 0, buf.length, null);
        if (n > 0) {
          this.parser.parse(buf.subarray(0, n));
        }
      } catch (err: any) {
        if (err.code === 'EAGAIN' || err.code === 'EWOULDBLOCK') {
          // Normal: no data available
        } else if (err.code === 'EIO') {
          // Slave side closed
          this.emit('close');
          this.disconnect();
        } else {
          console.error('[virtual] read error:', err.message);
        }
      }
    }, POLL_INTERVAL);
  }
}
