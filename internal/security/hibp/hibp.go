package hibp

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// PasswordResult represents the result of a password breach check
type PasswordResult struct {
	Compromised  bool   `json:"compromised"`
	TimesExposed int    `json:"times_exposed"`
	Message      string `json:"message"`
}

// Breach represents a public data breach
type Breach struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Domain      string   `json:"domain"`
	BreachDate  string   `json:"breach_date"`
	PwnCount    int64    `json:"pwn_count"`
	Description string   `json:"description"`
	DataClasses []string `json:"data_classes"`
	IsVerified  bool     `json:"is_verified"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hibp",
		Aliases: []string{"pwned", "breach"},
		Short:   "Have I Been Pwned checks",
		Long:    "Have I Been Pwned integration for password breach checks and public breach listings.",
	}

	cmd.AddCommand(newPasswordCmd())
	cmd.AddCommand(newBreachesCmd())

	return cmd
}

func newPasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "password [password]",
		Short: "Check if a password has been exposed in data breaches",
		Long:  "Uses k-anonymity to safely check if a password appears in known data breaches without sending the full password to the API.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			password := args[0]

			// SHA1 hash the password
			hasher := sha1.New()
			hasher.Write([]byte(password))
			hash := strings.ToUpper(hex.EncodeToString(hasher.Sum(nil)))

			// Split into prefix (first 5 chars) and suffix (rest)
			prefix := hash[:5]
			suffix := hash[5:]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			reqURL := fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", prefix)
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
				return output.PrintError("api_error", fmt.Sprintf("HIBP API error (HTTP %d): %s", resp.StatusCode, string(body)), nil)
			}

			// Parse response: each line is SUFFIX:COUNT
			lines := strings.Split(strings.TrimSpace(string(body)), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					continue
				}

				if strings.ToUpper(parts[0]) == suffix {
					count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
					if err != nil {
						count = 0
					}

					result := PasswordResult{
						Compromised:  true,
						TimesExposed: count,
						Message:      "This password has been exposed in data breaches",
					}
					return output.Print(result)
				}
			}

			result := PasswordResult{
				Compromised:  false,
				TimesExposed: 0,
				Message:      "This password was not found in known data breaches",
			}
			return output.Print(result)
		},
	}

	return cmd
}

func newBreachesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "breaches",
		Short: "List recent public data breaches",
		Long:  "Retrieve a list of all public data breaches from Have I Been Pwned.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			reqURL := "https://haveibeenpwned.com/api/v3/breaches"
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
				return output.PrintError("api_error", fmt.Sprintf("HIBP API error (HTTP %d): %s", resp.StatusCode, string(body)), nil)
			}

			var rawBreaches []struct {
				Name        string   `json:"Name"`
				Title       string   `json:"Title"`
				Domain      string   `json:"Domain"`
				BreachDate  string   `json:"BreachDate"`
				PwnCount    int64    `json:"PwnCount"`
				Description string   `json:"Description"`
				DataClasses []string `json:"DataClasses"`
				IsVerified  bool     `json:"IsVerified"`
			}

			if err := json.Unmarshal(body, &rawBreaches); err != nil {
				return output.PrintError("parse_error", "failed to parse breaches response", nil)
			}

			// Apply limit
			if limit > 0 && len(rawBreaches) > limit {
				rawBreaches = rawBreaches[:limit]
			}

			results := make([]Breach, len(rawBreaches))
			for i, raw := range rawBreaches {
				dataClasses := raw.DataClasses
				if dataClasses == nil {
					dataClasses = []string{}
				}
				results[i] = Breach{
					Name:        raw.Name,
					Title:       raw.Title,
					Domain:      raw.Domain,
					BreachDate:  raw.BreachDate,
					PwnCount:    raw.PwnCount,
					Description: raw.Description,
					DataClasses: dataClasses,
					IsVerified:  raw.IsVerified,
				}
			}

			return output.Print(results)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of breaches to return")

	return cmd
}
