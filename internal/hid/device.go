package hid

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/karalabe/hid"
)

// Device represents a connection to the macropad HID device
type Device struct {
	vendorID  uint16
	productID uint16
	device    *hid.Device
	mu        sync.Mutex
	closed    bool
}

func executableName() string {
	return "camel-pad"
}

// NewDevice opens a connection to a HID device with the specified vendor and product IDs
func NewDevice(vendorID, productID uint16) (*Device, error) {
	devices := hid.Enumerate(vendorID, productID)
	if len(devices) == 0 {
		// List available devices to help user find the right one
		allDevices := hid.Enumerate(0, 0)
		if len(allDevices) == 0 {
			return nil, fmt.Errorf("no HID devices found on system - check USB connection")
		}
		return nil, fmt.Errorf("no device found with VendorID=0x%04X, ProductID=0x%04X\n"+
			"  Run '"+executableName()+" --list-devices' to see available devices\n"+
			"  Run '"+executableName()+" set-device' to configure the correct device",
			vendorID, productID)
	}

	// Try to open each matching interface until one succeeds
	// Some devices have multiple interfaces, not all of which can be opened
	var lastErr error
	for i, devInfo := range devices {
		dev, err := devInfo.Open()
		if err == nil {
			return &Device{
				vendorID:  vendorID,
				productID: productID,
				device:    dev,
			}, nil
		}
		lastErr = err
		// Continue trying other interfaces
		_ = i // silence unused variable if logging is disabled
	}

	// All interfaces failed to open
	if len(devices) == 1 {
		return nil, fmt.Errorf("failed to open device 0x%04X:0x%04X: %w\n"+
			"  This may be a permissions issue. On macOS, try:\n"+
			"  1. System Settings > Privacy & Security > Input Monitoring\n"+
			"  2. Add Terminal (or your terminal app) to the list",
			vendorID, productID, lastErr)
	}
	return nil, fmt.Errorf("failed to open any of %d interfaces for device 0x%04X:0x%04X: %w\n"+
		"  This may be a permissions issue. On macOS, try:\n"+
		"  1. System Settings > Privacy & Security > Input Monitoring\n"+
		"  2. Add Terminal (or your terminal app) to the list",
		len(devices), vendorID, productID, lastErr)
}

// Close closes the HID device connection
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true

	if d.device != nil {
		return d.device.Close()
	}
	return nil
}

// ReadEvents continuously reads events from the device and sends them to the channel
func (d *Device) ReadEvents(ctx context.Context, events chan<- Event) error {
	buf := make([]byte, 64)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		d.mu.Lock()
		if d.closed {
			d.mu.Unlock()
			return fmt.Errorf("device closed")
		}
		dev := d.device
		d.mu.Unlock()

		// Read with timeout to allow checking context
		n, err := dev.Read(buf)
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		if n == 0 {
			continue
		}

		event, err := ParseEvent(buf[:n])
		if err != nil {
			// Log but don't fail on parse errors
			continue
		}

		select {
		case events <- *event:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Write sends data to the HID device
func (d *Device) Write(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return fmt.Errorf("device closed")
	}

	_, err := d.device.Write(data)
	return err
}

// SendFrame sends a display frame to the device
func (d *Device) SendFrame(frame *DisplayFrame) error {
	return d.Write(frame.Encode())
}

// Reconnect attempts to reconnect to the device
func (d *Device) Reconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Close existing connection if any
	if d.device != nil {
		d.device.Close()
		d.device = nil
	}
	d.closed = false

	// Try to find and open the device
	devices := hid.Enumerate(d.vendorID, d.productID)
	if len(devices) == 0 {
		return fmt.Errorf("device not found")
	}

	// Try each interface until one opens
	var lastErr error
	for _, devInfo := range devices {
		dev, err := devInfo.Open()
		if err == nil {
			d.device = dev
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("failed to open device: %w", lastErr)
}

// WaitForDevice waits for a device to become available and connects to it
func (d *Device) WaitForDevice(ctx context.Context, pollInterval time.Duration) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := d.Reconnect(); err == nil {
				return nil
			}
		}
	}
}
