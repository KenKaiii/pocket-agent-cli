package calendar

import (
	"testing"
	"time"
)

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		year    int
		month   time.Month
		day     int
	}{
		{"2024-03-15T10:30:00", false, 2024, time.March, 15},
		{"2024-03-15 10:30:00", false, 2024, time.March, 15},
		{"2024-03-15T10:30", false, 2024, time.March, 15},
		{"2024-03-15 10:30", false, 2024, time.March, 15},
		{"2024-03-15", false, 2024, time.March, 15},
		{"03/15/2024 10:30:00", false, 2024, time.March, 15},
		{"03/15/2024 10:30", false, 2024, time.March, 15},
		{"03/15/2024", false, 2024, time.March, 15},
		{"not-a-date", true, 0, 0, 0},
		{"", true, 0, 0, 0},
		{"2024-13-45", true, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDateTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if result.Year() != tt.year {
				t.Errorf("year: got %d, want %d", result.Year(), tt.year)
			}
			if result.Month() != tt.month {
				t.Errorf("month: got %v, want %v", result.Month(), tt.month)
			}
			if result.Day() != tt.day {
				t.Errorf("day: got %d, want %d", result.Day(), tt.day)
			}
		})
	}
}

func TestParseDateTimeRFC3339(t *testing.T) {
	input := "2024-03-15T10:30:00Z"
	result, err := parseDateTime(input)
	if err != nil {
		t.Fatalf("RFC3339 should parse: %v", err)
	}
	if result.Year() != 2024 || result.Month() != time.March || result.Day() != 15 {
		t.Errorf("unexpected date: %v", result)
	}
}

func TestFormatDateForAppleScript(t *testing.T) {
	dt := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	result := formatDateForAppleScript(dt)

	expected := "3/15/2024 14:30:45"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatDateForAppleScriptMidnight(t *testing.T) {
	dt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result := formatDateForAppleScript(dt)

	expected := "1/1/2024 00:00:00"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEscapeAppleScriptString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{`say "hello"`, `say \"hello\"`},
		{`path\to\file`, `path\\to\\file`},
		{"", ""},
		{"no special chars", "no special chars"},
	}

	for _, tt := range tests {
		got := escapeAppleScriptString(tt.input)
		if got != tt.want {
			t.Errorf("escapeAppleScriptString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseEventResults(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		count  int
		titles []string
	}{
		{
			name:  "empty input",
			input: "",
			count: 0,
		},
		{
			name:   "single event",
			input:  "Meeting|||Mon Mar 15 10:00|||Mon Mar 15 11:00|||Room A|||Notes here|||false|||Work",
			count:  1,
			titles: []string{"Meeting"},
		},
		{
			name:   "multiple events",
			input:  "Event1|||Start1|||End1|||Loc1|||Note1|||false|||Cal1, Event2|||Start2|||End2|||Loc2|||Note2|||true|||Cal2",
			count:  2,
			titles: []string{"Event1", "Event2"},
		},
		{
			name:  "malformed input (too few parts)",
			input: "bad|||data",
			count: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := parseEventResults(tt.input)
			if len(events) != tt.count {
				t.Errorf("expected %d events, got %d", tt.count, len(events))
			}
			for i, title := range tt.titles {
				if i < len(events) && events[i].Title != title {
					t.Errorf("event[%d].Title = %q, want %q", i, events[i].Title, title)
				}
			}
		})
	}
}

func TestParseEventResultsAllDay(t *testing.T) {
	input := "Holiday|||Dec 25 2024|||Dec 26 2024|||Home|||Xmas|||true|||Personal"
	events := parseEventResults(input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if !events[0].AllDay {
		t.Error("expected all-day event")
	}
	if events[0].Calendar != "Personal" {
		t.Errorf("expected calendar 'Personal', got %q", events[0].Calendar)
	}
	if events[0].Location != "Home" {
		t.Errorf("expected location 'Home', got %q", events[0].Location)
	}
}

func TestParseEventResultsWithEmptyFields(t *testing.T) {
	// 7 fields separated by 6 "|||" delimiters: title|||start|||end|||location|||notes|||allday|||calendar
	// 9 pipes between "end" and "allday" creates two empty fields (location + notes)
	input := "Meeting|||Mon 10:00|||Mon 11:00|||||||||false|||Work"
	events := parseEventResults(input)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Location != "" {
		t.Errorf("expected empty location, got %q", events[0].Location)
	}
	if events[0].Notes != "" {
		t.Errorf("expected empty notes, got %q", events[0].Notes)
	}
}
