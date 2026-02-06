package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{}

// WebhookResponse represents the response from a webhook call
type WebhookResponse struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	URL        string `json:"url"`
	Method     string `json:"method"`
	Response   string `json:"response,omitempty"`
}

// SlackMessage represents a Slack webhook message payload
type SlackMessage struct {
	Text      string `json:"text"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

// DiscordMessage represents a Discord webhook message payload
type DiscordMessage struct {
	Content   string `json:"content"`
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"wh", "hook"},
		Short:   "Webhook commands",
		Long:    `Send data to webhook URLs. Supports generic webhooks, Slack, and Discord.`,
	}

	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newSlackCmd())
	cmd.AddCommand(newDiscordCmd())

	return cmd
}

func newSendCmd() *cobra.Command {
	var method string
	var headers []string
	var contentType string

	cmd := &cobra.Command{
		Use:   "send [url] [data]",
		Short: "Send data to a webhook URL",
		Long:  `Send a request with JSON data to any webhook URL. The data should be valid JSON.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			data := args[1]

			// Validate method
			method = strings.ToUpper(method)
			validMethods := map[string]bool{
				"GET":    true,
				"POST":   true,
				"PUT":    true,
				"DELETE": true,
			}
			if !validMethods[method] {
				return output.PrintError("invalid_method", "Method must be GET, POST, PUT, or DELETE", map[string]any{
					"provided": method,
				})
			}

			// Validate JSON data for non-GET requests
			if method != "GET" && contentType == "application/json" {
				var js json.RawMessage
				if err := json.Unmarshal([]byte(data), &js); err != nil {
					return output.PrintError("invalid_json", "Data is not valid JSON", map[string]any{
						"error": err.Error(),
						"data":  data,
					})
				}
			}

			// Build request
			var body io.Reader
			if method != "GET" {
				body = bytes.NewBufferString(data)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, method, url, body)
			if err != nil {
				return output.PrintError("request_error", "Failed to create request", map[string]any{
					"error": err.Error(),
				})
			}

			// Set content type
			if method != "GET" {
				req.Header.Set("Content-Type", contentType)
			}

			// Parse and set custom headers
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) != 2 {
					return output.PrintError("invalid_header", "Header must be in format 'Key: Value'", map[string]any{
						"header": h,
					})
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				req.Header.Set(key, value)
			}

			// Send request
			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", "Failed to send request", map[string]any{
					"error": err.Error(),
					"url":   url,
				})
			}
			defer resp.Body.Close()

			// Read response body
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_error", "Failed to read response", map[string]any{
					"error": err.Error(),
				})
			}

			// Determine status
			status := "success"
			if resp.StatusCode >= 400 {
				status = "error"
			}

			return output.Print(WebhookResponse{
				Status:     status,
				StatusCode: resp.StatusCode,
				URL:        url,
				Method:     method,
				Response:   string(respBody),
			})
		},
	}

	cmd.Flags().StringVarP(&method, "method", "m", "POST", "HTTP method (GET, POST, PUT, DELETE)")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Custom headers (format: 'Key: Value', repeatable)")
	cmd.Flags().StringVarP(&contentType, "content-type", "c", "application/json", "Content-Type header")

	return cmd
}

