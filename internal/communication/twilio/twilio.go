package twilio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{}

const baseURL = "https://api.twilio.com/2010-04-01/Accounts"

// Message represents a Twilio SMS message (LLM-friendly)
type Message struct {
	SID         string `json:"sid"`
	From        string `json:"from"`
	To          string `json:"to"`
	Body        string `json:"body"`
	Status      string `json:"status"`
	Direction   string `json:"direction"`
	DateCreated string `json:"date_created"`
	DateSent    string `json:"date_sent,omitempty"`
	Price       string `json:"price,omitempty"`
	PriceUnit   string `json:"price_unit,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorMsg    string `json:"error_message,omitempty"`
}

// Account represents Twilio account info (LLM-friendly)
type Account struct {
	SID          string `json:"sid"`
	FriendlyName string `json:"friendly_name"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	DateCreated  string `json:"date_created"`
}

// SendResult represents the result of sending an SMS
type SendResult struct {
	Status   string `json:"status"`
	SID      string `json:"sid"`
	From     string `json:"from"`
	To       string `json:"to"`
	Body     string `json:"body"`
	DateSent string `json:"date_sent,omitempty"`
}

// twilioAPIMessage is the raw API response for a message
type twilioAPIMessage struct {
	SID                 string  `json:"sid"`
	AccountSID          string  `json:"account_sid"`
	From                string  `json:"from"`
	To                  string  `json:"to"`
	Body                string  `json:"body"`
	Status              string  `json:"status"`
	Direction           string  `json:"direction"`
	DateCreated         string  `json:"date_created"`
	DateSent            string  `json:"date_sent"`
	DateUpdated         string  `json:"date_updated"`
	Price               *string `json:"price"`
	PriceUnit           string  `json:"price_unit"`
	ErrorCode           *int    `json:"error_code"`
	ErrorMessage        *string `json:"error_message"`
	NumSegments         string  `json:"num_segments"`
	NumMedia            string  `json:"num_media"`
	MessagingServiceSID *string `json:"messaging_service_sid"`
}

// twilioAPIMessagesResponse is the raw API response for listing messages
type twilioAPIMessagesResponse struct {
	Messages []twilioAPIMessage `json:"messages"`
	PageSize int                `json:"page_size"`
}

