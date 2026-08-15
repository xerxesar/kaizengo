package extension

import "testing"

func TestMatchViewPattern(t *testing.T) {
	tests := []struct {
		pattern string
		view    string
		want    bool
	}{
		{"GreetingList", "GreetingList", true},
		{"*.list", "GreetingList", true},
		{"*.list", "NoteList", true},
		{"*.list", "GreetingForm", false},
		{"NoteList", "NoteList", true},
		{"NoteList", "GreetingList", false},
	}
	for _, tc := range tests {
		got := matchViewPattern(tc.pattern, tc.view)
		if got != tc.want {
			t.Errorf("matchViewPattern(%q, %q) = %v, want %v", tc.pattern, tc.view, got, tc.want)
		}
	}
}
