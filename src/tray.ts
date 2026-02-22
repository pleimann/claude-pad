#!/usr/bin/env bun
// camel-pad menu bar app entry point

import { readFileSync } from 'fs';
import { loadConfig, validateConfig } from '@/config/loader.js';
import { startBridge } from '@/bridge.js';
import { getTrayConfigPath } from '@/tray/config-store.js';
import { spawnSysTray, type SysTrayHandle } from '@/tray/systray-spawn.js';
import { startSettingsServer } from '@/tray/settings-server.js';
import type { BridgeHandle } from '@/bridge.js';

// Embed icon as a Bun asset
import iconPath from '@/static/tray-icon.png' with { type: 'file' };

const configPath = getTrayConfigPath();

// Item ID for the connection status line (used with updateItem for tooltip updates)
const ITEM_STATUS = 1;

let bridge: BridgeHandle | null = null;
let tray: SysTrayHandle | null = null;
let settingsHandle: { port: number; stop(): void } | null = null;

async function tryStartBridge() {
  const config = loadConfig(configPath);
  const errors = validateConfig(config);
  if (errors.length > 0) {
    console.log('Config invalid, bridge not started:', errors);
    return null;
  }
  try {
    const handle = await startBridge(configPath);
    handle.onStatusChange((status) => {
      tray?.updateItem(ITEM_STATUS, {
        title: status.connected ? '● Connected' : '○ Disconnected',
      });
    });
    return handle;
  } catch (err: any) {
    console.error('Failed to start bridge:', err.message);
    return null;
  }
}

async function onTrayClick() {
  if (settingsHandle) return; // popover already open
  settingsHandle = await startSettingsServer(configPath, async () => {
    // Config saved — restart bridge with new config, then close popover
    bridge?.shutdown();
    bridge = null;
    bridge = await tryStartBridge();
    tray?.hidePopover();
    settingsHandle = null;
  });
  tray?.showPopover(`http://localhost:${settingsHandle.port}`);
}

function onQuitClick() {
  settingsHandle?.stop();
  bridge?.shutdown();
  tray?.kill();
  process.exit(0);
}

async function main() {
  // Load icon as base64
  const iconBase64 = readFileSync(iconPath).toString('base64');

  // Start bridge if config is valid
  bridge = await tryStartBridge();

  const initialStatus = bridge
    ? (bridge.getStatus().connected ? '● Connected' : '○ Disconnected')
    : '○ Not configured';

  tray = await spawnSysTray(
    {
      icon: iconBase64,
      title: '',
      tooltip: 'camel-pad',
      items: [
        { title: initialStatus, enabled: false },
      ],
    },
    {
      onTrayClick,
      onQuitClick,
      onPopoverClosed() {
        settingsHandle?.stop();
        settingsHandle = null;
      },
    },
  );

  tray.onExit(() => process.exit(0));

  // If no valid config on first run, open settings automatically
  if (!bridge) {
    setTimeout(() => onTrayClick(), 500);
  }

  console.log('camel-pad tray running. Config:', configPath);
}

process.on('SIGINT', onQuitClick);
process.on('SIGTERM', onQuitClick);

main().catch((err) => {
  console.error('Fatal error:', err);
  process.exit(1);
});
