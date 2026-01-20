package gesture

import (
	"fmt"
	"sort"
	"strings"
)

// GestureType represents the type of gesture detected
type GestureType int

const (
	GesturePress GestureType = iota
	GestureDoublePress
	GestureLongPress
	GestureChord
)

func (g GestureType) String() string {
	switch g {
	case GesturePress:
		return "press"
	case GestureDoublePress:
		return "double_press"
	case GestureLongPress:
		return "long_press"
	case GestureChord:
		return "chord"
	default:
		return fmt.Sprintf("unknown(%d)", g)
	}
}

// Gesture represents a detected gesture
type Gesture struct {
	Type    GestureType
	Buttons []int // Button indices involved in the gesture
}

func (g Gesture) String() string {
	return fmt.Sprintf("%s(%v)", g.Type, g.Buttons)
}

// NewPressGesture creates a single press gesture for a button
func NewPressGesture(button int) Gesture {
	return Gesture{
		Type:    GesturePress,
		Buttons: []int{button},
	}
}

// NewDoublePressGesture creates a double press gesture for a button
func NewDoublePressGesture(button int) Gesture {
	return Gesture{
		Type:    GestureDoublePress,
		Buttons: []int{button},
	}
}

// NewLongPressGesture creates a long press gesture for a button
func NewLongPressGesture(button int) Gesture {
	return Gesture{
		Type:    GestureLongPress,
		Buttons: []int{button},
	}
}

// NewChordGesture creates a chord gesture for multiple buttons
func NewChordGesture(buttons []int) Gesture {
	// Sort buttons for consistent matching
	sorted := make([]int, len(buttons))
	copy(sorted, buttons)
	sort.Ints(sorted)
	return Gesture{
		Type:    GestureChord,
		Buttons: sorted,
	}
}

// MatchesChord checks if this gesture matches a chord configuration
func (g Gesture) MatchesChord(chordButtons []int) bool {
	if g.Type != GestureChord {
		return false
	}
	if len(g.Buttons) != len(chordButtons) {
		return false
	}

	// Sort chord buttons for comparison
	sorted := make([]int, len(chordButtons))
	copy(sorted, chordButtons)
	sort.Ints(sorted)

	for i, b := range g.Buttons {
		if b != sorted[i] {
			return false
		}
	}
	return true
}

// Key returns a unique key for this gesture, used for mapping lookups
func (g Gesture) Key() string {
	var sb strings.Builder
	sb.WriteString(g.Type.String())
	sb.WriteString(":")
	for i, b := range g.Buttons {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "%d", b)
	}
	return sb.String()
}
