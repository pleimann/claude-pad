package action

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// KeyWriter is the interface for writing key sequences
type KeyWriter interface {
	WriteKey(key KeyPress) error
}

// Executor executes key sequences
type Executor struct {
	writer KeyWriter
}

// NewExecutor creates a new action executor
func NewExecutor(writer KeyWriter) *Executor {
	return &Executor{writer: writer}
}

// Execute executes a sequence of key strings
func (e *Executor) Execute(keys []string) error {
	for _, keyStr := range keys {
		key, err := ParseKey(keyStr)
		if err != nil {
			return fmt.Errorf("invalid key %q: %w", keyStr, err)
		}
		if err := e.writer.WriteKey(key); err != nil {
			return fmt.Errorf("failed to write key %q: %w", keyStr, err)
		}
	}
	return nil
}

// KeyPress represents a parsed key with modifiers
type KeyPress struct {
	Ctrl  bool
	Alt   bool
	Shift bool
	Meta  bool
	Key   string // The base key (e.g., "c", "enter", "f1")
}

// ParseKey parses a key string like "ctrl+shift+c" into a KeyPress
func ParseKey(s string) (KeyPress, error) {
	var kp KeyPress

	parts := strings.Split(strings.ToLower(s), "+")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Last part is the actual key
		if i == len(parts)-1 {
			kp.Key = part
			break
		}

		// Otherwise it's a modifier
		switch part {
		case "ctrl", "control":
			kp.Ctrl = true
		case "alt", "option":
			kp.Alt = true
		case "shift":
			kp.Shift = true
		case "meta", "cmd", "command", "win", "super":
			kp.Meta = true
		default:
			return KeyPress{}, fmt.Errorf("unknown modifier: %s", part)
		}
	}

	if kp.Key == "" {
		return KeyPress{}, fmt.Errorf("no key specified")
	}

	if !isValidKey(kp.Key) {
		return KeyPress{}, fmt.Errorf("invalid key: %s", kp.Key)
	}

	return kp, nil
}

// isValidKey checks if a key name is valid
func isValidKey(key string) bool {
	// Single printable character
	if utf8.RuneCountInString(key) == 1 {
		return true
	}

	// Special keys
	specialKeys := map[string]bool{
		"enter":     true,
		"return":    true,
		"tab":       true,
		"esc":       true,
		"escape":    true,
		"space":     true,
		"backspace": true,
		"delete":    true,
		"del":       true,
		"insert":    true,
		"ins":       true,
		"home":      true,
		"end":       true,
		"pageup":    true,
		"pgup":      true,
		"pagedown":  true,
		"pgdn":      true,
		"up":        true,
		"down":      true,
		"left":      true,
		"right":     true,
		"f1":        true,
		"f2":        true,
		"f3":        true,
		"f4":        true,
		"f5":        true,
		"f6":        true,
		"f7":        true,
		"f8":        true,
		"f9":        true,
		"f10":       true,
		"f11":       true,
		"f12":       true,
	}

	return specialKeys[key]
}

// ToBytes converts a KeyPress to the bytes to write to a PTY
func (kp KeyPress) ToBytes() []byte {
	// Handle control characters
	if kp.Ctrl && !kp.Alt && !kp.Meta {
		if len(kp.Key) == 1 {
			char := kp.Key[0]
			// Control characters are ASCII 1-26 for ctrl+a through ctrl+z
			if char >= 'a' && char <= 'z' {
				return []byte{char - 'a' + 1}
			}
			if char >= 'A' && char <= 'Z' {
				return []byte{char - 'A' + 1}
			}
			// Some special control combos
			switch char {
			case '[':
				return []byte{0x1b} // ESC
			case '\\':
				return []byte{0x1c} // FS
			case ']':
				return []byte{0x1d} // GS
			case '^':
				return []byte{0x1e} // RS
			case '_':
				return []byte{0x1f} // US
			case '?':
				return []byte{0x7f} // DEL
			}
		}
	}

	// Handle special keys
	switch kp.Key {
	case "enter", "return":
		return []byte{'\r'}
	case "tab":
		return []byte{'\t'}
	case "esc", "escape":
		return []byte{0x1b}
	case "space":
		return []byte{' '}
	case "backspace":
		return []byte{0x7f}
	case "delete", "del":
		return []byte{0x1b, '[', '3', '~'}
	case "insert", "ins":
		return []byte{0x1b, '[', '2', '~'}
	case "home":
		return []byte{0x1b, '[', 'H'}
	case "end":
		return []byte{0x1b, '[', 'F'}
	case "pageup", "pgup":
		return []byte{0x1b, '[', '5', '~'}
	case "pagedown", "pgdn":
		return []byte{0x1b, '[', '6', '~'}
	case "up":
		return []byte{0x1b, '[', 'A'}
	case "down":
		return []byte{0x1b, '[', 'B'}
	case "right":
		return []byte{0x1b, '[', 'C'}
	case "left":
		return []byte{0x1b, '[', 'D'}
	case "f1":
		return []byte{0x1b, 'O', 'P'}
	case "f2":
		return []byte{0x1b, 'O', 'Q'}
	case "f3":
		return []byte{0x1b, 'O', 'R'}
	case "f4":
		return []byte{0x1b, 'O', 'S'}
	case "f5":
		return []byte{0x1b, '[', '1', '5', '~'}
	case "f6":
		return []byte{0x1b, '[', '1', '7', '~'}
	case "f7":
		return []byte{0x1b, '[', '1', '8', '~'}
	case "f8":
		return []byte{0x1b, '[', '1', '9', '~'}
	case "f9":
		return []byte{0x1b, '[', '2', '0', '~'}
	case "f10":
		return []byte{0x1b, '[', '2', '1', '~'}
	case "f11":
		return []byte{0x1b, '[', '2', '3', '~'}
	case "f12":
		return []byte{0x1b, '[', '2', '4', '~'}
	}

	// Alt key sends ESC prefix
	if kp.Alt {
		if len(kp.Key) == 1 {
			return []byte{0x1b, kp.Key[0]}
		}
	}

	// Plain character
	if len(kp.Key) == 1 {
		char := kp.Key[0]
		if kp.Shift && char >= 'a' && char <= 'z' {
			return []byte{char - 32} // Uppercase
		}
		return []byte{char}
	}

	return nil
}
