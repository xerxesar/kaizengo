package module

import "testing"

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.1.0", "0.2.0", true},
		{"0.2.0", "0.1.0", false},
		{"0.1.0", "0.1.0", false},
		{"0.3.0", "0.10.0", true},
		{"1.0", "1.0.1", true},
		{"1.0.1", "1.0", false},
		{"", "0.1.0", true},
		{"0.1.0", "", false},
	}
	for _, tc := range cases {
		if got := VersionLess(tc.a, tc.b); got != tc.want {
			t.Errorf("VersionLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
