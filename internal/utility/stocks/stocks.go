package stocks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://www.alphavantage.co/query"

var httpClient = &http.Client{}

// Quote is LLM-friendly stock quote output
type Quote struct {
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent string  `json:"change_percent"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Open          float64 `json:"open"`
	PrevClose     float64 `json:"previous_close"`
	Volume        int64   `json:"volume"`
	LatestDay     string  `json:"latest_day"`
}

// SearchResult is a stock symbol search result
type SearchResult struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Region   string `json:"region"`
	Currency string `json:"currency"`
}

// CompanyInfo is detailed company information
type CompanyInfo struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Description   string  `json:"description,omitempty"`
	Exchange      string  `json:"exchange"`
	Sector        string  `json:"sector,omitempty"`
	Industry      string  `json:"industry,omitempty"`
	MarketCap     int64   `json:"market_cap,omitempty"`
	PERatio       float64 `json:"pe_ratio,omitempty"`
	DividendYield float64 `json:"dividend_yield,omitempty"`
	EPS           float64 `json:"eps,omitempty"`
	High52Week    float64 `json:"high_52_week,omitempty"`
	Low52Week     float64 `json:"low_52_week,omitempty"`
	Country       string  `json:"country,omitempty"`
	Website       string  `json:"website,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stocks",
		Aliases: []string{"stock", "market"},
		Short:   "Stock market commands (Alpha Vantage)",
	}

	cmd.AddCommand(newQuoteCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newInfoCmd())

	return cmd
}

func newQuoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote [symbol]",
		Short: "Get current stock quote (price, change, volume)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			symbol := strings.ToUpper(args[0])
			reqURL := fmt.Sprintf("%s?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
				baseURL, url.QueryEscape(symbol), apiKey)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				GlobalQuote struct {
					Symbol        string `json:"01. symbol"`
					Open          string `json:"02. open"`
					High          string `json:"03. high"`
					Low           string `json:"04. low"`
					Price         string `json:"05. price"`
					Volume        string `json:"06. volume"`
					LatestDay     string `json:"07. latest trading day"`
					PrevClose     string `json:"08. previous close"`
					Change        string `json:"09. change"`
					ChangePercent string `json:"10. change percent"`
				} `json:"Global Quote"`
				Note string `json:"Note"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			// Check for rate limit note
			if data.Note != "" {
				return output.PrintError("rate_limited", "Alpha Vantage API rate limit reached. Free tier allows 25 requests/day.", nil)
			}

			if data.GlobalQuote.Symbol == "" {
				return output.PrintError("not_found", "Stock symbol not found: "+symbol, nil)
			}

			quote := Quote{
				Symbol:        data.GlobalQuote.Symbol,
				Price:         parseFloat(data.GlobalQuote.Price),
				Change:        parseFloat(data.GlobalQuote.Change),
				ChangePercent: data.GlobalQuote.ChangePercent,
				High:          parseFloat(data.GlobalQuote.High),
				Low:           parseFloat(data.GlobalQuote.Low),
				Open:          parseFloat(data.GlobalQuote.Open),
				PrevClose:     parseFloat(data.GlobalQuote.PrevClose),
				Volume:        parseInt(data.GlobalQuote.Volume),
				LatestDay:     data.GlobalQuote.LatestDay,
			}

			return output.Print(quote)
		},
	}

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for stock symbols",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			query := args[0]
			reqURL := fmt.Sprintf("%s?function=SYMBOL_SEARCH&keywords=%s&apikey=%s",
				baseURL, url.QueryEscape(query), apiKey)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				BestMatches []struct {
					Symbol   string `json:"1. symbol"`
					Name     string `json:"2. name"`
					Type     string `json:"3. type"`
					Region   string `json:"4. region"`
					Currency string `json:"8. currency"`
				} `json:"bestMatches"`
				Note string `json:"Note"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			// Check for rate limit note
			if data.Note != "" {
				return output.PrintError("rate_limited", "Alpha Vantage API rate limit reached. Free tier allows 25 requests/day.", nil)
			}

			if len(data.BestMatches) == 0 {
				return output.PrintError("not_found", "No symbols found for: "+query, nil)
			}

			// Limit results
			matches := data.BestMatches
			if len(matches) > limit {
				matches = matches[:limit]
			}

			var results []SearchResult
			for _, m := range matches {
				results = append(results, SearchResult{
					Symbol:   m.Symbol,
					Name:     m.Name,
					Type:     m.Type,
					Region:   m.Region,
					Currency: m.Currency,
				})
			}

			return output.Print(results)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Max results")

	return cmd
}

func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [symbol]",
		Short: "Get company overview/info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			symbol := strings.ToUpper(args[0])
			reqURL := fmt.Sprintf("%s?function=OVERVIEW&symbol=%s&apikey=%s",
				baseURL, url.QueryEscape(symbol), apiKey)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				Symbol            string `json:"Symbol"`
				Name              string `json:"Name"`
				Description       string `json:"Description"`
				Exchange          string `json:"Exchange"`
				Sector            string `json:"Sector"`
				Industry          string `json:"Industry"`
				MarketCap         string `json:"MarketCapitalization"`
				PERatio           string `json:"PERatio"`
				DividendYield     string `json:"DividendYield"`
				EPS               string `json:"EPS"`
				Week52High        string `json:"52WeekHigh"`
				Week52Low         string `json:"52WeekLow"`
				Country           string `json:"Country"`
				OfficialSite      string `json:"OfficialSite"`
				Note              string `json:"Note"`
				InformationNotice string `json:"Information"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			// Check for rate limit note
			if data.Note != "" {
				return output.PrintError("rate_limited", "Alpha Vantage API rate limit reached. Free tier allows 25 requests/day.", nil)
			}

			// Check for information notice (often indicates invalid API key or symbol)
			if data.InformationNotice != "" {
				return output.PrintError("api_error", data.InformationNotice, nil)
			}

			if data.Symbol == "" {
				return output.PrintError("not_found", "Company not found: "+symbol, nil)
			}

			// Truncate description for LLM friendliness
			desc := data.Description
			if len(desc) > 500 {
				desc = desc[:500] + "..."
			}

			info := CompanyInfo{
				Symbol:        data.Symbol,
				Name:          data.Name,
				Description:   desc,
				Exchange:      data.Exchange,
				Sector:        data.Sector,
				Industry:      data.Industry,
				MarketCap:     parseInt(data.MarketCap),
				PERatio:       parseFloat(data.PERatio),
				DividendYield: parseFloat(data.DividendYield),
				EPS:           parseFloat(data.EPS),
				High52Week:    parseFloat(data.Week52High),
				Low52Week:     parseFloat(data.Week52Low),
				Country:       data.Country,
				Website:       data.OfficialSite,
			}

			return output.Print(info)
		},
	}

	return cmd
}

func getAPIKey() (string, error) {
	apiKey, err := config.Get("alphavantage_key")
	if err != nil {
		return "", output.PrintError("config_error", err.Error(), nil)
	}

	if apiKey == "" {
		return "", output.PrintError("missing_api_key",
			"Alpha Vantage API key not configured",
			map[string]string{
				"setup":     "Get a free API key at https://www.alphavantage.co/support/#api-key",
				"configure": "pocket config set alphavantage_key YOUR_API_KEY",
				"guide":     "pocket setup show alphavantage",
			})
	}

	return apiKey, nil
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

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return resp, nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseInt(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}
