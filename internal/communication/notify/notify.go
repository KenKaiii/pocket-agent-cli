package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// NtfyResponse represents the response from ntfy.sh
type NtfyResponse struct {
	ID      string `json:"id"`
	Time    int64  `json:"time"`
	Event   string `json:"event"`
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

// PushoverResponse represents the response from Pushover API
type PushoverResponse struct {
	Status  int    `json:"status"`
	Request string `json:"request"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notify",
		Aliases: []string{"push", "alert"},
		Short:   "Push notification commands",
		Long:    `Send push notifications via ntfy.sh, Pushover, and other services.`,
	}

	cmd.AddCommand(newNtfyCmd())
	cmd.AddCommand(newPushoverCmd())

	return cmd
}

func newNtfyCmd() *cobra.Command {
	var title string
	var priority int
	var tags string

	cmd := &cobra.Command{
		Use:   "ntfy [topic] [message]",
		Short: "Send notification via ntfy.sh (no auth required)",
		Long: `Send a push notification via ntfy.sh.

No authentication required for public topics.
Subscribe to your topic at https://ntfy.sh/<topic> or via the ntfy app.

Examples:
  pocket notify ntfy mytopic "Hello World"
  pocket notify ntfy alerts "Server down!" --title "Alert" --priority 5
  pocket notify ntfy updates "New release" --tags "tada,release"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := args[0]
			message := args[1]

			return sendNtfy(topic, message, title, priority, tags)
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Notification title")
	cmd.Flags().IntVarP(&priority, "priority", "p", 3, "Priority (1=min, 2=low, 3=default, 4=high, 5=max)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags/emojis (e.g., 'warning,skull')")

	return cmd
}

func newPushoverCmd() *cobra.Command {
	var title string
	var priority int
	var sound string

	cmd := &cobra.Command{
		Use:   "pushover [message]",
		Short: "Send notification via Pushover (requires config)",
		Long: `Send a push notification via Pushover.

Requires pushover_token and pushover_user to be configured.
Get your API token at https://pushover.net/apps/build
Get your user key at https://pushover.net/

Examples:
  pocket notify pushover "Hello World"
  pocket notify pushover "Alert!" --title "Warning" --priority 1
  pocket notify pushover "Alarm" --sound siren --priority 2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := args[0]

			token, err := config.MustGet("pushover_token")
			if err != nil {
				return output.PrintError("config_missing", "Pushover API token not configured", map[string]any{
					"key":       "pushover_token",
					"hint":      "Run: pocket config set pushover_token <your-api-token>",
					"get_key":   "https://pushover.net/apps/build",
					"setup_cmd": "pocket setup show pushover",
				})
			}

			user, err := config.MustGet("pushover_user")
			if err != nil {
				return output.PrintError("config_missing", "Pushover user key not configured", map[string]any{
					"key":       "pushover_user",
					"hint":      "Run: pocket config set pushover_user <your-user-key>",
					"get_key":   "https://pushover.net/",
					"setup_cmd": "pocket setup show pushover",
				})
			}

			return sendPushover(token, user, message, title, priority, sound)
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Notification title")
	cmd.Flags().IntVarP(&priority, "priority", "p", 0, "Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)")
	cmd.Flags().StringVarP(&sound, "sound", "s", "", "Notification sound (e.g., pushover, bike, bugle, siren)")

	return cmd
}

func sendNtfy(topic, message, title string, priority int, tags string) error {
	url := fmt.Sprintf("https://ntfy.sh/%s", topic)

	// Validate priority
	if priority < 1 || priority > 5 {
		return output.PrintError("invalid_priority", "Priority must be between 1 and 5", map[string]any{
			"provided": priority,
			"valid":    "1 (min), 2 (low), 3 (default), 4 (high), 5 (max)",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(message))
	if err != nil {
		return output.PrintError("request_failed", err.Error(), nil)
	}

	// Set headers
	if title != "" {
		req.Header.Set("Title", title)
	}
	if priority != 3 {
		req.Header.Set("Priority", fmt.Sprintf("%d", priority))
	}
	if tags != "" {
		req.Header.Set("Tags", tags)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("send_failed", err.Error(), map[string]any{
			"topic": topic,
			"hint":  "Check your internet connection",
		})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return output.PrintError("send_failed", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), map[string]any{
			"topic":       topic,
			"status_code": resp.StatusCode,
		})
	}

	var ntfyResp NtfyResponse
	if err := json.Unmarshal(body, &ntfyResp); err != nil {
		// Even if we can't parse the response, the notification was sent
		return output.Print(map[string]any{
			"status":   "sent",
			"service":  "ntfy",
			"topic":    topic,
			"message":  message,
			"title":    title,
			"priority": priority,
			"tags":     tags,
		})
	}

	return output.Print(map[string]any{
		"status":   "sent",
		"service":  "ntfy",
		"id":       ntfyResp.ID,
		"topic":    ntfyResp.Topic,
		"message":  ntfyResp.Message,
		"title":    title,
		"priority": priority,
		"tags":     tags,
		"time":     ntfyResp.Time,
	})
}

func sendPushover(token, user, message, title string, priority int, sound string) error {
	url := "https://api.pushover.net/1/messages.json"

	// Validate priority
	if priority < -2 || priority > 2 {
		return output.PrintError("invalid_priority", "Priority must be between -2 and 2", map[string]any{
			"provided": priority,
			"valid":    "-2 (lowest), -1 (low), 0 (normal), 1 (high), 2 (emergency)",
		})
	}

	// Build form data
	data := map[string]string{
		"token":   token,
		"user":    user,
		"message": message,
	}

	if title != "" {
		data["title"] = title
	}
	if priority != 0 {
		data["priority"] = fmt.Sprintf("%d", priority)
	}
	if sound != "" {
		data["sound"] = sound
	}

	// Emergency priority requires retry and expire parameters
	if priority == 2 {
		data["retry"] = "60"
		data["expire"] = "3600"
	}

	// Encode as JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return output.PrintError("encode_failed", err.Error(), nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return output.PrintError("request_failed", err.Error(), nil)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("send_failed", err.Error(), map[string]any{
			"hint": "Check your internet connection",
		})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.Unmarshal(body, &errResp)
		return output.PrintError("send_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), map[string]any{
			"status_code": resp.StatusCode,
			"response":    errResp,
			"hint":        "Check your pushover_token and pushover_user configuration",
		})
	}

	var pushResp PushoverResponse
	if err := json.Unmarshal(body, &pushResp); err != nil {
		return output.Print(map[string]any{
			"status":   "sent",
			"service":  "pushover",
			"message":  message,
			"title":    title,
			"priority": priority,
			"sound":    sound,
		})
	}

	return output.Print(map[string]any{
		"status":     "sent",
		"service":    "pushover",
		"request_id": pushResp.Request,
		"message":    message,
		"title":      title,
		"priority":   priority,
		"sound":      sound,
	})
}
