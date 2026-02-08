package gist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const baseURL = "https://api.github.com"

// GistSummary is LLM-friendly gist list item
type GistSummary struct {
	ID          string   `json:"id"`
	Description string   `json:"description,omitempty"`
	Public      bool     `json:"public"`
	Files       []string `json:"files"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	URL         string   `json:"url"`
}

// GistDetail is LLM-friendly gist detail
type GistDetail struct {
	ID          string     `json:"id"`
	Description string     `json:"description,omitempty"`
	Public      bool       `json:"public"`
	Files       []GistFile `json:"files"`
	CreatedAt   string     `json:"created_at"`
	URL         string     `json:"url"`
}

// GistFile is LLM-friendly gist file
type GistFile struct {
	Filename string `json:"filename"`
	Language string `json:"language,omitempty"`
	Size     int    `json:"size"`
	Content  string `json:"content"`
}

// GistCreated is the result of creating a gist
type GistCreated struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Public      bool   `json:"public"`
}

// NewCmd returns the gist command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gist",
		Aliases: []string{"gists"},
		Short:   "GitHub Gist commands",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your gists",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			reqURL := fmt.Sprintf("%s/gists?per_page=%d", baseURL, limit)

			var data []map[string]any
			if err := ghGet(token, reqURL, &data); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			results := make([]GistSummary, 0, len(data))
			for _, g := range data {
				results = append(results, toGistSummary(g))
			}

			return output.Print(results)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of gists to return")

	return cmd
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Get gist details by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			reqURL := fmt.Sprintf("%s/gists/%s", baseURL, args[0])

			var data map[string]any
			if err := ghGet(token, reqURL, &data); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			return output.Print(toGistDetail(data))
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var desc string
	var filename string
	var public bool

	cmd := &cobra.Command{
		Use:   "create [content]",
		Short: "Create a new gist (also accepts stdin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

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

			return createGist(token, content, desc, filename, public)
		},
	}

	cmd.Flags().StringVarP(&desc, "desc", "d", "", "Gist description")
	cmd.Flags().StringVarP(&filename, "filename", "f", "file.txt", "Filename for the gist")
	cmd.Flags().BoolVar(&public, "public", false, "Make gist public")

	return cmd
}

func createGist(token, content, desc, filename string, public bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]any{
		"description": desc,
		"public":      public,
		"files": map[string]any{
			filename: map[string]string{
				"content": content,
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return output.PrintError("encode_failed", err.Error(), nil)
	}

	reqURL := fmt.Sprintf("%s/gists", baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			if msg, ok := errResp["message"].(string); ok && msg != "" {
				return output.PrintError("create_failed", msg, nil)
			}
		}
		return output.PrintError("create_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return output.PrintError("parse_failed", err.Error(), nil)
	}

	result := GistCreated{
		ID:          getString(data, "id"),
		URL:         getString(data, "html_url"),
		Description: getString(data, "description"),
		Public:      getBool(data, "public"),
	}

	return output.Print(result)
}

func ghGet(token, reqURL string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			if msg, ok := errResp["message"].(string); ok && msg != "" {
				return fmt.Errorf("%s", msg)
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func toGistSummary(g map[string]any) GistSummary {
	summary := GistSummary{
		ID:          getString(g, "id"),
		Description: getString(g, "description"),
		Public:      getBool(g, "public"),
		CreatedAt:   getString(g, "created_at"),
		UpdatedAt:   getString(g, "updated_at"),
		URL:         getString(g, "html_url"),
	}

	if files, ok := g["files"].(map[string]any); ok {
		for name := range files {
			summary.Files = append(summary.Files, name)
		}
	}

	return summary
}

func toGistDetail(g map[string]any) GistDetail {
	detail := GistDetail{
		ID:          getString(g, "id"),
		Description: getString(g, "description"),
		Public:      getBool(g, "public"),
		CreatedAt:   getString(g, "created_at"),
		URL:         getString(g, "html_url"),
	}

	if files, ok := g["files"].(map[string]any); ok {
		for _, v := range files {
			if file, ok := v.(map[string]any); ok {
				detail.Files = append(detail.Files, GistFile{
					Filename: getString(file, "filename"),
					Language: getString(file, "language"),
					Size:     getInt(file, "size"),
					Content:  getString(file, "content"),
				})
			}
		}
	}

	return detail
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
