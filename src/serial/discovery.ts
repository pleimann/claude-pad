import { readdirSync } from 'fs';
import { execSync } from 'child_process';

export interface PortInfo {
  path: string;
  vendorId?: string;
  productId?: string;
}

/** List serial port device files that look like USB serial devices. */
export async function listPorts(): Promise<PortInfo[]> {
  const devFiles = readdirSync('/dev').filter(f =>
    f.startsWith('cu.usbmodem') || f.startsWith('cu.usbserial') ||
    f.startsWith('ttyACM') || f.startsWith('ttyUSB')
  ).map(f => `/dev/${f}`);

  const usbInfo = process.platform === 'darwin' ? getAcmDeviceInfo() : [];

  return devFiles.map(path => {
    const info = usbInfo.find(u => u.path === path);
    return {
      path,
      vendorId: info?.vendorId,
      productId: info?.productId,
    };
  });
}

/**
 * Find a serial port by USB vendor and product ID.
 * Returns the device path, or undefined if not found.
 */
export async function findPort(vendorId: number, productId: number): Promise<string | undefined> {
  const ports = await listPorts();
  const vid = vendorId.toString(16).toLowerCase();
  const pid = productId.toString(16).toLowerCase();

  const match = ports.find(p =>
    p.vendorId === vid && p.productId === pid
  );

  return match?.path;
}

interface UsbInfo {
  path: string;
  vendorId?: string;
  productId?: string;
}

/**
 * Parse macOS ioreg for AppleUSBACMData entries to get
 * vendor/product IDs alongside tty device paths.
 */
function getAcmDeviceInfo(): UsbInfo[] {
  try {
    const output = execSync(
      'ioreg -r -c AppleUSBACMData -l 2>/dev/null',
      { encoding: 'utf8', timeout: 5000 }
    );

    const results: UsbInfo[] = [];
    // Split into per-ACM-device blocks. Each top-level AppleUSBACMData entry
    // contains idVendor/idProduct and a nested IOSerialBSDClient with the tty path.
    const blocks = output.split(/(?=\+-o AppleUSBACMData\b)/);

    for (const block of blocks) {
      const vidMatch = block.match(/"idVendor"\s*=\s*(\d+)/);
      const pidMatch = block.match(/"idProduct"\s*=\s*(\d+)/);
      const pathMatch = block.match(/"IOCalloutDevice"\s*=\s*"([^"]+)"/);

      if (pathMatch) {
        results.push({
          path: pathMatch[1],
          vendorId: vidMatch ? parseInt(vidMatch[1]).toString(16).toLowerCase() : undefined,
          productId: pidMatch ? parseInt(pidMatch[1]).toString(16).toLowerCase() : undefined,
        });
      }
    }

    return results;
  } catch {
    return [];
  }
}
