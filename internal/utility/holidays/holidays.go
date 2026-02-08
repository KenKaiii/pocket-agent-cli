package holidays

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://date.nager.at/api/v3"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Holiday is LLM-friendly holiday output
type Holiday struct {
	Date        string   `json:"date"`
	Name        string   `json:"name"`
	LocalName   string   `json:"local_name"`
	CountryCode string   `json:"country_code"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties,omitempty"`
	Types       []string `json:"types,omitempty"`
}

// Country is LLM-friendly country output
type Country struct {
	CountryCode string `json:"country_code"`
	Name        string `json:"name"`
}

// HolidayList is a wrapper for holiday list output
type HolidayList struct {
	CountryCode string    `json:"country_code"`
	Year        int       `json:"year"`
	Count       int       `json:"count"`
	Holidays    []Holiday `json:"holidays"`
}

// UpcomingHolidays is a wrapper for upcoming holidays output
type UpcomingHolidays struct {
	CountryCode string    `json:"country_code"`
	AsOf        string    `json:"as_of"`
	Count       int       `json:"count"`
	Holidays    []Holiday `json:"holidays"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "holidays",
		Aliases: []string{"holiday", "hol"},
		Short:   "Public holidays commands (Nager.Date API)",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newNextCmd())
	cmd.AddCommand(newCountriesCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [country-code] [year]",
		Short: "List public holidays for a country (e.g., US 2024)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			countryCode := strings.ToUpper(args[0])
			year := time.Now().Year()

			if len(args) > 1 {
				parsedYear, err := strconv.Atoi(args[1])
				if err != nil {
					return output.PrintError("invalid_year", "Year must be a valid number", nil)
				}
				year = parsedYear
			}

			reqURL := fmt.Sprintf("%s/PublicHolidays/%d/%s", baseURL, year, countryCode)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Country not found: "+countryCode, nil)
			}

			var data []struct {
				Date        string   `json:"date"`
				LocalName   string   `json:"localName"`
				Name        string   `json:"name"`
				CountryCode string   `json:"countryCode"`
				Fixed       bool     `json:"fixed"`
				Global      bool     `json:"global"`
				Counties    []string `json:"counties"`
				Types       []string `json:"types"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var holidays []Holiday
			for _, h := range data {
				holidays = append(holidays, Holiday{
					Date:        h.Date,
					Name:        h.Name,
					LocalName:   h.LocalName,
					CountryCode: h.CountryCode,
					Fixed:       h.Fixed,
					Global:      h.Global,
					Counties:    h.Counties,
					Types:       h.Types,
				})
			}

			result := HolidayList{
				CountryCode: countryCode,
				Year:        year,
				Count:       len(holidays),
				Holidays:    holidays,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newNextCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "next [country-code]",
		Short: "Get upcoming holidays from current date",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			countryCode := strings.ToUpper(args[0])
			now := time.Now()
			year := now.Year()
			today := now.Format("2006-01-02")

			reqURL := fmt.Sprintf("%s/PublicHolidays/%d/%s", baseURL, year, countryCode)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Country not found: "+countryCode, nil)
			}

			var data []struct {
				Date        string   `json:"date"`
				LocalName   string   `json:"localName"`
				Name        string   `json:"name"`
				CountryCode string   `json:"countryCode"`
				Fixed       bool     `json:"fixed"`
				Global      bool     `json:"global"`
				Counties    []string `json:"counties"`
				Types       []string `json:"types"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var upcoming []Holiday
			for _, h := range data {
				if h.Date >= today {
					upcoming = append(upcoming, Holiday{
						Date:        h.Date,
						Name:        h.Name,
						LocalName:   h.LocalName,
						CountryCode: h.CountryCode,
						Fixed:       h.Fixed,
						Global:      h.Global,
						Counties:    h.Counties,
						Types:       h.Types,
					})
				}
			}

			// Limit results if specified
			if limit > 0 && len(upcoming) > limit {
				upcoming = upcoming[:limit]
			}

			if len(upcoming) == 0 {
				return output.PrintError("not_found", "No upcoming holidays found for "+countryCode+" in "+strconv.Itoa(year), nil)
			}

			result := UpcomingHolidays{
				CountryCode: countryCode,
				AsOf:        today,
				Count:       len(upcoming),
				Holidays:    upcoming,
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Max number of upcoming holidays to return (0 = all)")

	return cmd
}

func newCountriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "countries",
		Short: "List available country codes",
		RunE: func(cmd *cobra.Command, args []string) error {
			reqURL := fmt.Sprintf("%s/AvailableCountries", baseURL)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data []struct {
				CountryCode string `json:"countryCode"`
				Name        string `json:"name"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if len(data) == 0 {
				return output.PrintError("not_found", "No countries found", nil)
			}

			var countries []Country
			for _, c := range data {
				countries = append(countries, Country{
					CountryCode: c.CountryCode,
					Name:        c.Name,
				})
			}

			return output.Print(countries)
		},
	}

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
		return nil, output.PrintError("rate_limited", "Nager.Date rate limit exceeded, try again later", nil)
	}

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		resp.Body.Close()
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return resp, nil
}
