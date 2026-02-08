package gdrive

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

const baseURL = "https://www.googleapis.com/drive/v3"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// DriveSearchResult holds search results
type DriveSearchResult struct {
	Query string      `json:"query"`
	Files []DriveFile `json:"files"`
	Count int         `json:"count"`
}

// DriveFile holds file metadata from search
type DriveFile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MimeType   string `json:"mime_type"`
	Size       string `json:"size,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
	URL        string `json:"url,omitempty"`
}

// FileInfo holds detailed file metadata
type FileInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MimeType    string `json:"mime_type"`
	Size        string `json:"size,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
}

// NewCmd returns the Google Drive command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gdrive",
		Aliases: []string{"drive"},
		Short:   "Google Drive commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newInfoCmd())

	return cmd
}

func getAPIKey() (string, error) {
	key, err := config.Get("google_api_key")
	if err != nil || key == "" {
		return "", output.PrintError("setup_required", "Google API key not configured", map[string]any{
			"missing":   []string{"google_api_key"},
			"setup_cmd": "pocket config set google_api_key <your-key>",
			"hint":      "Get an API key from https://console.cloud.google.com/apis/credentials",
		})
	}
	return key, nil
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search files in Google Drive",
		Long:  `Search public files using Google Drive query syntax: name contains 'test', mimeType='application/pdf', etc.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			query := args[0]
			params := url.Values{
				"q":        {query},
				"key":      {apiKey},
				"fields":   {"files(id,name,mimeType,size,createdTime,modifiedTime,webViewLink)"},
				"pageSize": {fmt.Sprintf("%d", limit)},
			}

			reqURL := fmt.Sprintf("%s/files?%s", baseURL, params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Files []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					MimeType     string `json:"mimeType"`
					Size         string `json:"size"`
					CreatedTime  string `json:"createdTime"`
					ModifiedTime string `json:"modifiedTime"`
					WebViewLink  string `json:"webViewLink"`
				} `json:"files"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
			}

			files := make([]DriveFile, 0, len(resp.Files))
			for _, f := range resp.Files {
				files = append(files, DriveFile{
					ID:         f.ID,
					Name:       f.Name,
					MimeType:   f.MimeType,
					Size:       formatSize(f.Size),
					CreatedAt:  formatTime(f.CreatedTime),
					ModifiedAt: formatTime(f.ModifiedTime),
					URL:        f.WebViewLink,
				})
			}

			result := DriveSearchResult{
				Query: query,
				Files: files,
				Count: len(files),
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of results")

	return cmd
}

func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [file-id]",
		Short: "Get file metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			fileID := args[0]
			params := url.Values{
				"key":    {apiKey},
				"fields": {"id,name,mimeType,size,createdTime,modifiedTime,webViewLink,description,owners"},
			}

			reqURL := fmt.Sprintf("%s/files/%s?%s", baseURL, url.PathEscape(fileID), params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				MimeType     string `json:"mimeType"`
				Size         string `json:"size"`
				CreatedTime  string `json:"createdTime"`
				ModifiedTime string `json:"modifiedTime"`
				WebViewLink  string `json:"webViewLink"`
				Description  string `json:"description"`
				Owners       []struct {
					DisplayName string `json:"displayName"`
				} `json:"owners"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
			}

			ownerName := ""
			if len(resp.Owners) > 0 {
				ownerName = resp.Owners[0].DisplayName
			}

			result := FileInfo{
				ID:          resp.ID,
				Name:        resp.Name,
				MimeType:    resp.MimeType,
				Size:        formatSize(resp.Size),
				CreatedAt:   formatTime(resp.CreatedTime),
				ModifiedAt:  formatTime(resp.ModifiedTime),
				URL:         resp.WebViewLink,
				Description: resp.Description,
				Owner:       ownerName,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func doRequest(reqURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("request_failed", fmt.Sprintf("Failed to create request: %s", err.Error()), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, output.PrintError("request_failed", fmt.Sprintf("Request failed: %s", err.Error()), nil)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, output.PrintError("read_failed", fmt.Sprintf("Failed to read response: %s", err.Error()), nil)
	}

	if resp.StatusCode == 403 {
		return nil, output.PrintError("forbidden", "Access denied. Ensure the file is public and the API key is valid", nil)
	}

	if resp.StatusCode == 404 {
		return nil, output.PrintError("not_found", "File not found", nil)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, output.PrintError("api_error", errResp.Error.Message, nil)
		}
		return nil, output.PrintError("api_error", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return body, nil
}

func formatTime(isoTime string) string {
	if isoTime == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime
	}

	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	case diff < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(diff.Hours()/(24*7)))
	case diff < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(diff.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(diff.Hours()/(24*365)))
	}
}

func formatSize(sizeStr string) string {
	if sizeStr == "" {
		return ""
	}

	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return sizeStr
	}

	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
