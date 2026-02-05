package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
)

var (
	format  = "json"
	verbose = false
)

// PrintedError wraps an error that has already been printed
type PrintedError struct {
	Err error
}

func (e *PrintedError) Error() string {
	return e.Err.Error()
}

func (e *PrintedError) Unwrap() error {
	return e.Err
}

// IsPrinted checks if an error has already been printed
func IsPrinted(err error) bool {
	var pe *PrintedError
	return errors.As(err, &pe)
}

// SetFormat sets the global output format
func SetFormat(f string) {
	format = f
}

// SetVerbose sets verbose mode
func SetVerbose(v bool) {
	verbose = v
}

// Response is the standard response structure
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents an error response
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Print outputs data in the configured format
func Print(data any) error {
	switch format {
	case "json":
		return printJSON(Response{Success: true, Data: data})
	case "text":
		return printText(data)
	case "table":
		return printTable(data)
	default:
		return printJSON(Response{Success: true, Data: data})
	}
}

// PrintError outputs an error in the configured format and returns a PrintedError
func PrintError(code, message string, details any) error {
	resp := Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	switch format {
	case "json":
		printJSON(resp)
	default:
		fmt.Fprintf(os.Stderr, "Error [%s]: %s\n", code, message)
	}

	// Return a PrintedError so callers know it's already been output
	return &PrintedError{Err: fmt.Errorf("%s: %s", code, message)}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	if verbose {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func printText(data any) error {
	switch v := data.(type) {
	case string:
		fmt.Println(v)
	case map[string]string:
		for k, val := range v {
			fmt.Printf("%s: %s\n", k, val)
		}
	case map[string]any:
		for k, val := range v {
			fmt.Printf("%s: %v\n", k, val)
		}
	default:
		// Fall back to JSON for complex types
		return printJSON(data)
	}
	return nil
}

func printTable(data any) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	switch v := data.(type) {
	case []map[string]any:
		if len(v) == 0 {
			return nil
		}
		// Print headers from first item
		for k := range v[0] {
			fmt.Fprintf(w, "%s\t", k)
		}
		fmt.Fprintln(w)
		// Print rows
		for _, row := range v {
			for _, val := range row {
				fmt.Fprintf(w, "%v\t", val)
			}
			fmt.Fprintln(w)
		}
	default:
		return printJSON(data)
	}
	return nil
}

// Verbose prints only in verbose mode
func Verbose(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}
