package gesture

import (
	"reflect"
	"testing"
)

func TestGestureTypeString(t *testing.T) {
	tests := []struct {
		gt   GestureType
		want string
	}{
		{GesturePress, "press"},
		{GestureDoublePress, "double_press"},
		{GestureLongPress, "long_press"},
		{GestureChord, "chord"},
		{GestureType(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.gt.String(); got != tt.want {
				t.Errorf("GestureType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewPressGesture(t *testing.T) {
	g := NewPressGesture(5)
	if g.Type != GesturePress {
		t.Errorf("Type = %v, want GesturePress", g.Type)
	}
	if !reflect.DeepEqual(g.Buttons, []int{5}) {
		t.Errorf("Buttons = %v, want [5]", g.Buttons)
	}
}

func TestNewDoublePressGesture(t *testing.T) {
	g := NewDoublePressGesture(3)
	if g.Type != GestureDoublePress {
		t.Errorf("Type = %v, want GestureDoublePress", g.Type)
	}
	if !reflect.DeepEqual(g.Buttons, []int{3}) {
		t.Errorf("Buttons = %v, want [3]", g.Buttons)
	}
}

func TestNewLongPressGesture(t *testing.T) {
	g := NewLongPressGesture(7)
	if g.Type != GestureLongPress {
		t.Errorf("Type = %v, want GestureLongPress", g.Type)
	}
	if !reflect.DeepEqual(g.Buttons, []int{7}) {
		t.Errorf("Buttons = %v, want [7]", g.Buttons)
	}
}

func TestNewChordGesture(t *testing.T) {
	// Test that buttons are sorted
	g := NewChordGesture([]int{5, 2, 8, 1})
	if g.Type != GestureChord {
		t.Errorf("Type = %v, want GestureChord", g.Type)
	}
	want := []int{1, 2, 5, 8}
	if !reflect.DeepEqual(g.Buttons, want) {
		t.Errorf("Buttons = %v, want %v (sorted)", g.Buttons, want)
	}
}

func TestGestureMatchesChord(t *testing.T) {
	tests := []struct {
		name         string
		gesture      Gesture
		chordButtons []int
		want         bool
	}{
		{
			name:         "matching chord",
			gesture:      NewChordGesture([]int{0, 1, 2}),
			chordButtons: []int{0, 1, 2},
			want:         true,
		},
		{
			name:         "matching chord different order",
			gesture:      NewChordGesture([]int{2, 0, 1}),
			chordButtons: []int{1, 2, 0},
			want:         true,
		},
		{
			name:         "non-matching chord",
			gesture:      NewChordGesture([]int{0, 1}),
			chordButtons: []int{0, 2},
			want:         false,
		},
		{
			name:         "different length",
			gesture:      NewChordGesture([]int{0, 1}),
			chordButtons: []int{0, 1, 2},
			want:         false,
		},
		{
			name:         "non-chord gesture",
			gesture:      NewPressGesture(0),
			chordButtons: []int{0},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gesture.MatchesChord(tt.chordButtons); got != tt.want {
				t.Errorf("MatchesChord(%v) = %v, want %v", tt.chordButtons, got, tt.want)
			}
		})
	}
}

func TestGestureKey(t *testing.T) {
	tests := []struct {
		name    string
		gesture Gesture
		want    string
	}{
		{
			name:    "press button 0",
			gesture: NewPressGesture(0),
			want:    "press:0",
		},
		{
			name:    "double press button 5",
			gesture: NewDoublePressGesture(5),
			want:    "double_press:5",
		},
		{
			name:    "long press button 3",
			gesture: NewLongPressGesture(3),
			want:    "long_press:3",
		},
		{
			name:    "chord buttons 1,2,5",
			gesture: NewChordGesture([]int{5, 2, 1}), // Will be sorted
			want:    "chord:1,2,5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gesture.Key(); got != tt.want {
				t.Errorf("Key() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGestureString(t *testing.T) {
	g := NewChordGesture([]int{0, 2})
	s := g.String()
	if s != "chord([0 2])" {
		t.Errorf("String() = %q, want %q", s, "chord([0 2])")
	}
}
