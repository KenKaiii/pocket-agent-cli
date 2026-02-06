package newsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const apiBaseURL = "https://newsapi.org/v2"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "newsapi",
		Aliases: []string{"news-api"},
		Short:   "NewsAPI commands",
		Long:    "NewsAPI integration for headlines and article search from 80,000+ sources.",
	}

	cmd.AddCommand(newHeadlinesCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newSourcesCmd())

	return cmd
}

type newsClient struct {
	apiKey     string
	httpClient *http.Client
}

func newNewsClient() (*newsClient, error) {
	apiKey, err := config.MustGet("newsapi_key")
	if err != nil {
		return nil, err
	}

	return &newsClient{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}, nil
}

func (c *newsClient) doRequest(endpoint string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiBaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Status  string `json:"status"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("NewsAPI error: %s", errResp.Message)
		}
		return nil, fmt.Errorf("NewsAPI error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

type article struct {
	Source struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"source"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	ImageURL    string `json:"urlToImage"`
	PublishedAt string `json:"publishedAt"`
	Content     string `json:"content"`
}

func formatArticles(articles []article) []map[string]any {
	result := make([]map[string]any, len(articles))
	for i, a := range articles {
		result[i] = map[string]any{
			"source":       a.Source.Name,
			"author":       a.Author,
			"title":        a.Title,
			"description":  a.Description,
			"url":          a.URL,
			"image":        a.ImageURL,
			"published_at": a.PublishedAt,
		}
	}
	return result
}

func newHeadlinesCmd() *cobra.Command {
	var country string
	var category string
	var limit int
	var query string

	cmd := &cobra.Command{
		Use:   "headlines",
		Short: "Get top headlines",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newNewsClient()
			if err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/top-headlines?pageSize=%d", limit)
			if country != "" {
				endpoint += "&country=" + country
			}
			if category != "" {
				endpoint += "&category=" + category
			}
			if query != "" {
				endpoint += "&q=" + url.QueryEscape(query)
			}

			body, err := client.doRequest(endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Status       string    `json:"status"`
				TotalResults int       `json:"totalResults"`
				Articles     []article `json:"articles"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"total":    result.TotalResults,
				"count":    len(result.Articles),
				"articles": formatArticles(result.Articles),
			})
		},
	}

	cmd.Flags().StringVar(&country, "country", "us", "Country code (us, gb, de, fr, etc.)")
	cmd.Flags().StringVar(&category, "category", "", "Category: business, entertainment, health, science, sports, technology")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of articles (max 100)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Keywords to search in headlines")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var sortBy string
	var limit int
	var language string
	var from string
	var to string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search news articles",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newNewsClient()
			if err != nil {
				return err
			}

			query := url.QueryEscape(args[0])
			endpoint := fmt.Sprintf("/everything?q=%s&pageSize=%d&sortBy=%s", query, limit, sortBy)

			if language != "" {
				endpoint += "&language=" + language
			}
			if from != "" {
				endpoint += "&from=" + from
			}
			if to != "" {
				endpoint += "&to=" + to
			}

			body, err := client.doRequest(endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Status       string    `json:"status"`
				TotalResults int       `json:"totalResults"`
				Articles     []article `json:"articles"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"query":    args[0],
				"total":    result.TotalResults,
				"count":    len(result.Articles),
				"articles": formatArticles(result.Articles),
			})
		},
	}

	cmd.Flags().StringVar(&sortBy, "sort", "publishedAt", "Sort: relevancy, popularity, publishedAt")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of articles (max 100)")
	cmd.Flags().StringVar(&language, "lang", "", "Language code (en, de, fr, es, etc.)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD)")

	return cmd
}

func newSourcesCmd() *cobra.Command {
	var category string
	var country string
	var language string

	cmd := &cobra.Command{
		Use:   "sources",
		Short: "List news sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newNewsClient()
			if err != nil {
				return err
			}

			endpoint := "/top-headlines/sources?"
			params := []string{}

			if category != "" {
				params = append(params, "category="+category)
			}
			if country != "" {
				params = append(params, "country="+country)
			}
			if language != "" {
				params = append(params, "language="+language)
			}

			for i, p := range params {
				if i > 0 {
					endpoint += "&"
				}
				endpoint += p
			}

			body, err := client.doRequest(endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Status  string `json:"status"`
				Sources []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
					URL         string `json:"url"`
					Category    string `json:"category"`
					Language    string `json:"language"`
					Country     string `json:"country"`
				} `json:"sources"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			sources := make([]map[string]any, len(result.Sources))
			for i, s := range result.Sources {
				sources[i] = map[string]any{
					"id":          s.ID,
					"name":        s.Name,
					"description": s.Description,
					"url":         s.URL,
					"category":    s.Category,
					"language":    s.Language,
					"country":     s.Country,
				}
			}

			return output.Print(map[string]any{
				"count":   len(sources),
				"sources": sources,
			})
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Category filter")
	cmd.Flags().StringVar(&country, "country", "", "Country filter")
	cmd.Flags().StringVar(&language, "lang", "", "Language filter")

	return cmd
}
