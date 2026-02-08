package shodan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://internetdb.shodan.io"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ShodanResult represents the InternetDB lookup result for an IP
type ShodanResult struct {
	IP        string   `json:"ip"`
	Ports     []int    `json:"ports"`
	Hostnames []string `json:"hostnames"`
	CPEs      []string `json:"cpes"`
	Tags      []string `json:"tags"`
	Vulns     []string `json:"vulns"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shodan",
		Aliases: []string{"sh"},
		Short:   "Shodan InternetDB lookups",
		Long:    "Shodan InternetDB integration for free IP reconnaissance. No API key required.",
	}

	cmd.AddCommand(newLookupCmd())

	return cmd
}

func newLookupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lookup [ip]",
		Short: "Look up an IP address in Shodan InternetDB",
		Long:  "Query the Shodan InternetDB for open ports, hostnames, CPEs, tags, and vulnerabilities associated with an IP address.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ip := args[0]

			// Validate IP address
			if net.ParseIP(ip) == nil {
				return output.PrintError("invalid_ip", fmt.Sprintf("'%s' is not a valid IP address", ip), nil)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			reqURL := fmt.Sprintf("%s/%s", baseURL, ip)
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

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", fmt.Sprintf("no information found for IP %s", ip), nil)
			}

			if resp.StatusCode >= 400 {
				return output.PrintError("api_error", fmt.Sprintf("Shodan API error (HTTP %d): %s", resp.StatusCode, string(body)), nil)
			}

			var raw struct {
				IP        string   `json:"ip"`
				Ports     []int    `json:"ports"`
				Hostnames []string `json:"hostnames"`
				CPEs      []string `json:"cpes"`
				Tags      []string `json:"tags"`
				Vulns     []string `json:"vulns"`
			}

			if err := json.Unmarshal(body, &raw); err != nil {
				return output.PrintError("parse_error", "failed to parse Shodan response", nil)
			}

			// Ensure slices are non-nil for clean JSON output
			ports := raw.Ports
			if ports == nil {
				ports = []int{}
			}
			hostnames := raw.Hostnames
			if hostnames == nil {
				hostnames = []string{}
			}
			cpes := raw.CPEs
			if cpes == nil {
				cpes = []string{}
			}
			tags := raw.Tags
			if tags == nil {
				tags = []string{}
			}
			vulns := raw.Vulns
			if vulns == nil {
				vulns = []string{}
			}

			result := ShodanResult{
				IP:        ip,
				Ports:     ports,
				Hostnames: hostnames,
				CPEs:      cpes,
				Tags:      tags,
				Vulns:     vulns,
			}

			return output.Print(result)
		},
	}

	return cmd
}
