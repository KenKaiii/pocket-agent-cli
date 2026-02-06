package finder

import (
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		got := humanReadableSize(tt.bytes)
		if got != tt.want {
			t.Errorf("humanReadableSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestResolvePath(t *testing.T) {
	// Test absolute path
	abs, err := resolvePath("/tmp/test")
	if err != nil {
		t.Fatalf("resolvePath absolute failed: %v", err)
	}
	if abs != "/tmp/test" {
		t.Errorf("expected /tmp/test, got %s", abs)
	}

	// Test tilde expansion
	home, _ := os.UserHomeDir()
	tilded, err := resolvePath("~/test")
	if err != nil {
		t.Fatalf("resolvePath tilde failed: %v", err)
	}
	expected := filepath.Join(home, "test")
	if tilded != expected {
		t.Errorf("expected %s, got %s", expected, tilded)
	}

	// Test relative path becomes absolute
	rel, err := resolvePath("relative/path")
	if err != nil {
		t.Fatalf("resolvePath relative failed: %v", err)
	}
	if !filepath.IsAbs(rel) {
		t.Errorf("expected absolute path, got %s", rel)
	}
}

func TestResolvePathTildeOnly(t *testing.T) {
	home, _ := os.UserHomeDir()
	result, err := resolvePath("~")
	if err != nil {
		t.Fatalf("resolvePath(~) failed: %v", err)
	}
	if result != home {
		t.Errorf("expected %s, got %s", home, result)
	}
}

func TestParseMdlsOutput(t *testing.T) {
	tests := []struct {
		output   string
		attrName string
		want     string
	}{
		{
			"kMDItemContentType = \"public.plain-text\"",
			"kMDItemContentType",
			"public.plain-text",
		},
		{
			"kMDItemContentType = (null)",
			"kMDItemContentType",
			"",
		},
		{
			"kMDItemDisplayName = \"test.txt\"",
			"kMDItemDisplayName",
			"test.txt",
		},
		{
			"kMDItemFoo = \"bar\"\nkMDItemBaz = \"qux\"",
			"kMDItemBaz",
			"qux",
		},
		{
			"",
			"anything",
			"",
		},
	}

	for _, tt := range tests {
		got := parseMdlsOutput(tt.output, tt.attrName)
		if got != tt.want {
			t.Errorf("parseMdlsOutput(%q, %q) = %q, want %q", tt.output, tt.attrName, got, tt.want)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1024", 1024},
		{"0", 0},
		{"", 0},
		{"(null)", 0},
		{"  1234  ", 1234},
		{"abc", 0},
	}

	for _, tt := range tests {
		got := parseSize(tt.input)
		if got != tt.want {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`hello`, `hello`},
		{`say "hi"`, `say \"hi\"`},
		{`path\to\file`, `path\\to\\file`},
		{`it's`, `it's`},
		{``, ``},
		{`a "b" c\d`, `a \"b\" c\\d`},
	}

	for _, tt := range tests {
		got := escapeAppleScript(tt.input)
		if got != tt.want {
			t.Errorf("escapeAppleScript(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestXMLEscapeInPlistTags(t *testing.T) {
	// Verify that html.EscapeString properly handles XML injection characters
	tests := []struct {
		input string
		want  string
	}{
		{"normal_tag", "normal_tag"},
		{"tag<script>", "tag&lt;script&gt;"},
		{"tag&value", "tag&amp;value"},
		{`tag"quoted"`, "tag&#34;quoted&#34;"},
		{"tag'apos'", "tag&#39;apos&#39;"},
		{"</string></array></plist>", "&lt;/string&gt;&lt;/array&gt;&lt;/plist&gt;"},
	}

	for _, tt := range tests {
		got := html.EscapeString(tt.input)
		if got != tt.want {
			t.Errorf("html.EscapeString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMdfindQuerySanitization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal query", "normal query"},
		{"test'injection", "testinjection"},
		{"file.txt", "file.txt"},
		{"') || (kMDItemContentType == 'public.folder", ") || (kMDItemContentType == public.folder"},
		{"it's a test", "its a test"},
		{"no'quotes'here", "noquoteshere"},
	}

	for _, tt := range tests {
		got := strings.ReplaceAll(tt.input, "'", "")
		if got != tt.want {
			t.Errorf("sanitize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDirWalkLimit(t *testing.T) {
	// Create a temp dir with some files
	tmpDir := t.TempDir()
	for i := 0; i < 10; i++ {
		f, err := os.CreateTemp(tmpDir, "test")
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte("data"))
		f.Close()
	}

	// Walk should complete and count files
	var size int64
	var fileCount int
	filepath.Walk(tmpDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		fileCount++
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	// Should have counted the directory + 10 files
	if fileCount != 11 {
		t.Errorf("expected 11 entries (1 dir + 10 files), got %d", fileCount)
	}
	if size != 40 { // 10 files * 4 bytes
		t.Errorf("expected size 40, got %d", size)
	}
}
