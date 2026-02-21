import { EventEmitter } from 'events';
import WebSocket from 'ws';
import { randomUUID } from 'crypto';
import { NotificationMessage, ResponseMessage, ErrorMessage } from '../types';

export class WsClient extends EventEmitter {
  private ws: WebSocket | null = null;
  private pending: Map<
    string,
    {
      resolve: (msg: ResponseMessage | ErrorMessage) => void;
      reject: (err: Error) => void;
      timer: NodeJS.Timeout;
    }
  > = new Map();

  private defaultTimeoutMs = 30000;

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  async connect(url: string): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(url);

        this.ws.on('open', () => {
          this.ws!.on('message', (data: any) => this.handleMessage(data));
          resolve();
        });

        this.ws.on('error', (err) => {
          reject(err);
        });

        this.ws.on('close', () => {
          // Reject any pending notifications
          for (const [id, { reject }] of this.pending) {
            reject(new Error('WebSocket closed'));
            this.pending.delete(id);
          }
        });
      } catch (err) {
        reject(err);
      }
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    // Reject any pending
    for (const [id, { reject }] of this.pending) {
      reject(new Error('Disconnected'));
    }
    this.pending.clear();
  }

  async sendNotification(text: string, category?: string): Promise<ResponseMessage | ErrorMessage> {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket not connected');
    }

    const id = randomUUID();
    const msg: NotificationMessage = {
      type: 'notification',
      id,
      text,
      category,
    };

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`Timeout after ${this.defaultTimeoutMs}ms`));
      }, this.defaultTimeoutMs);

      this.pending.set(id, { resolve, reject, timer });

      try {
        this.ws!.send(JSON.stringify(msg));
      } catch (err) {
        this.pending.delete(id);
        clearTimeout(timer);
        reject(err);
      }
    });
  }

  private handleMessage(data: any): void {
    try {
      const msg = typeof data === 'string' ? JSON.parse(data) : data;

      if (msg.type === 'response' || msg.type === 'error') {
        const pending = this.pending.get(msg.id);
        if (pending) {
          this.pending.delete(msg.id);
          clearTimeout(pending.timer);
          pending.resolve(msg);
        } else {
          // Unsolicited message - emit for REPL to display
          this.emit('message', msg);
        }
      } else {
        // Other message types
        this.emit('message', msg);
      }
    } catch (err) {
      console.error('[WS] parse error:', err);
    }
  }
}
