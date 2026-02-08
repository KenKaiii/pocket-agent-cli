package virustotal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://www.virustotal.com/api/v3"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// URLScanResult represents the result of a URL scan
type URLScanResult struct {
	URL          string `json:"url"`
	Malicious    int    `json:"malicious"`
	Suspicious   int    `json:"suspicious"`
	Harmless     int    `json:"harmless"`
	Undetected   int    `json:"undetected"`
	Timeout      int    `json:"timeout"`
	TotalEngines int    `json:"total_engines"`
	Permalink    string `json:"permalink"`
}

// DomainReport represents a domain analysis report
type DomainReport struct {
	Domain       string            `json:"domain"`
	Reputation   int               `json:"reputation"`
	Malicious    int               `json:"malicious"`
	Suspicious   int               `json:"suspicious"`
	Harmless     int               `json:"harmless"`
	Undetected   int               `json:"undetected"`
	Categories   map[string]string `json:"categories"`
	LastAnalysis string            `json:"last_analysis"`
}

// IPReport represents an IP address analysis report
type IPReport struct {
	IP           string `json:"ip"`
	ASOwner      string `json:"as_owner"`
	Country      string `json:"country"`
	Reputation   int    `json:"reputation"`
	Malicious    int    `json:"malicious"`
	Suspicious   int    `json:"suspicious"`
	Harmless     int    `json:"harmless"`
	Undetected   int    `json:"undetected"`
	LastAnalysis string `json:"last_analysis"`
}

