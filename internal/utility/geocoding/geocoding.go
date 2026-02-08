package geocoding

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

const baseURL = "https://nominatim.openstreetmap.org"

// GeoResult is LLM-friendly geocoding result
type GeoResult struct {
	Lat         string  `json:"lat"`
	Lon         string  `json:"lon"`
	DisplayName string  `json:"display_name"`
	Type        string  `json:"type"`
	Importance  float64 `json:"importance"`
}

// NewCmd returns the geocoding command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "geocode",
		Aliases: []string{"geo", "location"},
		Short:   "Geocoding commands (OpenStreetMap Nominatim)",
	}

	cmd.AddCommand(newForwardCmd())
	cmd.AddCommand(newReverseCmd())

	return cmd
}

func newForwardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forward [address]",
		Short: "Convert address to coordinates",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return forwardGeocode(query)
		},
	}

	return cmd
}

func newReverseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reverse [lat] [lon]",
		Short: "Convert coordinates to address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return reverseGeocode(args[0], args[1])
		},
	}

	return cmd
}

func forwardGeocode(query string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/search?q=%s&format=json&limit=5", baseURL, url.QueryEscape(query))

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

	var data []struct {
		Lat         string  `json:"lat"`
		Lon         string  `json:"lon"`
		DisplayName string  `json:"display_name"`
		Type        string  `json:"type"`
		Importance  float64 `json:"importance"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return output.PrintError("parse_failed", err.Error(), nil)
	}

	if len(data) == 0 {
		return output.PrintError("not_found", "No results found for: "+query, nil)
	}

	results := make([]GeoResult, 0, len(data))
	for _, d := range data {
		results = append(results, GeoResult{
			Lat:         d.Lat,
			Lon:         d.Lon,
			DisplayName: d.DisplayName,
			Type:        d.Type,
			Importance:  d.Importance,
		})
	}

	return output.Print(results)
}

func reverseGeocode(lat, lon string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/reverse?lat=%s&lon=%s&format=json",
		baseURL, url.QueryEscape(lat), url.QueryEscape(lon))

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

	var data struct {
		Lat         string  `json:"lat"`
		Lon         string  `json:"lon"`
		DisplayName string  `json:"display_name"`
		Type        string  `json:"type"`
		Importance  float64 `json:"importance"`
		Error       string  `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return output.PrintError("parse_failed", err.Error(), nil)
	}

	if data.Error != "" {
		return output.PrintError("not_found", data.Error, nil)
	}

	result := GeoResult{
		Lat:         data.Lat,
		Lon:         data.Lon,
		DisplayName: data.DisplayName,
		Type:        data.Type,
		Importance:  data.Importance,
	}

	return output.Print(result)
}
