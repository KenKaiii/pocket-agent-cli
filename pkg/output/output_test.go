package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintJSON(t *testing.T) {
	SetFormat("json")
	defer SetFormat("json")

	out := captureStdout(func() {
		Print(map[string]string{"key": "value"})
	})

	var resp Response
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Error != nil {
		t.Error("expected no error in response")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map data, got %T", resp.Data)
	}
	if data["key"] != "value" {
		t.Errorf("expected key=value, got key=%v", data["key"])
	}
}

func TestPrintText(t *testing.T) {
	SetFormat("text")
	defer SetFormat("json")

	out := captureStdout(func() {
		Print("hello world")
	})

	if strings.TrimSpace(out) != "hello world" {
		t.Errorf("expected 'hello world', got %q", strings.TrimSpace(out))
	}
}

func TestPrintTextMap(t *testing.T) {
	SetFormat("text")
	defer SetFormat("json")

	out := captureStdout(func() {
		Print(map[string]string{"name": "test"})
	})

	if !strings.Contains(out, "name: test") {
		t.Errorf("expected 'name: test' in output, got %q", out)
	}
}

func TestPrintErrorJSON(t *testing.T) {
	SetFormat("json")
	defer SetFormat("json")

	out := captureStdout(func() {
		PrintError("test_code", "test message", map[string]string{"detail": "info"})
	})

	var resp Response
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != "test_code" {
		t.Errorf("expected code=test_code, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "test message" {
		t.Errorf("expected message='test message', got %s", resp.Error.Message)
	}
}

func TestPrintErrorReturnsPrintedError(t *testing.T) {
	SetFormat("json")
	defer SetFormat("json")

	var err error
	captureStdout(func() {
		err = PrintError("code", "msg", nil)
	})

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	var pe *PrintedError
	if !errors.As(err, &pe) {
		t.Error("expected PrintedError type")
	}
}

func TestIsPrinted(t *testing.T) {
	pe := &PrintedError{Err: errors.New("test")}
	if !IsPrinted(pe) {
		t.Error("expected IsPrinted to return true for PrintedError")
	}

	regular := errors.New("regular error")
	if IsPrinted(regular) {
		t.Error("expected IsPrinted to return false for regular error")
	}

	if IsPrinted(nil) {
		t.Error("expected IsPrinted to return false for nil")
	}
}

func TestPrintedErrorUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	pe := &PrintedError{Err: inner}

	if pe.Error() != "inner error" {
		t.Errorf("expected 'inner error', got %q", pe.Error())
	}

	if !errors.Is(pe, inner) {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestSetFormat(t *testing.T) {
	original := format
	defer func() { format = original }()

	SetFormat("text")
	if format != "text" {
		t.Errorf("expected format=text, got %s", format)
	}

	SetFormat("table")
	if format != "table" {
		t.Errorf("expected format=table, got %s", format)
	}

	SetFormat("json")
	if format != "json" {
		t.Errorf("expected format=json, got %s", format)
	}
}

func TestSetVerbose(t *testing.T) {
	original := verbose
	defer func() { verbose = original }()

	SetVerbose(true)
	if !verbose {
		t.Error("expected verbose=true")
	}

	SetVerbose(false)
	if verbose {
		t.Error("expected verbose=false")
	}
}

func TestPrintJSONVerboseIndent(t *testing.T) {
	SetFormat("json")
	SetVerbose(true)
	defer func() {
		SetFormat("json")
		SetVerbose(false)
	}()

	out := captureStdout(func() {
		Print(map[string]string{"key": "val"})
	})

	// Verbose mode should produce indented JSON (multi-line)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Error("expected indented (multi-line) JSON in verbose mode")
	}
}

func TestPrintNilData(t *testing.T) {
	SetFormat("json")
	defer SetFormat("json")

	out := captureStdout(func() {
		Print(nil)
	})

	var resp Response
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true even for nil data")
	}
}

func TestPrintErrorNilDetails(t *testing.T) {
	SetFormat("json")
	defer SetFormat("json")

	out := captureStdout(func() {
		PrintError("err", "message", nil)
	})

	var resp Response
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if resp.Error.Details != nil {
		t.Errorf("expected nil details, got %v", resp.Error.Details)
	}
}

func TestPrintTableEmpty(t *testing.T) {
	SetFormat("table")
	defer SetFormat("json")

	out := captureStdout(func() {
		Print([]map[string]any{})
	})

	if out != "" {
		t.Errorf("expected empty output for empty table, got %q", out)
	}
}

func TestPrintDefaultFallsBackToJSON(t *testing.T) {
	SetFormat("unknown_format")
	defer SetFormat("json")

	out := captureStdout(func() {
		Print(map[string]string{"k": "v"})
	})

	var resp Response
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("unknown format should fallback to JSON: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}
