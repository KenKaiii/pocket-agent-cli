package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var client = &http.Client{Timeout: 10 * time.Second}

// Current is LLM-friendly current weather
type Current struct {
	Location    string `json:"location"`
	Condition   string `json:"condition"`
	TempC       int    `json:"temp_c"`
	TempF       int    `json:"temp_f"`
	FeelsLikeC  int    `json:"feels_like_c"`
	FeelsLikeF  int    `json:"feels_like_f"`
	Humidity    int    `json:"humidity"`
	WindKph     int    `json:"wind_kph"`
	WindMph     int    `json:"wind_mph"`
	WindDir     string `json:"wind_dir"`
	Visibility  int    `json:"visibility_km"`
	UV          int    `json:"uv"`
}

// Forecast is LLM-friendly forecast day
type Forecast struct {
	Date      string `json:"date"`
	Condition string `json:"condition"`
	MaxC      int    `json:"max_c"`
	MinC      int    `json:"min_c"`
	MaxF      int    `json:"max_f"`
	MinF      int    `json:"min_f"`
	ChanceRain int   `json:"chance_rain"`
	Humidity  int    `json:"humidity"`
}

// Weather is the full response
type Weather struct {
	Current  Current    `json:"current"`
	Forecast []Forecast `json:"forecast,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "weather",
		Aliases: []string{"wttr"},
		Short:   "Weather commands",
	}

	cmd.AddCommand(newNowCmd())
	cmd.AddCommand(newForecastCmd())

	return cmd
}

func newNowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "now [location]",
		Short: "Get current weather",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location := args[0]
			return fetchWeather(location, 0)
		},
	}

	return cmd
}

func newForecastCmd() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "forecast [location]",
		Short: "Get weather forecast",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location := args[0]
			return fetchWeather(location, days)
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 3, "Number of days (1-3)")

	return cmd
}

func fetchWeather(location string, forecastDays int) error {
	// Using wttr.in JSON API
	reqURL := fmt.Sprintf("https://wttr.in/%s?format=j1", url.PathEscape(location))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "curl/7.68.0")

	resp, err := client.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	var data struct {
		CurrentCondition []struct {
			TempC       string `json:"temp_C"`
			TempF       string `json:"temp_F"`
			FeelsLikeC  string `json:"FeelsLikeC"`
			FeelsLikeF  string `json:"FeelsLikeF"`
			Humidity    string `json:"humidity"`
			WindspeedKmph string `json:"windspeedKmph"`
			WindspeedMiles string `json:"windspeedMiles"`
			WindDir16Point string `json:"winddir16Point"`
			Visibility  string `json:"visibility"`
			UVIndex     string `json:"uvIndex"`
			WeatherDesc []struct {
				Value string `json:"value"`
			} `json:"weatherDesc"`
		} `json:"current_condition"`
		NearestArea []struct {
			AreaName []struct {
				Value string `json:"value"`
			} `json:"areaName"`
			Country []struct {
				Value string `json:"value"`
			} `json:"country"`
		} `json:"nearest_area"`
		Weather []struct {
			Date    string `json:"date"`
			MaxTempC string `json:"maxtempC"`
			MinTempC string `json:"mintempC"`
			MaxTempF string `json:"maxtempF"`
			MinTempF string `json:"mintempF"`
			Hourly []struct {
				ChanceOfRain string `json:"chanceofrain"`
				Humidity     string `json:"humidity"`
				WeatherDesc  []struct {
					Value string `json:"value"`
				} `json:"weatherDesc"`
			} `json:"hourly"`
		} `json:"weather"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return output.PrintError("parse_failed", err.Error(), nil)
	}

	if len(data.CurrentCondition) == 0 {
		return output.PrintError("not_found", "Location not found: "+location, nil)
	}

	cc := data.CurrentCondition[0]

	// Build location string
	loc := location
	if len(data.NearestArea) > 0 {
		area := data.NearestArea[0]
		if len(area.AreaName) > 0 && len(area.Country) > 0 {
			loc = area.AreaName[0].Value + ", " + area.Country[0].Value
		}
	}

	condition := ""
	if len(cc.WeatherDesc) > 0 {
		condition = cc.WeatherDesc[0].Value
	}

	weather := Weather{
		Current: Current{
			Location:   loc,
			Condition:  condition,
			TempC:      atoi(cc.TempC),
			TempF:      atoi(cc.TempF),
			FeelsLikeC: atoi(cc.FeelsLikeC),
			FeelsLikeF: atoi(cc.FeelsLikeF),
			Humidity:   atoi(cc.Humidity),
			WindKph:    atoi(cc.WindspeedKmph),
			WindMph:    atoi(cc.WindspeedMiles),
			WindDir:    cc.WindDir16Point,
			Visibility: atoi(cc.Visibility),
			UV:         atoi(cc.UVIndex),
		},
	}

	// Add forecast if requested
	if forecastDays > 0 && len(data.Weather) > 0 {
		limit := forecastDays
		if limit > len(data.Weather) {
			limit = len(data.Weather)
		}

		for i := 0; i < limit; i++ {
			day := data.Weather[i]

			// Get average chance of rain and humidity from hourly data
			chanceRain := 0
			humidity := 0
			condition := ""
			if len(day.Hourly) > 0 {
				// Use midday values
				mid := len(day.Hourly) / 2
				chanceRain = atoi(day.Hourly[mid].ChanceOfRain)
				humidity = atoi(day.Hourly[mid].Humidity)
				if len(day.Hourly[mid].WeatherDesc) > 0 {
					condition = day.Hourly[mid].WeatherDesc[0].Value
				}
			}

			weather.Forecast = append(weather.Forecast, Forecast{
				Date:       day.Date,
				Condition:  condition,
				MaxC:       atoi(day.MaxTempC),
				MinC:       atoi(day.MinTempC),
				MaxF:       atoi(day.MaxTempF),
				MinF:       atoi(day.MinTempF),
				ChanceRain: chanceRain,
				Humidity:   humidity,
			})
		}
	}

	return output.Print(weather)
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
