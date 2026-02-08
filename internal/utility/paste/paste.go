package paste

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 30 * time.Second,
}

const apiURL = "https://dpaste.com/api/"

// PasteResult is LLM-friendly paste result
type PasteResult struct {
	URL       string `json:"url"`
	ExpiresIn string `json:"expires_in"`
	Title     string `json:"title,omitempty"`
}

// PasteContent is LLM-friendly paste content
type PasteContent struct {
	URL     string `json:"url"`
	Content string `json:"content"`
}

// NewCmd returns the paste command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "paste",
		Aliases: []string{"pb", "pastebin"},
		Short:   "Paste sharing commands (dpaste.com)",
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newGetCmd())

	return cmd
}

func newCreateCmd() *cobra.Command {
	var expiry int
	var title string

	cmd := &cobra.Command{
		Use:   "create [content]",
		Short: "Create a new paste (also accepts stdin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var content string

			if len(args) > 0 {
				content = strings.Join(args, " ")
			} else {
				// Try reading from stdin
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return output.PrintError("read_failed", "Failed to read stdin: "+err.Error(), nil)
					}
					content = string(data)
				}
			}

			if strings.TrimSpace(content) == "" {
				return output.PrintError("missing_content", "Content is required (pass as argument or pipe via stdin)", nil)
			}

			return createPaste(content, expiry, title)
		},
	}

	cmd.Flags().IntVarP(&expiry, "expiry", "e", 7, "Expiry in days")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Paste title")

	return cmd
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [url]",
		Short: "Fetch a paste by URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getPaste(args[0])
		},
	}

	return cmd
}

func createPaste(content string, expiryDays int, title string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	formData := url.Values{}
	formData.Set("content", content)
	formData.Set("expiry_days", fmt.Sprintf("%d", expiryDays))
	if title != "" {
		formData.Set("title", title)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return output.PrintError("create_failed", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body))), nil)
	}

	// dpaste returns the URL in the response body (or Location header)
	pasteURL := resp.Header.Get("Location")
	if pasteURL == "" {
		// Fall back to reading body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return output.PrintError("parse_failed", err.Error(), nil)
		}
		pasteURL = strings.TrimSpace(string(body))
	}

	result := PasteResult{
		URL:       pasteURL,
		ExpiresIn: fmt.Sprintf("%d days", expiryDays),
		Title:     title,
	}

	return output.Print(result)
}

func getPaste(pasteURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ensure we fetch raw content by appending .txt
	rawURL := strings.TrimSuffix(pasteURL, "/")
	if !strings.HasSuffix(rawURL, ".txt") {
		rawURL += ".txt"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	// Use a separate client that follows redirects for GET
	getClient := &http.Client{}
	resp, err := getClient.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return output.PrintError("not_found", "Paste not found: "+pasteURL, nil)
	}

	if resp.StatusCode >= 400 {
		return output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return output.PrintError("read_failed", err.Error(), nil)
	}

	result := PasteContent{
		URL:     pasteURL,
		Content: string(body),
	}

	return output.Print(result)
}
