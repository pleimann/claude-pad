package action

import (
	"github.com/mike/claude-pad/internal/config"
	"github.com/mike/claude-pad/internal/gesture"
)

// Mapper maps gestures to key sequences based on configuration
type Mapper struct {
	buttonMap map[string][]string // gesture.Key() -> keys
	chordMap  map[string][]string // sorted button list -> keys
}

// NewMapper creates a new action mapper from configuration
func NewMapper(cfg *config.Config) *Mapper {
	m := &Mapper{
		buttonMap: make(map[string][]string),
		chordMap:  make(map[string][]string),
	}

	// Build button mappings
	for _, btn := range cfg.Buttons {
		if btn.Press != nil {
			g := gesture.NewPressGesture(btn.Index)
			m.buttonMap[g.Key()] = btn.Press.Keys
		}
		if btn.DoublePress != nil {
			g := gesture.NewDoublePressGesture(btn.Index)
			m.buttonMap[g.Key()] = btn.DoublePress.Keys
		}
		if btn.LongPress != nil {
			g := gesture.NewLongPressGesture(btn.Index)
			m.buttonMap[g.Key()] = btn.LongPress.Keys
		}
	}

	// Build chord mappings
	for _, chord := range cfg.Chords {
		g := gesture.NewChordGesture(chord.Buttons)
		m.chordMap[g.Key()] = chord.Keys
	}

	return m
}

// Map returns the key sequence for a gesture, or nil if not mapped
func (m *Mapper) Map(g gesture.Gesture) []string {
	key := g.Key()

	// Check chord mappings first
	if g.Type == gesture.GestureChord {
		if keys, ok := m.chordMap[key]; ok {
			return keys
		}
	}

	// Check button mappings
	if keys, ok := m.buttonMap[key]; ok {
		return keys
	}

	return nil
}

// Reload updates the mapper with new configuration
func (m *Mapper) Reload(cfg *config.Config) {
	newMapper := NewMapper(cfg)
	m.buttonMap = newMapper.buttonMap
	m.chordMap = newMapper.chordMap
}
