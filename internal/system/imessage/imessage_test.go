package imessage

import (
	"testing"
	"time"
)

func TestConvertAppleTimestamp(t *testing.T) {
	// Apple epoch is 2001-01-01 00:00:00 UTC

	tests := []struct {
		name      string
		timestamp int64
		wantEmpty bool
		contains  string
	}{
		{
			name:      "zero returns empty",
			timestamp: 0,
			wantEmpty: true,
		},
		{
			name:      "seconds since epoch",
			timestamp: 700000000, // ~2023-03-10
			contains:  "2023",
		},
		{
			name:      "nanoseconds since epoch",
			timestamp: 2000000000000000000, // > 1e18, treated as nanoseconds
			contains:  "2064",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertAppleTimestamp(tt.timestamp)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty, got %q", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Error("expected non-empty result")
			}
			if tt.contains != "" && len(got) > 0 {
				if !containsStr(got, tt.contains) {
					t.Errorf("expected %q to contain %q", got, tt.contains)
				}
			}
		})
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestConvertAppleTimestampKnownDate(t *testing.T) {
	// 2001-01-01 + 1 day = 2001-01-02 in seconds
	oneDay := int64(86400)
	result := convertAppleTimestamp(oneDay)

	// Parse the result to verify it's 2001-01-02
	parsed, err := time.Parse("2006-01-02 15:04:05", result)
	if err != nil {
		t.Fatalf("failed to parse result %q: %v", result, err)
	}

	// Convert to UTC for comparison (convertAppleTimestamp returns local time)
	utc := parsed.UTC()
	// The date should be 2001-01-02 in UTC (might be off by timezone)
	if utc.Year() != 2001 || utc.Month() != time.January {
		t.Errorf("expected January 2001, got %s", utc.Format("2006-01-02"))
	}
}

func TestConvertAppleTimestampMicroseconds(t *testing.T) {
	// Microseconds range (> 1e15, < 1e18)
	timestamp := int64(700000000000000) // microseconds
	result := convertAppleTimestamp(timestamp)
	if result == "" {
		t.Error("expected non-empty result for microsecond timestamp")
	}
}

func TestConvertAppleTimestampMilliseconds(t *testing.T) {
	// Milliseconds range (> 1e12, < 1e15)
	timestamp := int64(700000000000) // milliseconds
	result := convertAppleTimestamp(timestamp)
	if result == "" {
		t.Error("expected non-empty result for millisecond timestamp")
	}
}

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
	}

	for _, tt := range tests {
		got := escapeAppleScript(tt.input)
		if got != tt.want {
			t.Errorf("escapeAppleScript(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeAppleScriptBackslashFirst(t *testing.T) {
	// Verify backslash is escaped before quotes to prevent double-escaping
	input := `"\`
	got := escapeAppleScript(input)
	// Expected: backslash becomes \\, quote becomes \"
	want := `\"\\`
	if got != want {
		t.Errorf("escapeAppleScript(%q) = %q, want %q", input, got, want)
	}
}