// HashReport represents a file hash analysis report
type HashReport struct {
	Hash         string `json:"hash"`
	SHA256       string `json:"sha256"`
	FileName     string `json:"file_name"`
	FileType     string `json:"file_type"`
	Size         int64  `json:"size"`
	Malicious    int    `json:"malicious"`
	Suspicious   int    `json:"suspicious"`
	Harmless     int    `json:"harmless"`
	Undetected   int    `json:"undetected"`
	Reputation   int    `json:"reputation"`
	FirstSeen    string `json:"first_seen"`
	LastAnalysis string `json:"last_analysis"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "virustotal",
		Aliases: []string{"vt"},
		Short:   "VirusTotal security analysis",
		Long:    "VirusTotal API v3 integration for URL scanning, domain/IP reports, and file hash lookups.",
	}

	cmd.AddCommand(newURLCmd())
	cmd.AddCommand(newDomainCmd())
	cmd.AddCommand(newIPCmd())
	cmd.AddCommand(newHashCmd())

	return cmd
}

func getAPIKey() (string, error) {
	key, err := config.Get("virustotal_api_key")
	if err != nil {
		return "", fmt.Errorf("VirusTotal API key not configured. Set it with: pocket config set virustotal_api_key <key>")
	}
	if key == "" {
		return "", fmt.Errorf("VirusTotal API key not configured. Set it with: pocket config set virustotal_api_key <key>")
	}
	return key, nil
}

func doRequest(ctx context.Context, method, endpoint string, body io.Reader, contentType string) ([]byte, error) {
	apiKey, err := getAPIKey()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-apikey", apiKey)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("VirusTotal API error: %s (%s)", errResp.Error.Message, errResp.Error.Code)
		}
		return nil, fmt.Errorf("VirusTotal API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func newURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url [url]",
		Short: "Scan a URL for threats",
		Long:  "Submit a URL to VirusTotal for scanning and retrieve the analysis results.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]

			// Step 1: Submit URL for scanning
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			formData := url.Values{}
			formData.Set("url", targetURL)

			respBody, err := doRequest(ctx, "POST", baseURL+"/urls", strings.NewReader(formData.Encode()), "application/x-www-form-urlencoded")
			if err != nil {
				return output.PrintError("scan_submit_failed", err.Error(), nil)
			}

			var submitResp struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &submitResp); err != nil {
				return output.PrintError("parse_error", "failed to parse scan submission response", nil)
			}

			analysisID := submitResp.Data.ID
			if analysisID == "" {
				return output.PrintError("scan_error", "no analysis ID returned", nil)
			}

			// Step 2: Poll for results
			var stats struct {
				Malicious  int `json:"malicious"`
				Suspicious int `json:"suspicious"`
				Harmless   int `json:"harmless"`
				Undetected int `json:"undetected"`
				Timeout    int `json:"timeout"`
			}

			for attempt := 0; attempt < 10; attempt++ {
				time.Sleep(3 * time.Second)

				pollCtx, pollCancel := context.WithTimeout(context.Background(), 30*time.Second)
				pollBody, pollErr := doRequest(pollCtx, "GET", baseURL+"/analyses/"+analysisID, nil, "")
				pollCancel()

				if pollErr != nil {
					continue
				}

				var analysisResp struct {
					Data struct {
						Attributes struct {
							Status string `json:"status"`
							Stats  struct {
								Malicious  int `json:"malicious"`
								Suspicious int `json:"suspicious"`
								Harmless   int `json:"harmless"`
								Undetected int `json:"undetected"`
								Timeout    int `json:"timeout"`
							} `json:"stats"`
						} `json:"attributes"`
					} `json:"data"`
				}

				if err := json.Unmarshal(pollBody, &analysisResp); err != nil {
					continue
				}

				if analysisResp.Data.Attributes.Status == "completed" {
					stats = analysisResp.Data.Attributes.Stats
					result := URLScanResult{
						URL:          targetURL,
						Malicious:    stats.Malicious,
						Suspicious:   stats.Suspicious,
						Harmless:     stats.Harmless,
						Undetected:   stats.Undetected,
						Timeout:      stats.Timeout,
						TotalEngines: stats.Malicious + stats.Suspicious + stats.Harmless + stats.Undetected + stats.Timeout,
						Permalink:    fmt.Sprintf("https://www.virustotal.com/gui/url/%s", analysisID),
					}
					return output.Print(result)
				}
			}

			return output.PrintError("scan_timeout", "analysis did not complete within the polling window", map[string]string{
				"analysis_id": analysisID,
				"url":         targetURL,
			})
		},
	}

	return cmd
}

func newDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain [domain]",
		Short: "Get domain security report",
		Long:  "Retrieve a VirusTotal analysis report for a domain name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			respBody, err := doRequest(ctx, "GET", baseURL+"/domains/"+domain, nil, "")
			if err != nil {
				return output.PrintError("domain_lookup_failed", err.Error(), nil)
			}

			var resp struct {
				Data struct {
					Attributes struct {
						Reputation        int `json:"reputation"`
						LastAnalysisStats struct {
							Malicious  int `json:"malicious"`
							Suspicious int `json:"suspicious"`
							Harmless   int `json:"harmless"`
							Undetected int `json:"undetected"`
						} `json:"last_analysis_stats"`
						Categories       map[string]string `json:"categories"`
						LastAnalysisDate int64             `json:"last_analysis_date"`
					} `json:"attributes"`
				} `json:"data"`
			}

			if err := json.Unmarshal(respBody, &resp); err != nil {
				return output.PrintError("parse_error", "failed to parse domain report", nil)
			}

			attrs := resp.Data.Attributes
			lastAnalysis := ""
			if attrs.LastAnalysisDate > 0 {
				lastAnalysis = time.Unix(attrs.LastAnalysisDate, 0).UTC().Format(time.RFC3339)
			}

			categories := attrs.Categories
			if categories == nil {
				categories = make(map[string]string)
			}

			result := DomainReport{
				Domain:       domain,
				Reputation:   attrs.Reputation,
				Malicious:    attrs.LastAnalysisStats.Malicious,
				Suspicious:   attrs.LastAnalysisStats.Suspicious,
				Harmless:     attrs.LastAnalysisStats.Harmless,
				Undetected:   attrs.LastAnalysisStats.Undetected,
				Categories:   categories,
				LastAnalysis: lastAnalysis,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip [ip]",
		Short: "Get IP address security report",
		Long:  "Retrieve a VirusTotal analysis report for an IP address.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ip := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			respBody, err := doRequest(ctx, "GET", baseURL+"/ip_addresses/"+ip, nil, "")
			if err != nil {
				return output.PrintError("ip_lookup_failed", err.Error(), nil)
			}

			var resp struct {
				Data struct {
					Attributes struct {
						ASOwner           string `json:"as_owner"`
						Country           string `json:"country"`
						Reputation        int    `json:"reputation"`
						LastAnalysisStats struct {
							Malicious  int `json:"malicious"`
							Suspicious int `json:"suspicious"`
							Harmless   int `json:"harmless"`
							Undetected int `json:"undetected"`
						} `json:"last_analysis_stats"`
						LastAnalysisDate int64 `json:"last_analysis_date"`
					} `json:"attributes"`
				} `json:"data"`
			}

			if err := json.Unmarshal(respBody, &resp); err != nil {
				return output.PrintError("parse_error", "failed to parse IP report", nil)
			}

			attrs := resp.Data.Attributes
			lastAnalysis := ""
			if attrs.LastAnalysisDate > 0 {
				lastAnalysis = time.Unix(attrs.LastAnalysisDate, 0).UTC().Format(time.RFC3339)
			}

			result := IPReport{
				IP:           ip,
				ASOwner:      attrs.ASOwner,
				Country:      attrs.Country,
				Reputation:   attrs.Reputation,
				Malicious:    attrs.LastAnalysisStats.Malicious,
				Suspicious:   attrs.LastAnalysisStats.Suspicious,
				Harmless:     attrs.LastAnalysisStats.Harmless,
				Undetected:   attrs.LastAnalysisStats.Undetected,
				LastAnalysis: lastAnalysis,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newHashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hash [hash]",
		Short: "Get file hash report",
		Long:  "Retrieve a VirusTotal analysis report for a file hash (MD5, SHA1, or SHA256).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hash := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			respBody, err := doRequest(ctx, "GET", baseURL+"/files/"+hash, nil, "")
			if err != nil {
				return output.PrintError("hash_lookup_failed", err.Error(), nil)
			}

			var resp struct {
				Data struct {
					Attributes struct {
						SHA256            string `json:"sha256"`
						MeaningfulName    string `json:"meaningful_name"`
						TypeDescription   string `json:"type_description"`
						Size              int64  `json:"size"`
						Reputation        int    `json:"reputation"`
						LastAnalysisStats struct {
							Malicious  int `json:"malicious"`
							Suspicious int `json:"suspicious"`
							Harmless   int `json:"harmless"`
							Undetected int `json:"undetected"`
						} `json:"last_analysis_stats"`
						FirstSubmissionDate int64 `json:"first_submission_date"`
						LastAnalysisDate    int64 `json:"last_analysis_date"`
					} `json:"attributes"`
				} `json:"data"`
			}

			if err := json.Unmarshal(respBody, &resp); err != nil {
				return output.PrintError("parse_error", "failed to parse hash report", nil)
			}

			attrs := resp.Data.Attributes
			firstSeen := ""
			if attrs.FirstSubmissionDate > 0 {
				firstSeen = time.Unix(attrs.FirstSubmissionDate, 0).UTC().Format(time.RFC3339)
			}
			lastAnalysis := ""
			if attrs.LastAnalysisDate > 0 {
				lastAnalysis = time.Unix(attrs.LastAnalysisDate, 0).UTC().Format(time.RFC3339)
			}

			result := HashReport{
				Hash:         hash,
				SHA256:       attrs.SHA256,
				FileName:     attrs.MeaningfulName,
				FileType:     attrs.TypeDescription,
				Size:         attrs.Size,
				Malicious:    attrs.LastAnalysisStats.Malicious,
				Suspicious:   attrs.LastAnalysisStats.Suspicious,
				Harmless:     attrs.LastAnalysisStats.Harmless,
				Undetected:   attrs.LastAnalysisStats.Undetected,
				Reputation:   attrs.Reputation,
				FirstSeen:    firstSeen,
				LastAnalysis: lastAnalysis,
			}

			return output.Print(result)
		},
	}

	return cmd
}
