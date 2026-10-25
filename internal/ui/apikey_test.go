package ui

import "testing"

func TestKeyFromEvent(t *testing.T) {
	tests := []struct {
		name string
		data any
		want string
	}{
		{"map", map[string]any{"key": "sk-1"}, "sk-1"},
		{"wrapped in a list", []any{map[string]any{"key": "sk-2"}}, "sk-2"},
		{"no key", map[string]any{"other": "x"}, ""},
		{"wrong type", map[string]any{"key": 5}, ""},
		{"nil", nil, ""},
		{"string", "sk-3", ""},
		{"empty list", []any{}, ""},
	}
	for _, tt := range tests {
		if got := keyFromEvent(tt.data); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestIDFromEvent(t *testing.T) {
	tests := []struct {
		name string
		data any
		want uint64
		ok   bool
	}{
		{"map", map[string]any{"id": float64(7)}, 7, true},
		{"wrapped in a list", []any{map[string]any{"id": float64(12)}}, 12, true},
		{"missing", map[string]any{}, 0, false},
		{"negative", map[string]any{"id": float64(-1)}, 0, false},
		{"wrong type", map[string]any{"id": "7"}, 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		got, ok := idFromEvent(tt.data)
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: got (%d, %v), want (%d, %v)", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}
