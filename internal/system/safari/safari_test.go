package safari

import (
	"testing"
)

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{`say "hello"`, `say \"hello\"`},
		{`path\to\file`, `path\\to\\file`},
		{"", ""},
		{`a\b"c`, `a\\b\"c`},
		{"no special chars", "no special chars"},
		{`"quoted"`, `\"quoted\"`},
		{`back\\slash`, `back\\\\slash`},
	}

	for _, tt := range tests {
		got := escapeAppleScript(tt.input)
		if got != tt.want {
			t.Errorf("escapeAppleScript(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
