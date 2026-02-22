import { spawn } from 'child_process';
import { createInterface } from 'readline';
import { readFileSync, writeFileSync, existsSync, mkdirSync, chmodSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';

// Asset-embed the Go tray binaries. At compile time (bun build --compile),
// these become embedded $bunfs/ paths. At dev time, they resolve to real files.
import trayBinDarwin from '@/static/traybin/tray_darwin_release' with { type: 'file' };
import trayBinWindows from '@/static/traybin/tray_windows_release.exe' with { type: 'file' };

export interface MenuItem {
  title: string;
  tooltip?: string;
  enabled: boolean;
  checked?: boolean;
  hidden?: boolean;
}

export interface SysTrayMenu {
  icon: string; // base64-encoded PNG
  title: string;
  tooltip: string;
  items: (MenuItem | '<SEPARATOR>')[];
}

export interface ClickAction {
  type: 'clicked';
  seq_id: number;
  item: MenuItem & { __id: number };
}

export interface SysTrayHandle {
  updateItem(id: number, updates: Partial<MenuItem>): void;
  showPopover(url: string, width?: number, height?: number): void;
  hidePopover(): void;
  kill(): void;
  onExit(cb: () => void): void;
}

function getTrayBinPath(): string {
  const cacheDir = join(homedir(), '.cache', 'camel-pad');
  mkdirSync(cacheDir, { recursive: true });

  const isWindows = process.platform === 'win32';
  const binName = isWindows ? 'tray.exe' : 'tray';
  const destPath = join(cacheDir, binName);

  const srcPath = isWindows ? trayBinWindows : trayBinDarwin;
  const srcBuf = readFileSync(srcPath);

  // Only write if missing or content changed
  let needsWrite = !existsSync(destPath);
  if (!needsWrite) {
    const existing = readFileSync(destPath);
    needsWrite = !existing.equals(srcBuf);
  }

  if (needsWrite) {
    writeFileSync(destPath, srcBuf);
    if (!isWindows) chmodSync(destPath, 0o755);
  }

  return destPath;
}

export interface SysTrayHandlers {
  onClick?: (action: ClickAction) => void;
  onTrayClick?: () => void;
  onQuitClick?: () => void;
  onPopoverClosed?: () => void;
}

export function spawnSysTray(
  menu: SysTrayMenu,
  handlers: SysTrayHandlers,
): Promise<SysTrayHandle> {
  return new Promise((resolve, reject) => {
    let binPath: string;
    try {
      binPath = getTrayBinPath();
    } catch (err) {
      reject(err);
      return;
    }

    const proc = spawn(binPath, [], { windowsHide: true });
    proc.on('error', reject);

    const rl = createInterface({ input: proc.stdout });

    // Assign sequential __id to every item including separators (1-indexed).
    // Separators and real items all consume an ID slot so that the position-based
    // constants in tray.ts (ITEM_STATUS=1, ITEM_SETTINGS=3, ITEM_QUIT=5) match exactly.
    const itemIds = new Map<number, MenuItem>();
    menu.items.forEach((item, i) => {
      const id = i + 1;
      if (item !== '<SEPARATOR>') {
        itemIds.set(id, item);
        (item as any).__id = id;
      }
    });

    function trimItem(item: MenuItem | '<SEPARATOR>', id: number) {
      if (item === '<SEPARATOR>') {
        return { title: '<SEPARATOR>', tooltip: '', enabled: true, __id: id };
      }
      return {
        title: item.title,
        tooltip: item.tooltip ?? '',
        enabled: item.enabled,
        checked: item.checked ?? false,
        hidden: item.hidden ?? false,
        __id: id,
      };
    }

    // Wait for the 'ready' signal, then send the menu config
    rl.on('line', (line) => {
      let action: any;
      try {
        action = JSON.parse(line);
      } catch {
        return;
      }

      if (action.type === 'ready') {
        const menuPayload = {
          icon: menu.icon,
          title: menu.title,
          tooltip: menu.tooltip,
          items: menu.items.map((item, i) => trimItem(item, i + 1)),
        };
        proc.stdin.write(JSON.stringify(menuPayload) + '\n');

        const handle: SysTrayHandle = {
          updateItem(id: number, updates: Partial<MenuItem>) {
            const existing = itemIds.get(id);
            if (!existing) return;
            Object.assign(existing, updates);
            proc.stdin.write(JSON.stringify({
              type: 'update-item',
              item: { ...trimItem(existing, id), ...updates, __id: id },
              seq_id: -1,
            }) + '\n');
          },
          showPopover(url: string, width = 520, height = 720) {
            proc.stdin.write(JSON.stringify({ type: 'show-popover', url, width, height }) + '\n');
          },
          hidePopover() {
            proc.stdin.write(JSON.stringify({ type: 'hide-popover' }) + '\n');
          },
          kill() {
            proc.stdin.write(JSON.stringify({ type: 'exit' }) + '\n');
          },
          onExit(cb: () => void) {
            proc.on('exit', cb);
          },
        };

        resolve(handle);
      } else if (action.type === 'clicked') {
        handlers.onClick?.(action as ClickAction);
      } else if (action.type === 'tray-clicked') {
        handlers.onTrayClick?.();
      } else if (action.type === 'quit-clicked') {
        handlers.onQuitClick?.();
      } else if (action.type === 'popover-closed') {
        handlers.onPopoverClosed?.();
      }
    });

    proc.on('exit', (code) => {
      if (code !== 0 && code !== null) {
        console.error(`Tray process exited with code ${code}`);
      }
    });
  });
}
