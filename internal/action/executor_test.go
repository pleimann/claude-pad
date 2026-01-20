package action

import (
	"reflect"
	"testing"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    KeyPress
		wantErr bool
	}{
		{
			name:  "simple letter",
			input: "a",
			want:  KeyPress{Key: "a"},
		},
		{
			name:  "uppercase letter",
			input: "A",
			want:  KeyPress{Key: "a"}, // Normalized to lowercase
		},
		{
			name:  "number",
			input: "5",
			want:  KeyPress{Key: "5"},
		},
		{
			name:  "special key",
			input: "enter",
			want:  KeyPress{Key: "enter"},
		},
		{
			name:  "ctrl modifier",
			input: "ctrl+c",
			want:  KeyPress{Ctrl: true, Key: "c"},
		},
		{
			name:  "alt modifier",
			input: "alt+f4",
			want:  KeyPress{Alt: true, Key: "f4"},
		},
		{
			name:  "shift modifier",
			input: "shift+tab",
			want:  KeyPress{Shift: true, Key: "tab"},
		},
		{
			name:  "meta modifier",
			input: "meta+a",
			want:  KeyPress{Meta: true, Key: "a"},
		},
		{
			name:  "cmd modifier alias",
			input: "cmd+q",
			want:  KeyPress{Meta: true, Key: "q"},
		},
		{
			name:  "multiple modifiers",
			input: "ctrl+shift+z",
			want:  KeyPress{Ctrl: true, Shift: true, Key: "z"},
		},
		{
			name:  "all modifiers",
			input: "ctrl+alt+shift+meta+x",
			want:  KeyPress{Ctrl: true, Alt: true, Shift: true, Meta: true, Key: "x"},
		},
		{
			name:  "function key",
			input: "f12",
			want:  KeyPress{Key: "f12"},
		},
		{
			name:  "arrow key",
			input: "up",
			want:  KeyPress{Key: "up"},
		},
		{
			name:  "escape",
			input: "esc",
			want:  KeyPress{Key: "esc"},
		},
		{
			name:  "escape full name",
			input: "escape",
			want:  KeyPress{Key: "escape"},
		},
		{
			name:    "unknown modifier",
			input:   "foo+a",
			wantErr: true,
		},
		{
			name:    "empty key",
			input:   "ctrl+",
			wantErr: true,
		},
		{
			name:    "invalid key name",
			input:   "invalid_key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseKey(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestKeyPressToBytes(t *testing.T) {
	tests := []struct {
		name string
		key  KeyPress
		want []byte
	}{
		{
			name: "ctrl+c",
			key:  KeyPress{Ctrl: true, Key: "c"},
			want: []byte{0x03}, // ASCII ETX (ctrl+c)
		},
		{
			name: "ctrl+a",
			key:  KeyPress{Ctrl: true, Key: "a"},
			want: []byte{0x01}, // ASCII SOH
		},
		{
			name: "ctrl+z",
			key:  KeyPress{Ctrl: true, Key: "z"},
			want: []byte{0x1a}, // ASCII SUB
		},
		{
			name: "enter",
			key:  KeyPress{Key: "enter"},
			want: []byte{'\r'},
		},
		{
			name: "tab",
			key:  KeyPress{Key: "tab"},
			want: []byte{'\t'},
		},
		{
			name: "escape",
			key:  KeyPress{Key: "esc"},
			want: []byte{0x1b},
		},
		{
			name: "space",
			key:  KeyPress{Key: "space"},
			want: []byte{' '},
		},
		{
			name: "backspace",
			key:  KeyPress{Key: "backspace"},
			want: []byte{0x7f},
		},
		{
			name: "up arrow",
			key:  KeyPress{Key: "up"},
			want: []byte{0x1b, '[', 'A'},
		},
		{
			name: "down arrow",
			key:  KeyPress{Key: "down"},
			want: []byte{0x1b, '[', 'B'},
		},
		{
			name: "right arrow",
			key:  KeyPress{Key: "right"},
			want: []byte{0x1b, '[', 'C'},
		},
		{
			name: "left arrow",
			key:  KeyPress{Key: "left"},
			want: []byte{0x1b, '[', 'D'},
		},
		{
			name: "home",
			key:  KeyPress{Key: "home"},
			want: []byte{0x1b, '[', 'H'},
		},
		{
			name: "end",
			key:  KeyPress{Key: "end"},
			want: []byte{0x1b, '[', 'F'},
		},
		{
			name: "page up",
			key:  KeyPress{Key: "pageup"},
			want: []byte{0x1b, '[', '5', '~'},
		},
		{
			name: "page down",
			key:  KeyPress{Key: "pagedown"},
			want: []byte{0x1b, '[', '6', '~'},
		},
		{
			name: "delete",
			key:  KeyPress{Key: "delete"},
			want: []byte{0x1b, '[', '3', '~'},
		},
		{
			name: "insert",
			key:  KeyPress{Key: "insert"},
			want: []byte{0x1b, '[', '2', '~'},
		},
		{
			name: "f1",
			key:  KeyPress{Key: "f1"},
			want: []byte{0x1b, 'O', 'P'},
		},
		{
			name: "f5",
			key:  KeyPress{Key: "f5"},
			want: []byte{0x1b, '[', '1', '5', '~'},
		},
		{
			name: "f12",
			key:  KeyPress{Key: "f12"},
			want: []byte{0x1b, '[', '2', '4', '~'},
		},
		{
			name: "alt+x",
			key:  KeyPress{Alt: true, Key: "x"},
			want: []byte{0x1b, 'x'},
		},
		{
			name: "plain letter",
			key:  KeyPress{Key: "x"},
			want: []byte{'x'},
		},
		{
			name: "shift+letter",
			key:  KeyPress{Shift: true, Key: "a"},
			want: []byte{'A'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.key.ToBytes()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidKey(t *testing.T) {
	validKeys := []string{
		"a", "z", "0", "9", "/", "-",
		"enter", "return", "tab", "esc", "escape",
		"space", "backspace", "delete", "del",
		"insert", "ins", "home", "end",
		"pageup", "pgup", "pagedown", "pgdn",
		"up", "down", "left", "right",
		"f1", "f2", "f3", "f4", "f5", "f6",
		"f7", "f8", "f9", "f10", "f11", "f12",
	}

	for _, key := range validKeys {
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}

	invalidKeys := []string{
		"foo", "bar", "invalid", "f13", "ctrl",
	}

	for _, key := range invalidKeys {
		if isValidKey(key) {
			t.Errorf("isValidKey(%q) = true, want false", key)
		}
	}
}

// Mock KeyWriter for testing Executor
type mockKeyWriter struct {
	keys []KeyPress
}

func (m *mockKeyWriter) WriteKey(key KeyPress) error {
	m.keys = append(m.keys, key)
	return nil
}

func TestExecutorExecute(t *testing.T) {
	mock := &mockKeyWriter{}
	executor := NewExecutor(mock)

	keys := []string{"ctrl+c", "enter", "a"}
	err := executor.Execute(keys)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(mock.keys) != 3 {
		t.Fatalf("wrote %d keys, want 3", len(mock.keys))
	}

	if !mock.keys[0].Ctrl || mock.keys[0].Key != "c" {
		t.Errorf("key[0] = %+v, want ctrl+c", mock.keys[0])
	}
	if mock.keys[1].Key != "enter" {
		t.Errorf("key[1] = %+v, want enter", mock.keys[1])
	}
	if mock.keys[2].Key != "a" {
		t.Errorf("key[2] = %+v, want a", mock.keys[2])
	}
}

func TestExecutorExecuteInvalidKey(t *testing.T) {
	mock := &mockKeyWriter{}
	executor := NewExecutor(mock)

	err := executor.Execute([]string{"invalid_key_name"})
	if err == nil {
		t.Error("Execute() expected error for invalid key, got nil")
	}
}
