package web

import "testing"

func TestNormalizeListenAddr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "127.0.0.1:8765"},
		{"127.0.0.1", "127.0.0.1:8765"},
		{"127.0.0.1:8765", "127.0.0.1:8765"},
		{"localhost", "localhost:8765"},
		{":8080", ":8080"},
		{"0.0.0.0:3000", "0.0.0.0:3000"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeListenAddr(tt.in); got != tt.want {
				t.Fatalf("NormalizeListenAddr(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
