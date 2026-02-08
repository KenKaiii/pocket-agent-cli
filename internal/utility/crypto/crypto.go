package crypto

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

const baseURL = "https://api.coingecko.com/api/v3"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Price is LLM-friendly price output
type Price struct {
	ID        string  `json:"id"`
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	PriceUSD  float64 `json:"price_usd"`
	Change24h float64 `json:"change_24h"`
	MarketCap float64 `json:"market_cap,omitempty"`
	Volume24h float64 `json:"volume_24h,omitempty"`
}

// CoinInfo is detailed coin information
type CoinInfo struct {
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Description   string  `json:"desc,omitempty"`
	PriceUSD      float64 `json:"price_usd"`
	Change24h     float64 `json:"change_24h"`
	Change7d      float64 `json:"change_7d"`
	MarketCap     float64 `json:"market_cap"`
	MarketCapRank int     `json:"rank"`
	Volume24h     float64 `json:"volume_24h"`
	High24h       float64 `json:"high_24h"`
	Low24h        float64 `json:"low_24h"`
	ATH           float64 `json:"ath"`
	ATHDate       string  `json:"ath_date,omitempty"`
	Homepage      string  `json:"homepage,omitempty"`
}

// TrendingCoin is a trending coin
type TrendingCoin struct {
	ID         string  `json:"id"`
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	MarketRank int     `json:"rank"`
	PriceBTC   float64 `json:"price_btc"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "crypto",
		Aliases: []string{"coin", "cg"},
		Short:   "Cryptocurrency commands (CoinGecko)",
	}

	cmd.AddCommand(newPriceCmd())
	cmd.AddCommand(newInfoCmd())
	cmd.AddCommand(newTopCmd())
	cmd.AddCommand(newTrendingCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

func newPriceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price [coins...]",
		Short: "Get current prices for coins (e.g., bitcoin ethereum)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := strings.Join(args, ",")
			reqURL := fmt.Sprintf("%s/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&sparkline=false",
				baseURL, url.QueryEscape(ids))

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data []struct {
				ID        string  `json:"id"`
				Symbol    string  `json:"symbol"`
				Name      string  `json:"name"`
				Price     float64 `json:"current_price"`
				Change24h float64 `json:"price_change_percentage_24h"`
				MarketCap float64 `json:"market_cap"`
				Volume24h float64 `json:"total_volume"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if len(data) == 0 {
				return output.PrintError("not_found", "No coins found", nil)
			}

			var prices []Price
			for _, p := range data {
				prices = append(prices, Price{
					ID:        p.ID,
					Symbol:    p.Symbol,
					Name:      p.Name,
					PriceUSD:  p.Price,
					Change24h: p.Change24h,
					MarketCap: p.MarketCap,
					Volume24h: p.Volume24h,
				})
			}

			return output.Print(prices)
		},
	}

	return cmd
}

