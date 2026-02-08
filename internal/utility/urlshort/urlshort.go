package urlshort

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const isgdBaseURL = "https://is.gd/create.php"

var httpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// Don't follow redirects automatically for expand command
		return http.ErrUseLastResponse
	},
	Timeout: 30 * time.Second,
}

// ShortenResult is LLM-friendly shortened URL output
type ShortenResult struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
	Service     string `json:"service"`
}

// ExpandResult is LLM-friendly expanded URL output
type ExpandResult struct {
	ShortURL    string   `json:"short_url"`
	ExpandedURL string   `json:"expanded_url"`
	Hops        []string `json:"hops,omitempty"`
	FinalStatus int      `json:"final_status"`
}

// NewCmd returns the url shortener command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "url",
		Aliases: []string{"shorten", "short"},
		Short:   "URL shortener commands (is.gd)",
	}

	cmd.AddCommand(newShortenCmd())
	cmd.AddCommand(newExpandCmd())

	return cmd
}

func newShortenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shorten [url]",
		Short: "Shorten a URL using is.gd",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			originalURL := args[0]

			// Ensure URL has a scheme
			if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
				originalURL = "https://" + originalURL
			}

			// Validate URL
			if _, err := url.ParseRequestURI(originalURL); err != nil {
				return output.PrintError("invalid_url", "Invalid URL format: "+err.Error(), nil)
			}

			reqURL := fmt.Sprintf("%s?format=json&url=%s", isgdBaseURL, url.QueryEscape(originalURL))

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				ShortURL  string `json:"shorturl"`
				ErrorCode int    `json:"errorcode"`
				ErrorMsg  string `json:"errormessage"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if data.ErrorCode != 0 {
				return output.PrintError("shorten_failed", data.ErrorMsg, map[string]int{"error_code": data.ErrorCode})
			}

			if data.ShortURL == "" {
				return output.PrintError("shorten_failed", "No short URL returned", nil)
			}

			result := ShortenResult{
				OriginalURL: originalURL,
				ShortURL:    data.ShortURL,
				Service:     "is.gd",
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newExpandCmd() *cobra.Command {
	var maxHops int

	cmd := &cobra.Command{
		Use:   "expand [short-url]",
		Short: "Expand a shortened URL (follow redirects to get original)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shortURL := args[0]

			// Ensure URL has a scheme
			if !strings.HasPrefix(shortURL, "http://") && !strings.HasPrefix(shortURL, "https://") {
				shortURL = "https://" + shortURL
			}

			// Validate URL
			if _, err := url.ParseRequestURI(shortURL); err != nil {
				return output.PrintError("invalid_url", "Invalid URL format: "+err.Error(), nil)
			}

			var hops []string
			currentURL := shortURL
			finalStatus := 200

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Follow redirects manually to track each hop
			for i := 0; i < maxHops; i++ {
				req, err := http.NewRequestWithContext(ctx, "HEAD", currentURL, nil)
				if err != nil {
					return output.PrintError("fetch_failed", err.Error(), nil)
				}
				req.Header.Set("User-Agent", "Pocket-CLI/1.0")

				resp, err := httpClient.Do(req)
				if err != nil {
					return output.PrintError("fetch_failed", err.Error(), nil)
				}
				resp.Body.Close()

				finalStatus = resp.StatusCode

				// Check if it's a redirect
				if resp.StatusCode >= 300 && resp.StatusCode < 400 {
					location := resp.Header.Get("Location")
					if location == "" {
						break
					}

					// Handle relative redirects
					if !strings.HasPrefix(location, "http://") && !strings.HasPrefix(location, "https://") {
						baseURL, _ := url.Parse(currentURL)
						locURL, _ := url.Parse(location)
						location = baseURL.ResolveReference(locURL).String()
					}

					hops = append(hops, currentURL)
					currentURL = location
				} else {
					// Not a redirect, we've reached the final destination
					break
				}
			}

			result := ExpandResult{
				ShortURL:    shortURL,
				ExpandedURL: currentURL,
				FinalStatus: finalStatus,
			}

			if len(hops) > 0 {
				result.Hops = hops
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&maxHops, "max-hops", "m", 10, "Maximum number of redirects to follow")

	return cmd
}

func doRequest(reqURL string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a separate client for API requests that follows redirects
	apiClient := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		return nil, output.PrintError("rate_limited", "is.gd rate limit exceeded, try again later", nil)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return resp, nil
}