// twilioAPIAccount is the raw API response for account info
type twilioAPIAccount struct {
	SID          string `json:"sid"`
	FriendlyName string `json:"friendly_name"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	DateCreated  string `json:"date_created"`
	DateUpdated  string `json:"date_updated"`
}

// twilioAPIError is the raw API error response
type twilioAPIError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "twilio",
		Aliases: []string{"sms"},
		Short:   "Twilio SMS commands",
		Long:    "Send and receive SMS messages via Twilio API",
	}

	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newMessageCmd())
	cmd.AddCommand(newAccountCmd())

	return cmd
}

func newSendCmd() *cobra.Command {
	var to string

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send an SMS message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sid, token, phone, err := getTwilioConfig()
			if err != nil {
				return err
			}

			body := args[0]

			// Prepare form data
			data := url.Values{}
			data.Set("From", phone)
			data.Set("To", to)
			data.Set("Body", body)

			// Make API request
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			apiURL := fmt.Sprintf("%s/%s/Messages.json", baseURL, sid)
			req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Authorization", "Basic "+basicAuth(sid, token))

			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_failed", err.Error(), nil)
			}

			if resp.StatusCode >= 400 {
				var apiErr twilioAPIError
				if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
					return output.PrintError("twilio_error", apiErr.Message, map[string]any{
						"code":      apiErr.Code,
						"more_info": apiErr.MoreInfo,
					})
				}
				return output.PrintError("api_error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)), nil)
			}

			var msg twilioAPIMessage
			if err := json.Unmarshal(respBody, &msg); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			result := SendResult{
				Status:   "sent",
				SID:      msg.SID,
				From:     msg.From,
				To:       msg.To,
				Body:     msg.Body,
				DateSent: formatDate(msg.DateCreated),
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipient phone number (e.g., +15551234567) (required)")
	cmd.MarkFlagRequired("to")

	return cmd
}

func newMessagesCmd() *cobra.Command {
	var limit int
	var direction string

	cmd := &cobra.Command{
		Use:   "messages",
		Short: "List recent messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			sid, token, _, err := getTwilioConfig()
			if err != nil {
				return err
			}

			// Build URL with query params
			apiURL := fmt.Sprintf("%s/%s/Messages.json?PageSize=%d", baseURL, sid, limit)
			if direction != "" {
				// Twilio uses "inbound" and "outbound-api" / "outbound-call" / "outbound-reply"
				if direction == "inbound" {
					apiURL += "&To=" + url.QueryEscape(getConfigPhone())
				} else if direction == "outbound" {
					apiURL += "&From=" + url.QueryEscape(getConfigPhone())
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			req.Header.Set("Authorization", "Basic "+basicAuth(sid, token))

			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_failed", err.Error(), nil)
			}

			if resp.StatusCode >= 400 {
				var apiErr twilioAPIError
				if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
					return output.PrintError("twilio_error", apiErr.Message, map[string]any{
						"code":      apiErr.Code,
						"more_info": apiErr.MoreInfo,
					})
				}
				return output.PrintError("api_error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil)
			}

			var apiResp twilioAPIMessagesResponse
			if err := json.Unmarshal(body, &apiResp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			messages := make([]Message, 0, len(apiResp.Messages))
			for _, m := range apiResp.Messages {
				msg := Message{
					SID:         m.SID,
					From:        m.From,
					To:          m.To,
					Body:        m.Body,
					Status:      m.Status,
					Direction:   m.Direction,
					DateCreated: formatDate(m.DateCreated),
					DateSent:    formatDate(m.DateSent),
				}
				if m.Price != nil {
					msg.Price = *m.Price
					msg.PriceUnit = m.PriceUnit
				}
				if m.ErrorCode != nil && *m.ErrorCode != 0 {
					msg.ErrorCode = fmt.Sprintf("%d", *m.ErrorCode)
				}
				if m.ErrorMessage != nil {
					msg.ErrorMsg = *m.ErrorMessage
				}
				messages = append(messages, msg)
			}

			return output.Print(messages)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of messages to retrieve")
	cmd.Flags().StringVarP(&direction, "direction", "d", "", "Filter by direction: inbound or outbound")

	return cmd
}

func newMessageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "message [sid]",
		Short: "Get message details by SID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sid, token, _, err := getTwilioConfig()
			if err != nil {
				return err
			}

			messageSID := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			apiURL := fmt.Sprintf("%s/%s/Messages/%s.json", baseURL, sid, messageSID)
			req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			req.Header.Set("Authorization", "Basic "+basicAuth(sid, token))

			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_failed", err.Error(), nil)
			}

			if resp.StatusCode >= 400 {
				var apiErr twilioAPIError
				if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
					return output.PrintError("twilio_error", apiErr.Message, map[string]any{
						"code":      apiErr.Code,
						"more_info": apiErr.MoreInfo,
					})
				}
				return output.PrintError("api_error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil)
			}

			var m twilioAPIMessage
			if err := json.Unmarshal(body, &m); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			msg := Message{
				SID:         m.SID,
				From:        m.From,
				To:          m.To,
				Body:        m.Body,
				Status:      m.Status,
				Direction:   m.Direction,
				DateCreated: formatDate(m.DateCreated),
				DateSent:    formatDate(m.DateSent),
			}
			if m.Price != nil {
				msg.Price = *m.Price
				msg.PriceUnit = m.PriceUnit
			}
			if m.ErrorCode != nil && *m.ErrorCode != 0 {
				msg.ErrorCode = fmt.Sprintf("%d", *m.ErrorCode)
			}
			if m.ErrorMessage != nil {
				msg.ErrorMsg = *m.ErrorMessage
			}

			return output.Print(msg)
		},
	}

	return cmd
}

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Get account information",
		RunE: func(cmd *cobra.Command, args []string) error {
			sid, token, phone, err := getTwilioConfig()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			apiURL := fmt.Sprintf("%s/%s.json", baseURL, sid)
			req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			req.Header.Set("Authorization", "Basic "+basicAuth(sid, token))

			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_failed", err.Error(), nil)
			}

			if resp.StatusCode >= 400 {
				var apiErr twilioAPIError
				if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
					return output.PrintError("twilio_error", apiErr.Message, map[string]any{
						"code":      apiErr.Code,
						"more_info": apiErr.MoreInfo,
					})
				}
				return output.PrintError("api_error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil)
			}

			var apiAccount twilioAPIAccount
			if err := json.Unmarshal(body, &apiAccount); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			result := map[string]any{
				"sid":           apiAccount.SID,
				"friendly_name": apiAccount.FriendlyName,
				"status":        apiAccount.Status,
				"type":          apiAccount.Type,
				"date_created":  formatDate(apiAccount.DateCreated),
				"twilio_phone":  phone,
			}

			return output.Print(result)
		},
	}

	return cmd
}

// getTwilioConfig retrieves and validates Twilio credentials
func getTwilioConfig() (sid, token, phone string, err error) {
	sid, _ = config.Get("twilio_sid")
	token, _ = config.Get("twilio_token")
	phone, _ = config.Get("twilio_phone")

	var missing []string
	if sid == "" {
		missing = append(missing, "twilio_sid")
	}
	if token == "" {
		missing = append(missing, "twilio_token")
	}
	if phone == "" {
		missing = append(missing, "twilio_phone")
	}

	if len(missing) > 0 {
		return "", "", "", output.PrintError("setup_required", "Twilio not configured", map[string]any{
			"missing":   missing,
			"setup_cmd": "pocket setup show twilio",
			"hint":      "Run 'pocket setup show twilio' for setup instructions",
		})
	}

	return sid, token, phone, nil
}

// getConfigPhone returns the configured Twilio phone number (for filtering)
func getConfigPhone() string {
	phone, _ := config.Get("twilio_phone")
	return phone
}

// basicAuth creates a base64 encoded basic auth string
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// formatDate converts Twilio date format to a cleaner format
func formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	// Twilio returns dates like: "Wed, 05 Feb 2025 14:30:00 +0000"
	t, err := time.Parse(time.RFC1123Z, dateStr)
	if err != nil {
		// Try alternative format
		t, err = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", dateStr)
		if err != nil {
			return dateStr // Return original if parsing fails
		}
	}
	return t.Format("2006-01-02 15:04:05")
}
