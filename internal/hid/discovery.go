package hid

import (
	"github.com/karalabe/hid"
)

// DeviceInfo contains information about a discovered HID device
type DeviceInfo struct {
	VendorID     uint16
	ProductID    uint16
	Path         string
	Manufacturer string
	Product      string
	SerialNumber string
	UsagePage    uint16
	Usage        uint16
}

// ListDevices returns a list of all available HID devices
func ListDevices() ([]DeviceInfo, error) {
	devices := hid.Enumerate(0, 0)

	result := make([]DeviceInfo, len(devices))
	for i, d := range devices {
		result[i] = DeviceInfo{
			VendorID:     d.VendorID,
			ProductID:    d.ProductID,
			Path:         d.Path,
			Manufacturer: d.Manufacturer,
			Product:      d.Product,
			SerialNumber: d.Serial,
			UsagePage:    d.UsagePage,
			Usage:        d.Usage,
		}
	}

	return result, nil
}

// FindDevice searches for a device matching the given vendor and product IDs
func FindDevice(vendorID, productID uint16) (*DeviceInfo, error) {
	devices := hid.Enumerate(vendorID, productID)
	if len(devices) == 0 {
		return nil, nil
	}

	d := devices[0]
	return &DeviceInfo{
		VendorID:     d.VendorID,
		ProductID:    d.ProductID,
		Path:         d.Path,
		Manufacturer: d.Manufacturer,
		Product:      d.Product,
		SerialNumber: d.Serial,
		UsagePage:    d.UsagePage,
		Usage:        d.Usage,
	}, nil
}
