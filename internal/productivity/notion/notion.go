package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const apiBaseURL = "https://api.notion.com/v1"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notion",
		Short: "Notion commands",
		Long:  "Notion workspace integration for pages and databases.",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newPageCmd())
	cmd.AddCommand(newDatabaseCmd())
	cmd.AddCommand(newBlocksCmd())

	return cmd
}

type notionClient struct {
	token      string
	httpClient *http.Client
}

func newNotionClient() (*notionClient, error) {
	token, err := config.MustGet("notion_token")
	if err != nil {
		return nil, err
	}

	return &notionClient{
		token:      token,
		httpClient: &http.Client{},
	}, nil
}

func (c *notionClient) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiBaseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", "2022-06-28") // Pinned: 2025-09-03 has breaking database changes
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("Notion API error: %s", errResp.Message)
		}
		return nil, fmt.Errorf("Notion API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func extractTitle(props map[string]any) string {
	// Try "title" property first (pages)
	if title, ok := props["title"]; ok {
		if arr, ok := title.([]interface{}); ok && len(arr) > 0 {
			if item, ok := arr[0].(map[string]any); ok {
				if text, ok := item["plain_text"].(string); ok {
					return text
				}
			}
		}
	}

	// Try "Name" property (database items)
	if name, ok := props["Name"]; ok {
		if nameObj, ok := name.(map[string]any); ok {
			if titleArr, ok := nameObj["title"].([]interface{}); ok && len(titleArr) > 0 {
				if item, ok := titleArr[0].(map[string]any); ok {
					if text, ok := item["plain_text"].(string); ok {
						return text
					}
				}
			}
		}
	}

	return ""
}

func newSearchCmd() *cobra.Command {
	var limit int
	var filterType string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Notion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newNotionClient()
			if err != nil {
				return err
			}

			payload := map[string]any{
				"query":     args[0],
				"page_size": limit,
			}

			if filterType != "" && filterType != "all" {
				payload["filter"] = map[string]string{
					"property": "object",
					"value":    filterType,
				}
			}

			body, err := client.doRequest("POST", "/search", payload)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Results []struct {
					ID         string         `json:"id"`
					Object     string         `json:"object"`
					URL        string         `json:"url"`
					Properties map[string]any `json:"properties"`
					Title      []struct {
						PlainText string `json:"plain_text"`
					} `json:"title,omitempty"`
				} `json:"results"`
				HasMore bool `json:"has_more"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			items := make([]map[string]any, len(result.Results))
			for i, r := range result.Results {
				title := ""
				if len(r.Title) > 0 {
					title = r.Title[0].PlainText
				} else {
					title = extractTitle(r.Properties)
				}

				items[i] = map[string]any{
					"id":    r.ID,
					"type":  r.Object,
					"title": title,
					"url":   r.URL,
				}
			}

			return output.Print(map[string]any{
				"query":    args[0],
				"count":    len(items),
				"has_more": result.HasMore,
				"results":  items,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")
	cmd.Flags().StringVarP(&filterType, "type", "t", "all", "Filter: page, database, all")

	return cmd
}

func newPageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page [page-id]",
		Short: "Get page content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newNotionClient()
			if err != nil {
				return err
			}

			// Get page metadata
			pageBody, err := client.doRequest("GET", "/pages/"+args[0], nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var page struct {
				ID         string         `json:"id"`
				URL        string         `json:"url"`
				CreatedBy  map[string]any `json:"created_by"`
				Properties map[string]any `json:"properties"`
			}

			if err := json.Unmarshal(pageBody, &page); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			// Get page blocks (content)
			blocksBody, err := client.doRequest("GET", "/blocks/"+args[0]+"/children?page_size=100", nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var blocks struct {
				Results []map[string]any `json:"results"`
			}

			if err := json.Unmarshal(blocksBody, &blocks); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			// Extract text content from blocks
			content := make([]map[string]any, 0)
			for _, block := range blocks.Results {
				blockType, _ := block["type"].(string)
				item := map[string]any{
					"type": blockType,
				}

				// Try to extract text from the block type's rich_text
				if typeData, ok := block[blockType].(map[string]any); ok {
					if richText, ok := typeData["rich_text"].([]interface{}); ok {
						var text string
						for _, rt := range richText {
							if rtMap, ok := rt.(map[string]any); ok {
								if pt, ok := rtMap["plain_text"].(string); ok {
									text += pt
								}
							}
						}
						if text != "" {
							item["text"] = text
						}
					}
				}

				content = append(content, item)
			}

			return output.Print(map[string]any{
				"id":      page.ID,
				"url":     page.URL,
				"title":   extractTitle(page.Properties),
				"content": content,
			})
		},
	}

	return cmd
}

func newDatabaseCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "database [database-id]",
		Aliases: []string{"db"},
		Short:   "Query a database",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newNotionClient()
			if err != nil {
				return err
			}

			payload := map[string]any{
				"page_size": limit,
			}

			body, err := client.doRequest("POST", "/databases/"+args[0]+"/query", payload)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Results []struct {
					ID         string         `json:"id"`
					URL        string         `json:"url"`
					Properties map[string]any `json:"properties"`
				} `json:"results"`
				HasMore bool `json:"has_more"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			items := make([]map[string]any, len(result.Results))
			for i, r := range result.Results {
				items[i] = map[string]any{
					"id":    r.ID,
					"url":   r.URL,
					"title": extractTitle(r.Properties),
				}
			}

			return output.Print(map[string]any{
				"database_id": args[0],
				"count":       len(items),
				"has_more":    result.HasMore,
				"items":       items,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Number of results")

	return cmd
}

func newBlocksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocks [block-id]",
		Short: "Get block children",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newNotionClient()
			if err != nil {
				return err
			}

			body, err := client.doRequest("GET", "/blocks/"+args[0]+"/children?page_size=100", nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Results []map[string]any `json:"results"`
				HasMore bool             `json:"has_more"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"block_id": args[0],
				"count":    len(result.Results),
				"has_more": result.HasMore,
				"blocks":   result.Results,
			})
		},
	}

	return cmd
}
