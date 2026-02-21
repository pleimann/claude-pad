import { EventEmitter } from 'events';
import {
  MSG_DISPLAY_TEXT,
  MSG_STATUS,
  MSG_CLEAR,
  MSG_SET_LEDS,
  MSG_SET_LABELS,
  ResponseMessage,
  ErrorMessage,
} from '../types';
import { SerialDevice } from '../serial/device';

// Forward declarations
export interface VirtualDevice extends EventEmitter {
  mode: 'virtual';
  slavePath: string;
  sendText(text: string): boolean;
  sendStatus(text: string): boolean;
  clear(): boolean;
  sendLeds(leds: Array<{ index: number; r: number; g: number; b: number }>): boolean;
  sendLabels(labels: string[]): boolean;
  disconnect(): void;
  sendButtonEvent(buttonId: number, holdMs?: number): Promise<void>;
  sendHeartbeat(status: number): void;
}

export type Device = (SerialDevice & { mode: 'real' }) | VirtualDevice;

export interface WsClient extends EventEmitter {
  isConnected: boolean;
  connect(url: string): Promise<void>;
  disconnect(): void;
  sendNotification(text: string, category?: string): Promise<ResponseMessage | ErrorMessage>;
}

export interface HarnessContext {
  device: Device | null;
  ws: WsClient | null;
  mode: 'real' | 'virtual' | 'none';
  log: (line: string) => void;
  wsUrl?: string;
  portPath?: string;
  createDevice?: (portOrMode: string) => Promise<Device>;
}

export type HarnessResult = { ok: true; lines: string[] } | { ok: false; error: string };
export type CommandHandler = (args: string[], ctx: HarnessContext) => Promise<HarnessResult>;

interface CommandEntry {
  handler: CommandHandler;
  usage: string;
  description: string;
}

