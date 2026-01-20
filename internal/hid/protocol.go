package hid

import (
	"encoding/binary"
	"fmt"
)

// Report IDs
const (
	ReportIDButtonEvent  byte = 0x01
	ReportIDDisplay      byte = 0x02
)

// Event types for button events
const (
	EventTypePress   byte = 0x01
	EventTypeRelease byte = 0x02
)

// Display commands
const (
	DisplayCmdFullFrame byte = 0x01
	DisplayCmdPartial   byte = 0x02
	DisplayCmdClear     byte = 0x03
)

// Event represents a button event from the device
type Event struct {
	Type       EventType
	ButtonMask uint16
	Timestamp  uint32
}

type EventType byte

const (
	Press   EventType = EventType(EventTypePress)
	Release EventType = EventType(EventTypeRelease)
)

func (e EventType) String() string {
	switch e {
	case Press:
		return "press"
	case Release:
		return "release"
	default:
		return fmt.Sprintf("unknown(%d)", e)
	}
}

// ParseEvent parses a raw HID report into an Event
// Expected format:
//   Byte 0: Report ID (0x01)
//   Byte 1: Event type (0x01=press, 0x02=release)
//   Byte 2-3: Button bitmask (16 buttons max, little-endian)
//   Byte 4-7: Timestamp (ms since boot, little-endian u32)
func ParseEvent(data []byte) (*Event, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("event data too short: %d bytes", len(data))
	}

	if data[0] != ReportIDButtonEvent {
		return nil, fmt.Errorf("unexpected report ID: 0x%02X", data[0])
	}

	eventType := data[1]
	if eventType != EventTypePress && eventType != EventTypeRelease {
		return nil, fmt.Errorf("unknown event type: 0x%02X", eventType)
	}

	return &Event{
		Type:       EventType(eventType),
		ButtonMask: binary.LittleEndian.Uint16(data[2:4]),
		Timestamp:  binary.LittleEndian.Uint32(data[4:8]),
	}, nil
}

// PressedButtons returns a slice of button indices that are pressed
func (e *Event) PressedButtons() []int {
	var buttons []int
	for i := 0; i < 16; i++ {
		if e.ButtonMask&(1<<i) != 0 {
			buttons = append(buttons, i)
		}
	}
	return buttons
}

// DisplayFrame represents a frame to be sent to the OLED display
type DisplayFrame struct {
	Command byte
	X       uint16
	Y       uint16
	Width   uint16
	Height  uint16
	Data    []byte // 1-bit packed pixel data, row-major
}

// Encode serializes the DisplayFrame for transmission
// Format:
//   Byte 0: Report ID (0x02)
//   Byte 1: Command (0x01=full frame, 0x02=partial, 0x03=clear)
//   Byte 2-3: X offset (for partial)
//   Byte 4-5: Y offset (for partial)
//   Byte 6-7: Width
//   Byte 8-9: Height
//   Byte 10+: Pixel data (1-bit packed, row-major)
func (f *DisplayFrame) Encode() []byte {
	headerSize := 10
	buf := make([]byte, headerSize+len(f.Data))

	buf[0] = ReportIDDisplay
	buf[1] = f.Command
	binary.LittleEndian.PutUint16(buf[2:4], f.X)
	binary.LittleEndian.PutUint16(buf[4:6], f.Y)
	binary.LittleEndian.PutUint16(buf[6:8], f.Width)
	binary.LittleEndian.PutUint16(buf[8:10], f.Height)

	if len(f.Data) > 0 {
		copy(buf[headerSize:], f.Data)
	}

	return buf
}

// NewFullFrame creates a full frame display update
func NewFullFrame(width, height uint16, data []byte) *DisplayFrame {
	return &DisplayFrame{
		Command: DisplayCmdFullFrame,
		X:       0,
		Y:       0,
		Width:   width,
		Height:  height,
		Data:    data,
	}
}

// NewPartialFrame creates a partial frame display update
func NewPartialFrame(x, y, width, height uint16, data []byte) *DisplayFrame {
	return &DisplayFrame{
		Command: DisplayCmdPartial,
		X:       x,
		Y:       y,
		Width:   width,
		Height:  height,
		Data:    data,
	}
}

// NewClearCommand creates a display clear command
func NewClearCommand() *DisplayFrame {
	return &DisplayFrame{
		Command: DisplayCmdClear,
	}
}
