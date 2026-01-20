package hid

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *Event
		wantErr bool
	}{
		{
			name: "press single button",
			data: func() []byte {
				buf := make([]byte, 8)
				buf[0] = ReportIDButtonEvent
				buf[1] = EventTypePress
				binary.LittleEndian.PutUint16(buf[2:4], 0x0001) // Button 0
				binary.LittleEndian.PutUint32(buf[4:8], 12345)  // Timestamp
				return buf
			}(),
			want: &Event{
				Type:       Press,
				ButtonMask: 0x0001,
				Timestamp:  12345,
			},
		},
		{
			name: "release multiple buttons",
			data: func() []byte {
				buf := make([]byte, 8)
				buf[0] = ReportIDButtonEvent
				buf[1] = EventTypeRelease
				binary.LittleEndian.PutUint16(buf[2:4], 0x0005) // Buttons 0 and 2
				binary.LittleEndian.PutUint32(buf[4:8], 99999)
				return buf
			}(),
			want: &Event{
				Type:       Release,
				ButtonMask: 0x0005,
				Timestamp:  99999,
			},
		},
		{
			name:    "data too short",
			data:    []byte{0x01, 0x01, 0x00},
			wantErr: true,
		},
		{
			name: "wrong report ID",
			data: func() []byte {
				buf := make([]byte, 8)
				buf[0] = 0xFF // Wrong report ID
				buf[1] = EventTypePress
				return buf
			}(),
			wantErr: true,
		},
		{
			name: "unknown event type",
			data: func() []byte {
				buf := make([]byte, 8)
				buf[0] = ReportIDButtonEvent
				buf[1] = 0xFF // Unknown event type
				return buf
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEvent(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseEvent() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestEventPressedButtons(t *testing.T) {
	tests := []struct {
		name       string
		buttonMask uint16
		want       []int
	}{
		{
			name:       "no buttons",
			buttonMask: 0x0000,
			want:       nil,
		},
		{
			name:       "button 0",
			buttonMask: 0x0001,
			want:       []int{0},
		},
		{
			name:       "button 7",
			buttonMask: 0x0080,
			want:       []int{7},
		},
		{
			name:       "buttons 0, 2, 4",
			buttonMask: 0x0015,
			want:       []int{0, 2, 4},
		},
		{
			name:       "all 16 buttons",
			buttonMask: 0xFFFF,
			want:       []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{ButtonMask: tt.buttonMask}
			got := e.PressedButtons()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PressedButtons() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDisplayFrameEncode(t *testing.T) {
	tests := []struct {
		name  string
		frame *DisplayFrame
		check func([]byte) bool
	}{
		{
			name:  "full frame",
			frame: NewFullFrame(128, 64, []byte{0xAA, 0xBB, 0xCC}),
			check: func(data []byte) bool {
				return data[0] == ReportIDDisplay &&
					data[1] == DisplayCmdFullFrame &&
					binary.LittleEndian.Uint16(data[2:4]) == 0 && // X
					binary.LittleEndian.Uint16(data[4:6]) == 0 && // Y
					binary.LittleEndian.Uint16(data[6:8]) == 128 && // Width
					binary.LittleEndian.Uint16(data[8:10]) == 64 && // Height
					data[10] == 0xAA && data[11] == 0xBB && data[12] == 0xCC
			},
		},
		{
			name:  "partial frame",
			frame: NewPartialFrame(10, 20, 32, 16, []byte{0x11, 0x22}),
			check: func(data []byte) bool {
				return data[0] == ReportIDDisplay &&
					data[1] == DisplayCmdPartial &&
					binary.LittleEndian.Uint16(data[2:4]) == 10 && // X
					binary.LittleEndian.Uint16(data[4:6]) == 20 && // Y
					binary.LittleEndian.Uint16(data[6:8]) == 32 && // Width
					binary.LittleEndian.Uint16(data[8:10]) == 16 && // Height
					data[10] == 0x11 && data[11] == 0x22
			},
		},
		{
			name:  "clear command",
			frame: NewClearCommand(),
			check: func(data []byte) bool {
				return data[0] == ReportIDDisplay &&
					data[1] == DisplayCmdClear &&
					len(data) == 10 // Header only
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.frame.Encode()
			if !tt.check(data) {
				t.Errorf("Encode() = %v, check failed", data)
			}
		})
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{Press, "press"},
		{Release, "release"},
		{EventType(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.et.String(); got != tt.want {
				t.Errorf("EventType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
