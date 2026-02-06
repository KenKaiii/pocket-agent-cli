package wikipedia

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

const baseURL = "https://en.wikipedia.org/w/api.php"

var httpClient = &http.Client{}

// Article is LLM-friendly article output
type Article struct {
	Title   string `json:"title"`
	Extract string `json:"extract"`
	URL     string `json:"url"`
}

// SearchResult is LLM-friendly search result
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wikipedia",
		Aliases: []string{"wiki", "wp"},
		Short:   "Wikipedia commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newSummaryCmd())
	cmd.AddCommand(newArticleCmd())

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Wikipedia",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			params := url.Values{
				"action":   {"query"},
				"list":     {"search"},
				"srsearch": {query},
				"srlimit":  {fmt.Sprintf("%d", limit)},
				"format":   {"json"},
			}

			var resp struct {
				Query struct {
					Search []struct {
						Title   string `json:"title"`
						Snippet string `json:"snippet"`
						PageID  int    `json:"pageid"`
					} `json:"search"`
				} `json:"query"`
			}

			if err := wikiGet(params, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			results := make([]SearchResult, 0, len(resp.Query.Search))
			for _, r := range resp.Query.Search {
				results = append(results, SearchResult{
					Title:   r.Title,
					Snippet: cleanSnippet(r.Snippet),
					URL:     fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.PathEscape(strings.ReplaceAll(r.Title, " ", "_"))),
				})
			}

			return output.Print(results)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 5, "Number of results")

	return cmd
}

func newSummaryCmd() *cobra.Command {
	var sentences int

	cmd := &cobra.Command{
		Use:   "summary [title]",
		Short: "Get article summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			params := url.Values{
				"action":      {"query"},
				"titles":      {title},
				"prop":        {"extracts"},
				"exintro":     {"true"},
				"explaintext": {"true"},
				"exsentences": {fmt.Sprintf("%d", sentences)},
				"redirects":   {"1"},
				"format":      {"json"},
			}

			var resp struct {
				Query struct {
					Pages map[string]struct {
						PageID  int    `json:"pageid"`
						Title   string `json:"title"`
						Extract string `json:"extract"`
					} `json:"pages"`
				} `json:"query"`
			}

			if err := wikiGet(params, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			for _, page := range resp.Query.Pages {
				if page.PageID == 0 {
					return output.PrintError("not_found", "Article not found: "+title, nil)
				}

				return output.Print(Article{
					Title:   page.Title,
					Extract: page.Extract,
					URL:     fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.PathEscape(strings.ReplaceAll(page.Title, " ", "_"))),
				})
			}

			return output.PrintError("not_found", "Article not found: "+title, nil)
		},
	}

	cmd.Flags().IntVarP(&sentences, "sentences", "s", 5, "Number of sentences")

	return cmd
}

func newArticleCmd() *cobra.Command {
	var maxChars int

	cmd := &cobra.Command{
		Use:   "article [title]",
		Short: "Get full article extract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			params := url.Values{
				"action":      {"query"},
				"titles":      {title},
				"prop":        {"extracts"},
				"explaintext": {"true"},
				"exchars":     {fmt.Sprintf("%d", maxChars)},
				"redirects":   {"1"},
				"format":      {"json"},
			}

			var resp struct {
				Query struct {
					Pages map[string]struct {
						PageID  int    `json:"pageid"`
						Title   string `json:"title"`
						Extract string `json:"extract"`
					} `json:"pages"`
				} `json:"query"`
			}

			if err := wikiGet(params, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			for _, page := range resp.Query.Pages {
				if page.PageID == 0 {
					return output.PrintError("not_found", "Article not found: "+title, nil)
				}

				return output.Print(Article{
					Title:   page.Title,
					Extract: page.Extract,
					URL:     fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.PathEscape(strings.ReplaceAll(page.Title, " ", "_"))),
				})
			}

			return output.PrintError("not_found", "Article not found: "+title, nil)
		},
	}

	cmd.Flags().IntVarP(&maxChars, "chars", "c", 2000, "Maximum characters")

	return cmd
}

func wikiGet(params url.Values, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0 (https://github.com/unstablemind/pocket)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func cleanSnippet(s string) string {
	// Remove HTML tags from search snippets
	s = strings.ReplaceAll(s, "<span class=\"searchmatch\">", "")
	s = strings.ReplaceAll(s, "</span>", "")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}
