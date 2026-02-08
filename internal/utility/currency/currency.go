package currency

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

const baseURL = "https://api.frankfurter.app"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ExchangeRate is LLM-friendly exchange rate output
type ExchangeRate struct {
	From string  `json:"from"`
	To   string  `json:"to"`
	Rate float64 `json:"rate"`
	Date string  `json:"date"`
}

// Conversion is LLM-friendly conversion result
type Conversion struct {
	Amount float64 `json:"amount"`
	From   string  `json:"from"`
	To     string  `json:"to"`
	Rate   float64 `json:"rate"`
	Result float64 `json:"result"`
	Date   string  `json:"date"`
}

// Currency represents a supported currency
type Currency struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "currency",
		Aliases: []string{"fx", "forex"},
		Short:   "Currency exchange commands",
	}

	cmd.AddCommand(newRateCmd())
	cmd.AddCommand(newConvertCmd())
	cmd.AddCommand(newListCmd())

	return cmd
}

func newRateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate [from] [to]",
		Short: "Get exchange rate between two currencies (e.g., USD EUR)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			from := strings.ToUpper(args[0])
			to := strings.ToUpper(args[1])

			reqURL := fmt.Sprintf("%s/latest?from=%s&to=%s", baseURL, from, to)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				Base   string             `json:"base"`
				Date   string             `json:"date"`
				Rates  map[string]float64 `json:"rates"`
				Amount float64            `json:"amount"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			rate, ok := data.Rates[to]
			if !ok {
				return output.PrintError("not_found", "Currency not found: "+to, nil)
			}

			result := ExchangeRate{
				From: from,
				To:   to,
				Rate: rate,
				Date: data.Date,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newConvertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert [amount] [from] [to]",
		Short: "Convert amount between currencies (e.g., 100 USD EUR)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return output.PrintError("invalid_amount", "Amount must be a number", nil)
			}

			from := strings.ToUpper(args[1])
			to := strings.ToUpper(args[2])

			reqURL := fmt.Sprintf("%s/latest?amount=%f&from=%s&to=%s", baseURL, amount, from, to)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data struct {
				Base   string             `json:"base"`
				Date   string             `json:"date"`
				Rates  map[string]float64 `json:"rates"`
				Amount float64            `json:"amount"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			converted, ok := data.Rates[to]
			if !ok {
				return output.PrintError("not_found", "Currency not found: "+to, nil)
			}

			// Calculate the rate (converted / amount)
			rate := converted / amount

			result := Conversion{
				Amount: amount,
				From:   from,
				To:     to,
				Rate:   rate,
				Result: converted,
				Date:   data.Date,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available currencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			reqURL := fmt.Sprintf("%s/currencies", baseURL)

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data map[string]string

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var currencies []Currency
			for code, name := range data {
				currencies = append(currencies, Currency{
					Code: code,
					Name: name,
				})
			}

			if len(currencies) == 0 {
				return output.PrintError("not_found", "No currencies found", nil)
			}

			return output.Print(currencies)
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
		return nil, output.PrintError("rate_limited", "Rate limit exceeded, try again later", nil)
	}

	if resp.StatusCode == 404 {
		resp.Body.Close()
		return nil, output.PrintError("not_found", "Currency not found or invalid", nil)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return resp, nil
}
