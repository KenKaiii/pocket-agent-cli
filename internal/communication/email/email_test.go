package email

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name string
		addr *imap.Address
		want string
	}{
		{
			name: "nil address",
			addr: nil,
			want: "",
		},
		{
			name: "with personal name",
			addr: &imap.Address{
				PersonalName: "John Doe",
				MailboxName:  "john",
				HostName:     "example.com",
			},
			want: "John Doe <john@example.com>",
		},
		{
			name: "without personal name",
			addr: &imap.Address{
				MailboxName: "jane",
				HostName:    "example.com",
			},
			want: "jane@example.com",
		},
		{
			name: "empty personal name",
			addr: &imap.Address{
				PersonalName: "",
				MailboxName:  "user",
				HostName:     "test.com",
			},
			want: "user@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAddress(tt.addr)
			if got != tt.want {
				t.Errorf("formatAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatEmailOnly(t *testing.T) {
	tests := []struct {
		name string
		addr *imap.Address
		want string
	}{
		{
			name: "nil address",
			addr: nil,
			want: "",
		},
		{
			name: "valid address",
			addr: &imap.Address{
				PersonalName: "John Doe",
				MailboxName:  "john",
				HostName:     "example.com",
			},
			want: "john@example.com",
		},
		{
			name: "empty mailbox",
			addr: &imap.Address{
				MailboxName: "",
				HostName:    "example.com",
			},
			want: "",
		},
		{
			name: "empty host",
			addr: &imap.Address{
				MailboxName: "john",
				HostName:    "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatEmailOnly(tt.addr)
			if got != tt.want {
				t.Errorf("formatEmailOnly() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		contains string
	}{
		{
			name:     "zero time",
			input:    time.Time{},
			contains: "",
		},
		{
			name:     "today shows time only",
			input:    time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, now.Location()),
			contains: "14:30",
		},
		{
			name:     "this year shows month and day",
			input:    time.Date(now.Year(), time.January, 15, 10, 0, 0, 0, now.Location()),
			contains: "Jan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.input)
			if tt.input.IsZero() {
				if got != "" {
					t.Errorf("expected empty for zero time, got %q", got)
				}
				return
			}
			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("formatTime(%v) = %q, expected to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestFormatTimeOldDate(t *testing.T) {
	old := time.Date(2020, time.June, 15, 12, 0, 0, 0, time.UTC)
	result := formatTime(old)

	if !strings.Contains(result, "2020") {
		t.Errorf("old date should include year, got %q", result)
	}
	if !strings.Contains(result, "Jun") {
		t.Errorf("old date should include month, got %q", result)
	}
}

func TestFormatTimeRecent(t *testing.T) {
	// 3 days ago should show weekday
	recent := time.Now().Add(-3 * 24 * time.Hour)
	result := formatTime(recent)

	// Should contain a day abbreviation
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	found := false
	for _, d := range days {
		if strings.Contains(result, d) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("recent date should show weekday, got %q", result)
	}
}

func TestDecodeHeader(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plain text", "plain text"},
		{"", ""},
		{"=?UTF-8?B?SGVsbG8gV29ybGQ=?=", "Hello World"},
		{"=?UTF-8?Q?Hello_World?=", "Hello World"},
		{"no encoding needed", "no encoding needed"},
	}

	for _, tt := range tests {
		got := decodeHeader(tt.input)
		if got != tt.want {
			t.Errorf("decodeHeader(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<b>Bold</b> and <i>italic</i>", "Bold and italic"},
		{"No HTML here", "No HTML here"},
		{"<div>Line 1</div><div>Line 2</div>", "Line 1Line 2"},
		{"&amp; &lt; &gt; &quot; &#39;", "& < > \" '"},
		{"&nbsp;space&nbsp;", " space "},
		{"", ""},
	}

	for _, tt := range tests {
		got := stripHTML(tt.input)
		// Normalize whitespace for comparison
		got = strings.TrimSpace(got)
		want := strings.TrimSpace(tt.want)
		if got != want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, want)
		}
	}
}

func TestCleanBody(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normalize line endings",
			input: "line1\r\nline2\rline3\nline4",
			want:  "line1\nline2\nline3\nline4",
		},
		{
			name:  "collapse excessive blank lines",
			input: "line1\n\n\n\n\nline2",
			want:  "line1\n\n\nline2",
		},
		{
			name:  "trim whitespace",
			input: "  \n  hello world  \n  ",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only lines collapsed",
			input: "hello\n   \n   \n   \n   \nworld",
			want:  "hello\n\n\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanBody(tt.input)
			if got != tt.want {
				t.Errorf("cleanBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeBody(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		encoding string
		want     string
	}{
		{
			name:     "no encoding",
			body:     []byte("plain text"),
			encoding: "",
			want:     "plain text",
		},
		{
			name:     "7bit",
			body:     []byte("7bit text"),
			encoding: "7bit",
			want:     "7bit text",
		},
		{
			name:     "quoted-printable",
			body:     []byte("Hello=20World"),
			encoding: "quoted-printable",
			want:     "Hello World",
		},
		{
			name:     "quoted-printable case insensitive",
			body:     []byte("Hello=20World"),
			encoding: "Quoted-Printable",
			want:     "Hello World",
		},
		{
			name:     "base64 passthrough",
			body:     []byte("base64content"),
			encoding: "base64",
			want:     "base64content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeBody(tt.body, tt.encoding)
			if got != tt.want {
				t.Errorf("decodeBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTextBodyPlain(t *testing.T) {
	raw := "Content-Type: text/plain\r\n\r\nHello World"
	got := extractTextBody([]byte(raw))
	if !strings.Contains(got, "Hello World") {
		t.Errorf("expected 'Hello World' in body, got %q", got)
	}
}

func TestExtractTextBodyHTML(t *testing.T) {
	raw := "Content-Type: text/html\r\n\r\n<html><body><p>Hello HTML</p></body></html>"
	got := extractTextBody([]byte(raw))
	if !strings.Contains(got, "Hello HTML") {
		t.Errorf("expected 'Hello HTML' in body, got %q", got)
	}
}

func TestExtractTextBodyMultipart(t *testing.T) {
	raw := "Content-Type: multipart/alternative; boundary=boundary123\r\n\r\n" +
		"--boundary123\r\n" +
		"Content-Type: text/plain\r\n\r\n" +
		"Plain text version\r\n" +
		"--boundary123\r\n" +
		"Content-Type: text/html\r\n\r\n" +
		"<p>HTML version</p>\r\n" +
		"--boundary123--\r\n"

	got := extractTextBody([]byte(raw))
	if !strings.Contains(got, "Plain text version") {
		t.Errorf("expected plain text part, got %q", got)
	}
}

func TestExtractTextBodyFallback(t *testing.T) {
	// Non-parseable input should fall back to cleaning
	raw := "Just raw text without headers"
	got := extractTextBody([]byte(raw))
	if got == "" {
		t.Error("expected non-empty fallback result")
	}
}

func TestExtractMultipartEmpty(t *testing.T) {
	result := extractMultipart(strings.NewReader(""), "")
	if result != "" {
		t.Errorf("expected empty for empty boundary, got %q", result)
	}
}
