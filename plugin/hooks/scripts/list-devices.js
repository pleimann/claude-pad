#!/usr/bin/env node

/**
 * List available HID devices for selection
 */

const HID = require('node-hid');

const devices = HID.devices();

// Filter to unique vendor/product combinations and format for display
const seen = new Set();
const uniqueDevices = [];

for (const device of devices) {
  const key = `${device.vendorId}:${device.productId}`;
  if (!seen.has(key) && device.vendorId && device.productId) {
    seen.add(key);
    uniqueDevices.push({
      vendorId: device.vendorId,
      productId: device.productId,
      vendor: `0x${device.vendorId.toString(16).padStart(4, '0')}`,
      product: `0x${device.productId.toString(16).padStart(4, '0')}`,
      name: device.product || 'Unknown Device',
      manufacturer: device.manufacturer || 'Unknown'
    });
  }
}

console.log(JSON.stringify(uniqueDevices, null, 2));
