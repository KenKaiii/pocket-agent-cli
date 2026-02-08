package wayback

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const (
	availabilityAPI = "https://archive.org/wayback/available"
	cdxAPI          = "http://web.archive.org/cdx/search/cdx"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Snapshot is an LLM-friendly archived snapshot
type Snapshot struct {
	URL       string `json:"url"`
	Timestamp string `json:"timestamp"`
	Available bool   `json:"available"`
	Status    string `json:"status,omitempty"`
}

// SnapshotDetail is detailed snapshot information
type SnapshotDetail struct {
	OriginalURL string `json:"original_url"`
	ArchiveURL  string `json:"archive_url"`
	Timestamp   string `json:"timestamp"`
	Status      string `json:"status"`
	MimeType    string `json:"mime_type,omitempty"`
	Digest      string `json:"digest,omitempty"`
	Length      string `json:"length,omitempty"`
}

// AvailabilityResult is the result of checking URL availability
type AvailabilityResult struct {
	URL       string          `json:"url"`
	Available bool            `json:"available"`
	Snapshot  *SnapshotDetail `json:"snapshot,omitempty"`
	CheckedAt string          `json:"checked_at"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wayback",
		Aliases: []string{"archive", "ia"},
		Short:   "Wayback Machine commands (Internet Archive)",
	}

	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newLatestCmd())
	cmd.AddCommand(newSnapshotsCmd())

	return cmd
}

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [url]",
		Short: "Check if URL has archived snapshots",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := normalizeURL(args[0])
			reqURL := fmt.Sprintf("%s?url=%s", availabilityAPI, url.QueryEscape(targetURL))

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				URL               string `json:"url"`
				ArchivedSnapshots struct {
					Closest *struct {
						Status    string `json:"status"`
						Available bool   `json:"available"`
						URL       string `json:"url"`
						Timestamp string `json:"timestamp"`
					} `json:"closest"`
				} `json:"archived_snapshots"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			result := AvailabilityResult{
				URL:       targetURL,
				Available: false,
				CheckedAt: time.Now().UTC().Format(time.RFC3339),
			}

			if data.ArchivedSnapshots.Closest != nil && data.ArchivedSnapshots.Closest.Available {
				result.Available = true
				result.Snapshot = &SnapshotDetail{
					OriginalURL: targetURL,
					ArchiveURL:  data.ArchivedSnapshots.Closest.URL,
					Timestamp:   formatTimestamp(data.ArchivedSnapshots.Closest.Timestamp),
					Status:      data.ArchivedSnapshots.Closest.Status,
				}
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newLatestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "latest [url]",
		Short: "Get latest archived snapshot of URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := normalizeURL(args[0])
			reqURL := fmt.Sprintf("%s?url=%s", availabilityAPI, url.QueryEscape(targetURL))

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				URL               string `json:"url"`
				ArchivedSnapshots struct {
					Closest *struct {
						Status    string `json:"status"`
						Available bool   `json:"available"`
						URL       string `json:"url"`
						Timestamp string `json:"timestamp"`
					} `json:"closest"`
				} `json:"archived_snapshots"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if data.ArchivedSnapshots.Closest == nil || !data.ArchivedSnapshots.Closest.Available {
				return output.PrintError("not_found", "No archived snapshot found for: "+targetURL, nil)
			}

			snapshot := SnapshotDetail{
				OriginalURL: targetURL,
				ArchiveURL:  data.ArchivedSnapshots.Closest.URL,
				Timestamp:   formatTimestamp(data.ArchivedSnapshots.Closest.Timestamp),
				Status:      data.ArchivedSnapshots.Closest.Status,
			}

			return output.Print(snapshot)
		},
	}

	return cmd
}

func newSnapshotsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "snapshots [url]",
		Short: "List available snapshots for URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := normalizeURL(args[0])
			reqURL := fmt.Sprintf("%s?url=%s&output=json&limit=%d&fl=timestamp,original,statuscode,mimetype,digest,length",
				cdxAPI, url.QueryEscape(targetURL), limit)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data [][]string
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if len(data) <= 1 {
				return output.PrintError("not_found", "No snapshots found for: "+targetURL, nil)
			}

			// First row is headers: [timestamp, original, statuscode, mimetype, digest, length]
			// Skip header row
			var snapshots []SnapshotDetail
			for i := 1; i < len(data); i++ {
				row := data[i]
				if len(row) < 6 {
					continue
				}

				archiveURL := fmt.Sprintf("https://web.archive.org/web/%s/%s", row[0], row[1])
				snapshots = append(snapshots, SnapshotDetail{
					OriginalURL: row[1],
					ArchiveURL:  archiveURL,
					Timestamp:   formatTimestamp(row[0]),
					Status:      row[2],
					MimeType:    row[3],
					Digest:      row[4],
					Length:      row[5],
				})
			}

			if len(snapshots) == 0 {
				return output.PrintError("not_found", "No snapshots found for: "+targetURL, nil)
			}

			return output.Print(snapshots)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of snapshots to return")

	return cmd
}

func doRequest(reqURL string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		return nil, output.PrintError("rate_limited", "Wayback Machine rate limit exceeded, try again later", nil)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return resp, nil
}

// normalizeURL ensures the URL has a scheme
func normalizeURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}
	// If no scheme, add https://
	if len(rawURL) < 8 || (rawURL[:7] != "http://" && rawURL[:8] != "https://") {
		return "https://" + rawURL
	}
	return rawURL
}

// formatTimestamp converts Wayback timestamp (YYYYMMDDhhmmss) to readable format
func formatTimestamp(ts string) string {
	if len(ts) < 14 {
		return ts
	}
	// Parse YYYYMMDDhhmmss format
	t, err := time.Parse("20060102150405", ts)
	if err != nil {
		return ts
	}
	return t.Format(time.RFC3339)
}
