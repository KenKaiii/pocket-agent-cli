package crtsh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://crt.sh"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// CertEntry represents a certificate transparency log entry
type CertEntry struct {
	ID           int64  `json:"id"`
	IssuerCA     string `json:"issuer_ca"`
	IssuerName   string `json:"issuer_name"`
	CommonName   string `json:"common_name"`
	NameValue    string `json:"name_value"`
	NotBefore    string `json:"not_before"`
	NotAfter     string `json:"not_after"`
	SerialNumber string `json:"serial_number"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "crtsh",
		Aliases: []string{"cert", "ct"},
		Short:   "Certificate Transparency lookups",
		Long:    "Query crt.sh for Certificate Transparency logs. No API key required.",
	}

	cmd.AddCommand(newLookupCmd())

	return cmd
}

func newLookupCmd() *cobra.Command {
	var limit int
	var includeExpired bool

	cmd := &cobra.Command{
		Use:   "lookup [domain]",
		Short: "Look up certificates for a domain",
		Long:  "Query Certificate Transparency logs via crt.sh for certificates issued for a domain.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			params := url.Values{}
			params.Set("q", domain)
			params.Set("output", "json")
			if !includeExpired {
				params.Set("exclude", "expired")
			}

			reqURL := fmt.Sprintf("%s/?%s", baseURL, params.Encode())
			req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
			if err != nil {
				return output.PrintError("request_error", fmt.Sprintf("failed to create request: %s", err.Error()), nil)
			}
			req.Header.Set("User-Agent", "Pocket-CLI/1.0")

			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", fmt.Sprintf("request failed: %s", err.Error()), nil)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_error", fmt.Sprintf("failed to read response: %s", err.Error()), nil)
			}

			if resp.StatusCode >= 400 {
				return output.PrintError("api_error", fmt.Sprintf("crt.sh error (HTTP %d): %s", resp.StatusCode, string(body)), nil)
			}

			var rawEntries []struct {
				ID           int64  `json:"id"`
				IssuerCAID   int64  `json:"issuer_ca_id"`
				IssuerName   string `json:"issuer_name"`
				CommonName   string `json:"common_name"`
				NameValue    string `json:"name_value"`
				NotBefore    string `json:"not_before"`
				NotAfter     string `json:"not_after"`
				SerialNumber string `json:"serial_number"`
			}

			if err := json.Unmarshal(body, &rawEntries); err != nil {
				return output.PrintError("parse_error", "failed to parse crt.sh response", nil)
			}

			// Apply limit
			if limit > 0 && len(rawEntries) > limit {
				rawEntries = rawEntries[:limit]
			}

			results := make([]CertEntry, len(rawEntries))
			for i, raw := range rawEntries {
				results[i] = CertEntry{
					ID:           raw.ID,
					IssuerCA:     fmt.Sprintf("%d", raw.IssuerCAID),
					IssuerName:   raw.IssuerName,
					CommonName:   raw.CommonName,
					NameValue:    raw.NameValue,
					NotBefore:    raw.NotBefore,
					NotAfter:     raw.NotAfter,
					SerialNumber: raw.SerialNumber,
				}
			}

			return output.Print(results)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of results to return")
	cmd.Flags().BoolVar(&includeExpired, "expired", false, "Include expired certificates")

	return cmd
}
