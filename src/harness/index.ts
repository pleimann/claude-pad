import { HarnessContext, commands } from './commands';
import { VirtualDevice } from './virtual';
import { WsClient } from './ws-client';
import { SerialDevice } from '../serial/device';
import { listPorts } from '../serial/discovery';
import { startRepl } from './repl';

async function main() {
  const [, , ...argv] = process.argv;

  // Parse CLI args
  let portPath: string | null = null;
  let isVirtual = false;
  let wsUrl: string | null = null;
  const cmdArgs: string[] = [];

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--port' && i + 1 < argv.length) {
      portPath = argv[++i];
    } else if (arg === '--virtual') {
      isVirtual = true;
    } else if (arg === '--ws' && i + 1 < argv.length) {
      wsUrl = argv[++i];
    } else if (arg.startsWith('--')) {
      // skip unknown flags
    } else {
      cmdArgs.push(arg);
    }
  }

  // Create device factory
  const createDevice = async (portOrMode: string) => {
    if (portOrMode === 'virtual') {
      const vdev = new VirtualDevice();
      await vdev.connect();
      return Object.assign(vdev, { mode: 'virtual' }) as any;
    } else if (portOrMode === 'auto') {
      const ports = await listPorts();
      if (ports.length === 0) throw new Error('no serial devices found — is the device connected?');
      const resolved = ports[0].path;
      const device = new SerialDevice({ port: resolved });
      await device.connect();
      return Object.assign(device, { mode: 'real', portPath: resolved }) as any;
    } else {
      const device = new SerialDevice({ port: portOrMode });
      await device.connect();
      return Object.assign(device, { mode: 'real', portPath: portOrMode }) as any;
    }
  };

  // Create context
  const ctx: HarnessContext = {
    device: null,
    ws: new WsClient(),
    mode: 'none',
    log: console.log,
    wsUrl,
    portPath,
    createDevice,
  };

  // Cleanup handler
  const cleanup = async () => {
    if (ctx.device) {
      ctx.device.disconnect();
    }
    if (ctx.ws?.isConnected) {
      ctx.ws.disconnect();
    }
    process.exit(0);
  };

  process.on('SIGINT', cleanup);
  process.on('SIGTERM', cleanup);

  try {
    // Single-shot mode: command provided as args
    if (cmdArgs.length > 0) {
      await runSingleShot(cmdArgs, ctx, isVirtual, portPath, wsUrl);
    } else {
      // REPL mode
      await startRepl(ctx);
    }
  } catch (err: any) {
    console.error('error:', err.message);
    await cleanup();
  }
}

async function runSingleShot(
  cmdArgs: string[],
  ctx: HarnessContext,
  isVirtual: boolean,
  portPath: string | null,
  wsUrl: string | null
): Promise<void> {
  const cmdName = cmdArgs[0];
  const args = cmdArgs.slice(1);

  // Auto-connect if needed
  if (['send-text', 'send-status', 'clear', 'set-leds', 'set-labels', 'press-button'].includes(cmdName)) {
    if (isVirtual) {
      const vdev = new VirtualDevice();
      const { slavePath } = await vdev.connect();
      ctx.device = vdev as any;
      ctx.mode = 'virtual';
      console.log(`[virtual] ${slavePath}`);
    } else if (portPath) {
      const device = new SerialDevice({ port: portPath });
      await device.connect();
      ctx.device = Object.assign(device, { mode: 'real', portPath }) as any;
      ctx.mode = 'real';
    } else {
      const ports = await listPorts();
      if (ports.length === 0) {
        console.error('error: no serial devices found (use --port <path> or --virtual)');
        process.exit(1);
      }
      const resolved = ports[0].path;
      const device = new SerialDevice({ port: resolved });
      await device.connect();
      ctx.device = Object.assign(device, { mode: 'real', portPath: resolved }) as any;
      ctx.mode = 'real';
      console.log(`[auto] ${resolved}`);
    }
  }

  // Auto-connect WS if needed
  if (['notify', 'ws-disconnect'].includes(cmdName)) {
    if (wsUrl || !cmdArgs.includes('--no-auto-ws')) {
      const url = wsUrl || 'ws://localhost:52914';
      try {
        await ctx.ws!.connect(url);
        ctx.wsUrl = url;
      } catch (err: any) {
        console.error(`error: could not connect to WebSocket: ${err.message}`);
        process.exit(1);
      }
    }
  }

  // Run command
  const cmd = commands[cmdName];
  if (!cmd) {
    console.error(`error: unknown command: ${cmdName}`);
    process.exit(1);
  }

  const result = await cmd.handler(args, ctx);
  if (result.ok) {
    result.lines.forEach((line) => console.log(line));
  } else {
    console.error(`error: ${result.error}`);
    process.exit(1);
  }

  // Clean up
  if (ctx.device) {
    ctx.device.disconnect();
  }
  if (ctx.ws?.isConnected) {
    ctx.ws.disconnect();
  }
}

main().catch((err) => {
  console.error('fatal error:', err);
  process.exit(1);
});
