package keyout

import "testing"

func TestParseKeyName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want VKCode
		ok   bool
	}{
		{"letter Z", "Z", 0x5A, true},
		{"letter lower", "a", 0x41, true},
		{"digit", "7", 0x37, true},
		{"UP", "UP", VKUp, true},
		{"down lower", "down", VKDown, true},
		{"ENTER", "ENTER", VKReturn, true},
		{"BACKSPACE", "BACKSPACE", VKBack, true},
		{"F12", "F12", 0x7B, true},
		{"unknown", "FOO", 0, false},
		{"empty", "", 0, false},
		{"F0 invalid", "F0", 0, false},
		{"F25 invalid", "F25", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKeyName(tt.in)
			if tt.ok {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got %#x, want %#x", got, tt.want)
				}
			} else if err == nil {
				t.Fatalf("expected error, got %#x", got)
			}
		})
	}
}
