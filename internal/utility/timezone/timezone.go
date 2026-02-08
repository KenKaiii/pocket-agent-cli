package timezone

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

var httpClient = &http.Client{Timeout: 30 * time.Second}

const baseURL = "http://worldtimeapi.org/api"

// TimeInfo is LLM-friendly timezone information
type TimeInfo struct {
	Timezone     string `json:"timezone"`
	DateTime     string `json:"datetime"`
	UTCOffset    string `json:"utc_offset"`
	DayOfWeek    int    `json:"day_of_week"`
	WeekNumber   int    `json:"week_number"`
	DST          bool   `json:"dst"`
	Abbreviation string `json:"abbreviation"`
	UnixTime     int64  `json:"unixtime"`
}

// NewCmd returns the timezone command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "timezone",
		Aliases: []string{"tz", "time"},
		Short:   "Timezone commands (WorldTimeAPI)",
	}

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newIPCmd())
	cmd.AddCommand(newListCmd())

	return cmd
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [timezone]",
		Short: "Get time for a timezone (e.g., America/New_York)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tz := args[0]
			return fetchTimezone(fmt.Sprintf("%s/timezone/%s", baseURL, url.PathEscape(tz)))
		},
	}

	return cmd
}

func newIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip [ip-address]",
		Short: "Get timezone by IP address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ip := args[0]
			return fetchTimezone(fmt.Sprintf("%s/ip/%s", baseURL, url.PathEscape(ip)))
		},
	}

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available timezones",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listTimezones()
		},
	}

	return cmd
}

func fetchTimezone(reqURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return output.PrintError("not_found", "Timezone not found", nil)
	}

	if resp.StatusCode >= 400 {
		return output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	var data struct {
		Timezone     string `json:"timezone"`
		DateTime     string `json:"datetime"`
		UTCOffset    string `json:"utc_offset"`
		DayOfWeek    int    `json:"day_of_week"`
		WeekNumber   int    `json:"week_number"`
		DST          bool   `json:"dst"`
		Abbreviation string `json:"abbreviation"`
		UnixTime     int64  `json:"unixtime"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return output.PrintError("parse_failed", err.Error(), nil)
	}

	result := TimeInfo{
		Timezone:     data.Timezone,
		DateTime:     data.DateTime,
		UTCOffset:    data.UTCOffset,
		DayOfWeek:    data.DayOfWeek,
		WeekNumber:   data.WeekNumber,
		DST:          data.DST,
		Abbreviation: data.Abbreviation,
		UnixTime:     data.UnixTime,
	}

	return output.Print(result)
}

func listTimezones() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/timezone", baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	var timezones []string
	if err := json.NewDecoder(resp.Body).Decode(&timezones); err != nil {
		return output.PrintError("parse_failed", err.Error(), nil)
	}

	// Group by region for LLM-friendly output
	regions := make(map[string][]string)
	for _, tz := range timezones {
		parts := strings.SplitN(tz, "/", 2)
		region := parts[0]
		regions[region] = append(regions[region], tz)
	}

	type TimezoneList struct {
		Total   int                 `json:"total"`
		Regions map[string][]string `json:"regions"`
	}

	return output.Print(TimezoneList{
		Total:   len(timezones),
		Regions: regions,
	})
}
