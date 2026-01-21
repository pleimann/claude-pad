package action

import (
	"reflect"
	"testing"

	"github.com/pleimann/camel-pad/internal/config"
	"github.com/pleimann/camel-pad/internal/gesture"
)

func TestMapperMap(t *testing.T) {
	cfg := &config.Config{
		Buttons: []config.Button{
			{
				Index: 0,
				Name:  "btn_a",
				Press: &config.KeyAction{
					Keys: []string{"ctrl+c"},
				},
				DoublePress: &config.KeyAction{
					Keys: []string{"ctrl+z"},
				},
				LongPress: &config.KeyAction{
					Keys: []string{"q", "enter"},
				},
			},
			{
				Index: 1,
				Name:  "btn_b",
				Press: &config.KeyAction{
					Keys: []string{"down"},
				},
			},
		},
		Chords: []config.Chord{
			{
				Buttons: []int{0, 1},
				Keys:    []string{"ctrl+r"},
			},
			{
				Buttons: []int{0, 1, 2},
				Keys:    []string{"ctrl+alt+delete"},
			},
		},
	}

	mapper := NewMapper(cfg)

	tests := []struct {
		name    string
		gesture gesture.Gesture
		want    []string
	}{
		{
			name:    "button 0 press",
			gesture: gesture.NewPressGesture(0),
			want:    []string{"ctrl+c"},
		},
		{
			name:    "button 0 double press",
			gesture: gesture.NewDoublePressGesture(0),
			want:    []string{"ctrl+z"},
		},
		{
			name:    "button 0 long press",
			gesture: gesture.NewLongPressGesture(0),
			want:    []string{"q", "enter"},
		},
		{
			name:    "button 1 press",
			gesture: gesture.NewPressGesture(1),
			want:    []string{"down"},
		},
		{
			name:    "chord 0+1",
			gesture: gesture.NewChordGesture([]int{0, 1}),
			want:    []string{"ctrl+r"},
		},
		{
			name:    "chord 0+1 reversed order",
			gesture: gesture.NewChordGesture([]int{1, 0}),
			want:    []string{"ctrl+r"}, // Should still match due to sorting
		},
		{
			name:    "chord 0+1+2",
			gesture: gesture.NewChordGesture([]int{2, 0, 1}),
			want:    []string{"ctrl+alt+delete"},
		},
		{
			name:    "unmapped button",
			gesture: gesture.NewPressGesture(5),
			want:    nil,
		},
		{
			name:    "unmapped gesture type",
			gesture: gesture.NewDoublePressGesture(1), // btn_b has no double press
			want:    nil,
		},
		{
			name:    "unmapped chord",
			gesture: gesture.NewChordGesture([]int{0, 2}),
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.Map(tt.gesture)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Map(%v) = %v, want %v", tt.gesture, got, tt.want)
			}
		})
	}
}

func TestMapperReload(t *testing.T) {
	cfg1 := &config.Config{
		Buttons: []config.Button{
			{
				Index: 0,
				Press: &config.KeyAction{Keys: []string{"a"}},
			},
		},
	}

	mapper := NewMapper(cfg1)

	// Verify initial mapping
	got := mapper.Map(gesture.NewPressGesture(0))
	if !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("initial Map() = %v, want [a]", got)
	}

	// Reload with new config
	cfg2 := &config.Config{
		Buttons: []config.Button{
			{
				Index: 0,
				Press: &config.KeyAction{Keys: []string{"b", "c"}},
			},
		},
	}

	mapper.Reload(cfg2)

	// Verify new mapping
	got = mapper.Map(gesture.NewPressGesture(0))
	if !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Errorf("after Reload() Map() = %v, want [b c]", got)
	}
}

func TestMapperEmptyConfig(t *testing.T) {
	cfg := &config.Config{}
	mapper := NewMapper(cfg)

	// Should return nil for any gesture
	if got := mapper.Map(gesture.NewPressGesture(0)); got != nil {
		t.Errorf("Map() with empty config = %v, want nil", got)
	}
}