func newSlackCmd() *cobra.Command {
	var username string
	var iconEmoji string

	cmd := &cobra.Command{
		Use:   "slack [webhook-url] [message]",
		Short: "Send message to Slack incoming webhook",
		Long:  `Send a message to a Slack channel via incoming webhook URL.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			webhookURL := args[0]
			message := args[1]

			// Build Slack message payload
			payload := SlackMessage{
				Text: message,
			}
			if username != "" {
				payload.Username = username
			}
			if iconEmoji != "" {
				// Ensure emoji is wrapped in colons
				if !strings.HasPrefix(iconEmoji, ":") {
					iconEmoji = ":" + iconEmoji
				}
				if !strings.HasSuffix(iconEmoji, ":") {
					iconEmoji = iconEmoji + ":"
				}
				payload.IconEmoji = iconEmoji
			}

			// Marshal payload
			jsonData, err := json.Marshal(payload)
			if err != nil {
				return output.PrintError("marshal_error", "Failed to create message payload", map[string]any{
					"error": err.Error(),
				})
			}

			// Send request
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
			if err != nil {
				return output.PrintError("request_error", "Failed to create request", map[string]any{
					"error": err.Error(),
				})
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", "Failed to send message to Slack", map[string]any{
					"error": err.Error(),
					"url":   webhookURL,
				})
			}
			defer resp.Body.Close()

			// Read response
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_error", "Failed to read response", map[string]any{
					"error": err.Error(),
				})
			}

			// Slack returns "ok" on success
			respStr := string(respBody)
			if resp.StatusCode >= 400 || respStr != "ok" {
				return output.PrintError("slack_error", "Slack webhook returned error", map[string]any{
					"status_code": resp.StatusCode,
					"response":    respStr,
				})
			}

			return output.Print(map[string]any{
				"status":      "sent",
				"platform":    "slack",
				"message":     message,
				"username":    username,
				"icon_emoji":  iconEmoji,
				"status_code": resp.StatusCode,
			})
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "Override webhook username")
	cmd.Flags().StringVarP(&iconEmoji, "icon-emoji", "i", "", "Override webhook icon emoji (e.g., :robot:)")

	return cmd
}

func newDiscordCmd() *cobra.Command {
	var username string
	var avatarURL string

	cmd := &cobra.Command{
		Use:   "discord [webhook-url] [message]",
		Short: "Send message to Discord webhook",
		Long:  `Send a message to a Discord channel via webhook URL.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			webhookURL := args[0]
			message := args[1]

			// Validate message length (Discord limit is 2000 characters)
			if len(message) > 2000 {
				return output.PrintError("message_too_long", "Discord messages must be 2000 characters or less", map[string]any{
					"length": len(message),
					"limit":  2000,
				})
			}

			// Build Discord message payload
			payload := DiscordMessage{
				Content: message,
			}
			if username != "" {
				payload.Username = username
			}
			if avatarURL != "" {
				payload.AvatarURL = avatarURL
			}

			// Marshal payload
			jsonData, err := json.Marshal(payload)
			if err != nil {
				return output.PrintError("marshal_error", "Failed to create message payload", map[string]any{
					"error": err.Error(),
				})
			}

			// Send request
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
			if err != nil {
				return output.PrintError("request_error", "Failed to create request", map[string]any{
					"error": err.Error(),
				})
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := httpClient.Do(req)
			if err != nil {
				return output.PrintError("request_failed", "Failed to send message to Discord", map[string]any{
					"error": err.Error(),
					"url":   webhookURL,
				})
			}
			defer resp.Body.Close()

			// Read response
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("read_error", "Failed to read response", map[string]any{
					"error": err.Error(),
				})
			}

			// Discord returns 204 No Content on success, or 200 with message details
			if resp.StatusCode >= 400 {
				var errResp map[string]any
				if err := json.Unmarshal(respBody, &errResp); err == nil {
					return output.PrintError("discord_error", fmt.Sprintf("Discord webhook returned error: %v", errResp["message"]), map[string]any{
						"status_code": resp.StatusCode,
						"response":    errResp,
					})
				}
				return output.PrintError("discord_error", "Discord webhook returned error", map[string]any{
					"status_code": resp.StatusCode,
					"response":    string(respBody),
				})
			}

			return output.Print(map[string]any{
				"status":      "sent",
				"platform":    "discord",
				"message":     message,
				"username":    username,
				"avatar_url":  avatarURL,
				"status_code": resp.StatusCode,
			})
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "Override webhook username")
	cmd.Flags().StringVarP(&avatarURL, "avatar-url", "a", "", "Override webhook avatar URL")

	return cmd
}
