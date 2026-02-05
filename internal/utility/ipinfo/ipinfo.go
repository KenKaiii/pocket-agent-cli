package ipinfo

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://ipinfo.io"

var client = &http.Client{Timeout: 10 * time.Second}

// IPInfo is LLM-friendly IP information
type IPInfo struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname,omitempty"`
	City     string `json:"city,omitempty"`
	Region   string `json:"region,omitempty"`
	Country  string `json:"country,omitempty"`
	Location string `json:"loc,omitempty"`
	Org      string `json:"org,omitempty"`
	Postal   string `json:"postal,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ip",
		Aliases: []string{"ipinfo", "geo"},
		Short:   "IP geolocation commands",
	}

	cmd.AddCommand(newLookupCmd())
	cmd.AddCommand(newMyIPCmd())

	return cmd
}

func newLookupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lookup [ip]",
		Short: "Get geolocation info for an IP address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ip := args[0]

			// Validate IP
			if net.ParseIP(ip) == nil {
				return output.PrintError("invalid_ip", "Invalid IP address: "+ip, nil)
			}

			return fetchIPInfo(ip)
		},
	}

	return cmd
}

func newMyIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Get your current public IP and location",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchIPInfo("")
		},
	}

	return cmd
}

func fetchIPInfo(ip string) error {
	reqURL := baseURL + "/json"
	if ip != "" {
		reqURL = fmt.Sprintf("%s/%s/json", baseURL, ip)
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return output.PrintError("rate_limited", "Rate limit exceeded, try again later", nil)
	}

	if resp.StatusCode >= 400 {
		return output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return output.PrintError("parse_failed", err.Error(), nil)
	}

	return output.Print(info)
}