// Command registry
export const commands: Record<string, CommandEntry> = {
  connect: {
    usage: 'connect [port|virtual|auto]',
    description: 'Connect to a serial port (auto-detects if omitted)',
    handler: async (args, ctx) => {
      if (ctx.device) return { ok: false, error: 'already connected — run "disconnect" first' };
      if (!ctx.createDevice) return { ok: false, error: 'createDevice not available' };

      const portOrMode = args[0] ?? 'auto';
      try {
        const device = await ctx.createDevice(portOrMode);
        ctx.device = device;
        ctx.mode = device.mode ?? 'real';

        if (ctx.mode === 'virtual') {
          const vdev = device as VirtualDevice;
          return { ok: true, lines: [`virtual device ready at ${vdev.slavePath}`] };
        } else {
          const resolvedPath = (device as any).portPath ?? portOrMode;
          ctx.portPath = resolvedPath;
          return { ok: true, lines: [`connected to ${resolvedPath}`] };
        }
      } catch (err: any) {
        return { ok: false, error: `connection failed: ${err.message}` };
      }
    },
  },
  disconnect: {
    usage: 'disconnect',
    description: 'Disconnect from serial device',
    handler: async (args, ctx) => {
      if (!ctx.device) return { ok: false, error: 'no serial connection' };
      ctx.device.disconnect();
      ctx.device = null;
      ctx.mode = 'none';
      return { ok: true, lines: ['disconnected'] };
    },
  },
  'send-text': {
    usage: 'send-text <text>',
    description: 'Send display text (MSG_DISPLAY_TEXT)',
    handler: async (args, ctx) => {
      if (!ctx.device) return { ok: false, error: 'no serial connection — run "connect <port|virtual>" first' };
      if (args.length < 1) return { ok: false, error: 'usage: send-text <text>' };
      const text = args.join(' ').replace(/^"|"$/g, '');
      const ok = ctx.device.sendText(text);
      return ok ? { ok: true, lines: [`sent: ${text}`] } : { ok: false, error: 'send failed' };
    },
  },
  'send-status': {
    usage: 'send-status <text>',
    description: 'Send status text (MSG_STATUS)',
    handler: async (args, ctx) => {
      if (!ctx.device) return { ok: false, error: 'no serial connection' };
      if (args.length < 1) return { ok: false, error: 'usage: send-status <text>' };
      const text = args.join(' ').replace(/^"|"$/g, '');
      const ok = ctx.device.sendStatus(text);
      return ok ? { ok: true, lines: [`status: ${text}`] } : { ok: false, error: 'send failed' };
    },
  },
  clear: {
    usage: 'clear',
    description: 'Clear display (MSG_CLEAR)',
    handler: async (args, ctx) => {
      if (!ctx.device) return { ok: false, error: 'no serial connection' };
      const ok = ctx.device.clear();
      return ok ? { ok: true, lines: ['display cleared'] } : { ok: false, error: 'send failed' };
    },
  },
  'set-leds': {
    usage: 'set-leds <idx> <r> <g> <b> [...]',
    description: 'Set NeoPixel LED colors',
    handler: async (args, ctx) => {
      if (!ctx.device) return { ok: false, error: 'no serial connection' };
      if (args.length < 4) return { ok: false, error: 'usage: set-leds <idx> <r> <g> <b>' };

      const leds: Array<{ index: number; r: number; g: number; b: number }> = [];
      for (let i = 0; i < args.length; i += 4) {
        const idx = parseInt(args[i]);
        const r = parseInt(args[i + 1]);
        const g = parseInt(args[i + 2]);
        const b = parseInt(args[i + 3]);
        if (isNaN(idx) || isNaN(r) || isNaN(g) || isNaN(b)) {
          return { ok: false, error: 'idx, r, g, b must be integers' };
        }
        leds.push({ index: idx, r, g, b });
      }

      const ok = ctx.device.sendLeds(leds);
      return ok ? { ok: true, lines: [`set ${leds.length} LED(s)`] } : { ok: false, error: 'send failed' };
    },
  },
  'set-labels': {
    usage: 'set-labels <label1> <label2> <label3> <label4>',
    description: 'Set button labels',
    handler: async (args, ctx) => {
      if (!ctx.device) return { ok: false, error: 'no serial connection' };
      if (args.length < 4) return { ok: false, error: 'usage: set-labels <l1> <l2> <l3> <l4>' };

      const labels = args.slice(0, 4).map((l) => l.replace(/^"|"$/g, ''));
      const ok = ctx.device.sendLabels(labels);
      return ok ? { ok: true, lines: [`set labels: ${labels.join(', ')}`] } : { ok: false, error: 'send failed' };
    },
  },
  'press-button': {
    usage: 'press-button <id> [duration_ms]',
    description: 'Simulate button press (virtual mode only)',
    handler: async (args, ctx) => {
      if (!ctx.device || ctx.mode !== 'virtual') {
        return { ok: false, error: 'press-button only works in virtual mode' };
      }
      if (args.length < 1) return { ok: false, error: 'usage: press-button <id> [duration_ms]' };

      const id = parseInt(args[0]);
      const duration = args[1] ? parseInt(args[1]) : 50;
      if (isNaN(id) || isNaN(duration)) return { ok: false, error: 'id and duration must be integers' };

      const vdev = ctx.device as VirtualDevice;
      await vdev.sendButtonEvent(id, duration);
      return { ok: true, lines: [`simulated button ${id} press (${duration}ms)`] };
    },
  },
  'ws-connect': {
    usage: 'ws-connect [url]',
    description: 'Connect to WebSocket server',
    handler: async (args, ctx) => {
      if (ctx.ws?.isConnected) return { ok: false, error: 'already connected to WebSocket' };

      const url = args[0] || 'ws://localhost:52914';
      try {
        if (!ctx.ws) {
          return { ok: false, error: 'ws client not initialized' };
        }
        await ctx.ws.connect(url);
        ctx.wsUrl = url;
        return { ok: true, lines: [`connected to ${url}`] };
      } catch (err: any) {
        return { ok: false, error: `connection failed: ${err.message}` };
      }
    },
  },
  'ws-disconnect': {
    usage: 'ws-disconnect',
    description: 'Disconnect from WebSocket',
    handler: async (args, ctx) => {
      if (!ctx.ws?.isConnected) return { ok: false, error: 'not connected to WebSocket' };
      ctx.ws.disconnect();
      return { ok: true, lines: ['disconnected from WebSocket'] };
    },
  },
  notify: {
    usage: 'notify <text> [category]',
    description: 'Send a notification and wait for response',
    handler: async (args, ctx) => {
      if (!ctx.ws?.isConnected) return { ok: false, error: 'not connected to WebSocket' };
      if (args.length < 1) return { ok: false, error: 'usage: notify <text> [category]' };

      const text = args[0].replace(/^"|"$/g, '');
      const category = args[1]?.replace(/^"|"$/g, '');

      try {
        const response = await ctx.ws.sendNotification(text, category);
        if ('action' in response) {
          return {
            ok: true,
            lines: [`response: action="${response.action}" label="${response.label}"`],
          };
        } else {
          return { ok: false, error: response.error };
        }
      } catch (err: any) {
        return { ok: false, error: `notify failed: ${err.message}` };
      }
    },
  },
  status: {
    usage: 'status',
    description: 'Show connection status',
    handler: async (args, ctx) => {
      const lines: string[] = [];
      if (ctx.device) {
        lines.push(`serial: ${ctx.mode} ${ctx.portPath || ctx.mode === 'virtual' ? (ctx.device as VirtualDevice).slavePath : 'unknown'}`);
      } else {
        lines.push('serial: disconnected');
      }
      if (ctx.ws?.isConnected) {
        lines.push(`websocket: connected to ${ctx.wsUrl || '?'}`);
      } else {
        lines.push('websocket: disconnected');
      }
      return { ok: true, lines };
    },
  },
  help: {
    usage: 'help [command]',
    description: 'Show help',
    handler: async (args, ctx) => {
      const lines: string[] = [];
      if (args.length > 0) {
        const cmd = commands[args[0]];
        if (cmd) {
          lines.push(`${args[0]}: ${cmd.description}`);
          lines.push(`usage: ${cmd.usage}`);
        } else {
          lines.push(`unknown command: ${args[0]}`);
        }
      } else {
        lines.push('Available commands:');
        Object.entries(commands).forEach(([name, cmd]) => {
          lines.push(`  ${name.padEnd(15)} ${cmd.description}`);
        });
      }
      return { ok: true, lines };
    },
  },
  exit: {
    usage: 'exit',
    description: 'Exit harness',
    handler: async (args, ctx) => {
      process.exit(0);
    },
  },
  quit: {
    usage: 'quit',
    description: 'Exit harness',
    handler: async (args, ctx) => {
      process.exit(0);
    },
  },
};
