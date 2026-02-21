import * as readline from 'readline';
import { HarnessContext, commands } from './commands';
import { VirtualDevice } from './virtual';
import { WsClient } from './ws-client';

// Tokeniser: split on spaces, handle "quoted strings"
function tokenise(line: string): string[] {
  const tokens: string[] = [];
  let current = '';
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const ch = line[i];

    if (ch === '"') {
      inQuotes = !inQuotes;
      current += ch;
    } else if (ch === ' ' && !inQuotes) {
      if (current) {
        tokens.push(current);
        current = '';
      }
    } else {
      current += ch;
    }
  }

  if (current) {
    tokens.push(current);
  }

  return tokens;
}

// Async-safe printer: clear line, print, re-prompt
function asyncPrint(rl: readline.Interface, line: string): void {
  process.stdout.write('\r\x1b[K');
  console.log(line);
  rl.prompt(true);
}

// Build dynamic prompt based on context
function buildPrompt(ctx: HarnessContext): string {
  let serial = 'serial:none';
  if (ctx.device) {
    if (ctx.mode === 'virtual') {
      const vdev = ctx.device as VirtualDevice;
      serial = `serial:virtual ${vdev.slavePath}`;
    } else {
      serial = `serial:${ctx.portPath || 'connected'}`;
    }
  }

  const ws = ctx.ws?.isConnected ? `ws:${ctx.wsUrl || 'localhost:52914'}` : 'ws:disconnected';

  return `[${serial}] [${ws}] > `;
}

export async function startRepl(ctx: HarnessContext): Promise<void> {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: true,
  });

  // Wrap ctx.log to use asyncPrint
  ctx.log = (line) => asyncPrint(rl, line);

  // Tab completion
  rl.on('line', () => {
    /* handled below */
  });

  rl.setPrompt(buildPrompt(ctx));
  rl.prompt();

  // Wire up events from device and WS
  if (ctx.device) {
    ctx.device.on('frame', (frame) => {
      const msgName = getMessageName(frame.msgType);
      const payload = formatPayload(frame.msgType, frame.payload);
      asyncPrint(rl, `<-- [${msgName}] ${payload}`);
    });

    ctx.device.on('close', () => {
      asyncPrint(rl, '[DEVICE] connection closed');
      ctx.device = null;
      ctx.mode = 'none';
      rl.setPrompt(buildPrompt(ctx));
    });
  }

  if (ctx.ws) {
    ctx.ws.on('message', (msg) => {
      asyncPrint(rl, `[WS] <-- ${JSON.stringify(msg)}`);
    });
  }

  // Main REPL loop — serialize async handlers so rapid/pasted input doesn't
  // race (e.g. "send-text" running before "connect" has finished its await).
  let processing = Promise.resolve();

  rl.on('line', (input) => {
    processing = processing.then(async () => {
      const line = input.trim();
      if (!line) {
        rl.prompt();
        return;
      }

      const tokens = tokenise(line);
      const cmdName = tokens[0];
      const args = tokens.slice(1);

      const cmd = commands[cmdName];
      if (!cmd) {
        asyncPrint(rl, `unknown command: ${cmdName}`);
        rl.setPrompt(buildPrompt(ctx));
        rl.prompt();
        return;
      }

      try {
        const result = await cmd.handler(args, ctx);

        if (result.ok) {
          result.lines.forEach((line) => asyncPrint(rl, line));
        } else {
          asyncPrint(rl, `error: ${result.error}`);
        }
      } catch (err: any) {
        asyncPrint(rl, `error: ${err.message}`);
      }

      rl.setPrompt(buildPrompt(ctx));
      rl.prompt();
    });
  });

  rl.on('close', () => {
    process.exit(0);
  });
}

function getMessageName(msgType: number): string {
  const names: Record<number, string> = {
    0x01: 'DISPLAY_TEXT',
    0x02: 'BUTTON',
    0x03: 'SET_LEDS',
    0x04: 'STATUS',
    0x05: 'CLEAR',
    0x06: 'SET_LABELS',
    0x07: 'HEARTBEAT',
  };
  return names[msgType] || `MSG_${msgType.toString(16).toUpperCase()}`;
}

function formatPayload(msgType: number, payload: Buffer): string {
  switch (msgType) {
    case 0x01: // DISPLAY_TEXT
    case 0x04: // STATUS
      return `"${payload.toString('utf8')}"`;

    case 0x02: // BUTTON
      if (payload.length >= 2) {
        return `id=${payload[0]} pressed=${payload[1] === 1 ? 'true' : 'false'}`;
      }
      return payload.toString('hex');

    case 0x03: // SET_LEDS
      const leds: string[] = [];
      for (let i = 0; i < payload.length; i += 4) {
        leds.push(
          `idx=${payload[i]} r=${payload[i + 1]} g=${payload[i + 2]} b=${payload[i + 3]}`
        );
      }
      return leds.join(', ');

    case 0x07: // HEARTBEAT
      return `status=0x${payload[0]?.toString(16).toUpperCase().padStart(2, '0') || '?'}`;

    default:
      return payload.toString('hex');
  }
}