func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [coin]",
		Short: "Get detailed info for a coin (e.g., bitcoin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			coinID := strings.ToLower(args[0])
			reqURL := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&community_data=false&developer_data=false",
				baseURL, url.PathEscape(coinID))

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Coin not found: "+coinID, nil)
			}

			var data struct {
				ID          string `json:"id"`
				Symbol      string `json:"symbol"`
				Name        string `json:"name"`
				Description struct {
					En string `json:"en"`
				} `json:"description"`
				Links struct {
					Homepage []string `json:"homepage"`
				} `json:"links"`
				MarketCapRank int `json:"market_cap_rank"`
				MarketData    struct {
					CurrentPrice struct {
						USD float64 `json:"usd"`
					} `json:"current_price"`
					MarketCap struct {
						USD float64 `json:"usd"`
					} `json:"market_cap"`
					TotalVolume struct {
						USD float64 `json:"usd"`
					} `json:"total_volume"`
					High24h struct {
						USD float64 `json:"usd"`
					} `json:"high_24h"`
					Low24h struct {
						USD float64 `json:"usd"`
					} `json:"low_24h"`
					PriceChange24h float64 `json:"price_change_percentage_24h"`
					PriceChange7d  float64 `json:"price_change_percentage_7d"`
					ATH            struct {
						USD float64 `json:"usd"`
					} `json:"ath"`
					ATHDate struct {
						USD string `json:"usd"`
					} `json:"ath_date"`
				} `json:"market_data"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			// Truncate description for LLM friendliness
			desc := data.Description.En
			if len(desc) > 300 {
				desc = desc[:300] + "..."
			}

			homepage := ""
			if len(data.Links.Homepage) > 0 && data.Links.Homepage[0] != "" {
				homepage = data.Links.Homepage[0]
			}

			info := CoinInfo{
				ID:            data.ID,
				Symbol:        data.Symbol,
				Name:          data.Name,
				Description:   desc,
				PriceUSD:      data.MarketData.CurrentPrice.USD,
				Change24h:     data.MarketData.PriceChange24h,
				Change7d:      data.MarketData.PriceChange7d,
				MarketCap:     data.MarketData.MarketCap.USD,
				MarketCapRank: data.MarketCapRank,
				Volume24h:     data.MarketData.TotalVolume.USD,
				High24h:       data.MarketData.High24h.USD,
				Low24h:        data.MarketData.Low24h.USD,
				ATH:           data.MarketData.ATH.USD,
				ATHDate:       data.MarketData.ATHDate.USD,
				Homepage:      homepage,
			}

			return output.Print(info)
		},
	}

	return cmd
}

func newTopCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "top",
		Short: "Get top coins by market cap",
		RunE: func(cmd *cobra.Command, args []string) error {
			reqURL := fmt.Sprintf("%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=%d&page=1&sparkline=false",
				baseURL, limit)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data []struct {
				ID            string  `json:"id"`
				Symbol        string  `json:"symbol"`
				Name          string  `json:"name"`
				CurrentPrice  float64 `json:"current_price"`
				MarketCap     float64 `json:"market_cap"`
				TotalVolume   float64 `json:"total_volume"`
				PriceChange24 float64 `json:"price_change_percentage_24h"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var prices []Price
			for _, coin := range data {
				prices = append(prices, Price{
					ID:        coin.ID,
					Symbol:    coin.Symbol,
					Name:      coin.Name,
					PriceUSD:  coin.CurrentPrice,
					Change24h: coin.PriceChange24,
					MarketCap: coin.MarketCap,
					Volume24h: coin.TotalVolume,
				})
			}

			return output.Print(prices)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of coins to return")

	return cmd
}

func newTrendingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trending",
		Short: "Get trending coins (most searched)",
		RunE: func(cmd *cobra.Command, args []string) error {
			reqURL := fmt.Sprintf("%s/search/trending", baseURL)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				Coins []struct {
					Item struct {
						ID            string  `json:"id"`
						Symbol        string  `json:"symbol"`
						Name          string  `json:"name"`
						MarketCapRank int     `json:"market_cap_rank"`
						PriceBTC      float64 `json:"price_btc"`
					} `json:"item"`
				} `json:"coins"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var trending []TrendingCoin
			for _, c := range data.Coins {
				trending = append(trending, TrendingCoin{
					ID:         c.Item.ID,
					Symbol:     c.Item.Symbol,
					Name:       c.Item.Name,
					MarketRank: c.Item.MarketCapRank,
					PriceBTC:   c.Item.PriceBTC,
				})
			}

			if len(trending) == 0 {
				return output.PrintError("not_found", "No trending coins found", nil)
			}

			return output.Print(trending)
		},
	}

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for coins by name or symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			reqURL := fmt.Sprintf("%s/search?query=%s", baseURL, url.QueryEscape(query))

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				Coins []struct {
					ID            string `json:"id"`
					Symbol        string `json:"symbol"`
					Name          string `json:"name"`
					MarketCapRank int    `json:"market_cap_rank"`
				} `json:"coins"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if len(data.Coins) == 0 {
				return output.PrintError("not_found", "No coins found for: "+query, nil)
			}

			// Limit results
			coins := data.Coins
			if len(coins) > limit {
				coins = coins[:limit]
			}

			type SearchResult struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
				Name   string `json:"name"`
				Rank   int    `json:"rank,omitempty"`
			}

			var results []SearchResult
			for _, c := range coins {
				results = append(results, SearchResult{
					ID:     c.ID,
					Symbol: c.Symbol,
					Name:   c.Name,
					Rank:   c.MarketCapRank,
				})
			}

			return output.Print(results)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Max results")

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
		return nil, output.PrintError("rate_limited", "CoinGecko rate limit exceeded, try again later", nil)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return resp, nil
}
