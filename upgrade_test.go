package main

import "testing"

func TestIsNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"0.2.0", "0.1.9", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.1.0-dev", false},
		{"1.0", "0.9.9", true},
		{"bad", "0.1.0", false},
	} {
		if got := isNewer(c.a, c.b); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
