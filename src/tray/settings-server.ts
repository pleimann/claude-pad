import { readFileSync, writeFileSync } from 'fs';
import { stringify } from 'yaml';
import { loadConfig } from '@/config/loader.js';
import { listPorts } from '@/serial/discovery.js';
import type { Config } from '@/types.js';
import type { BridgeHandle } from '@/bridge.js';

// Embed the settings HTML as a Bun asset (works in dev mode and after bun --compile)
import settingsHtmlPath from '@/static/settings.html' with { type: 'file' };

export interface SettingsServerHandle {
  port: number;
  stop(): void;
}

/**
 * Starts a temporary HTTP server for the settings UI on a random port.
 * Returns the port so the caller can open it in a popover or browser.
 * The server shuts down automatically when the user saves or after a timeout.
 */
export async function startSettingsServer(
  configPath: string,
  bridge: BridgeHandle | null,
  onSaved?: () => void,
): Promise<SettingsServerHandle> {
  const settingsHtml = readFileSync(settingsHtmlPath, 'utf8');

  let idleTimer: ReturnType<typeof setTimeout> | null = null;
  let server: ReturnType<typeof Bun.serve> | null = null;

  function resetIdleTimer() {
    if (idleTimer) clearTimeout(idleTimer);
    idleTimer = setTimeout(() => server?.stop(), 5 * 60 * 1000);
  }

  const corsHeaders = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type',
  };

  server = Bun.serve({
    port: 0, // OS assigns a free port
    async fetch(req) {
      resetIdleTimer();
      const url = new URL(req.url);

      // Handle CORS preflight
      if (req.method === 'OPTIONS') {
        return new Response(null, { status: 204, headers: corsHeaders });
      }

      if (url.pathname === '/') {
        return new Response(settingsHtml, {
          headers: { 'Content-Type': 'text/html; charset=utf-8' },
        });
      }

      if (url.pathname === '/api/config') {
        if (req.method === 'GET') {
          const config = loadConfig(configPath);
          return Response.json(config);
        }

        if (req.method === 'POST') {
          try {
            const body = await req.json() as Partial<Config>;
            const yaml = buildYaml(body);
            writeFileSync(configPath, yaml, 'utf8');
            onSaved?.();
            return new Response('ok');
          } catch (err: any) {
            return new Response(err.message, { status: 400 });
          }
        }
      }

      if (url.pathname === '/api/devices') {
        try {
          const ports = await listPorts();
          return Response.json(ports);
        } catch (err: any) {
          return new Response(err.message, { status: 500 });
        }
      }

      if (url.pathname === '/api/status') {
        const status = bridge ? bridge.getStatus() : { connected: false, portPath: null, pendingCount: 0 };
        return Response.json(status, { headers: corsHeaders });
      }

      if (url.pathname === '/api/device/text' && req.method === 'POST') {
        if (!bridge) return Response.json({ ok: false, error: 'bridge not running' }, { headers: corsHeaders });
        const { text } = await req.json() as { text: string };
        const ok = bridge.sendText(text);
        return Response.json({ ok }, { headers: corsHeaders });
      }

      if (url.pathname === '/api/device/status-text' && req.method === 'POST') {
        if (!bridge) return Response.json({ ok: false, error: 'bridge not running' }, { headers: corsHeaders });
        const { text } = await req.json() as { text: string };
        const ok = bridge.sendStatus(text);
        return Response.json({ ok }, { headers: corsHeaders });
      }

      if (url.pathname === '/api/device/clear' && req.method === 'POST') {
        if (!bridge) return Response.json({ ok: false, error: 'bridge not running' }, { headers: corsHeaders });
        const ok = bridge.clearDisplay();
        return Response.json({ ok }, { headers: corsHeaders });
      }

      if (url.pathname === '/api/device/leds' && req.method === 'POST') {
        if (!bridge) return Response.json({ ok: false, error: 'bridge not running' }, { headers: corsHeaders });
        const { leds } = await req.json() as { leds: Array<{ index: number; r: number; g: number; b: number }> };
        const ok = bridge.sendLeds(leds);
        return Response.json({ ok }, { headers: corsHeaders });
      }

      if (url.pathname === '/api/device/labels' && req.method === 'POST') {
        if (!bridge) return Response.json({ ok: false, error: 'bridge not running' }, { headers: corsHeaders });
        const { labels } = await req.json() as { labels: string[] };
        const ok = bridge.sendLabels(labels);
        return Response.json({ ok }, { headers: corsHeaders });
      }

      if (url.pathname === '/api/close') {
        setTimeout(() => server?.stop(), 200);
        return new Response('ok');
      }

      return new Response('Not found', { status: 404 });
    },
  });

  resetIdleTimer();

  if (!server || !server.port) {
    throw new Error('Failed to start settings server');
  }

  const url = `http://localhost:${server.port}`;
  console.log('Settings server:', url);

  return {
    port: server.port,
    stop() {
      if (idleTimer) clearTimeout(idleTimer);
      server?.stop();
    },
  };
}

/**
 * Converts a partial Config object (from the settings form) back to YAML.
 * Only writes fields that are set; preserves the simple format.
 */
function buildYaml(config: Partial<Config>): string {
  const out: Record<string, any> = {};

  if (config.device) {
    out.device = {};
    if (config.device.port) out.device.port = config.device.port;
    if (config.device.vendorId) out.device.vendorId = `0x${config.device.vendorId.toString(16)}`;
    if (config.device.productId) out.device.productId = `0x${config.device.productId.toString(16)}`;
  }

  if (config.server) {
    out.server = {
      port: config.server.port,
      host: config.server.host,
    };
  }

  if (config.gestures) {
    out.gestures = {
      longPressMs: config.gestures.longPressMs,
      doublePressMs: config.gestures.doublePressMs,
    };
  }

  if (config.defaults) {
    out.defaults = { timeoutMs: config.defaults.timeoutMs };
  }

  if (config.keys && Object.keys(config.keys).length > 0) {
    out.keys = config.keys;
  }

  return stringify(out);
}
