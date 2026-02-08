package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://discord.com/api/v10"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Guild is LLM-friendly guild output
type Guild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon,omitempty"`
	Owner       bool   `json:"owner,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
	Permissions string `json:"permissions,omitempty"`
}

// Channel is LLM-friendly channel output
type Channel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Topic    string `json:"topic,omitempty"`
	Position int    `json:"position"`
	ParentID string `json:"parent_id,omitempty"`
	NSFW     bool   `json:"nsfw,omitempty"`
}

// Message is LLM-friendly message output
type Message struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	Author    string   `json:"author"`
	AuthorID  string   `json:"author_id"`
	Timestamp string   `json:"timestamp"`
	Edited    string   `json:"edited,omitempty"`
	Pinned    bool     `json:"pinned,omitempty"`
	Mentions  []string `json:"mentions,omitempty"`
}

// SentMessage is LLM-friendly sent message response
type SentMessage struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// DMChannel is LLM-friendly DM channel response
type DMChannel struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	Recipient string `json:"recipient,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "discord",
		Aliases: []string{"dc"},
		Short:   "Discord commands",
	}

	cmd.AddCommand(newGuildsCmd())
	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newDMCmd())

	return cmd
}

func newGuildsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guilds",
		Short: "List guilds/servers the bot is in",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			reqURL := fmt.Sprintf("%s/users/@me/guilds", baseURL)
			data, err := doRequest("GET", reqURL, token, nil)
			if err != nil {
				return err
			}

			var guilds []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Icon        string `json:"icon"`
				Owner       bool   `json:"owner"`
				Permissions string `json:"permissions"`
			}

			if err := json.Unmarshal(data, &guilds); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			result := make([]Guild, 0, len(guilds))
			for _, g := range guilds {
				result = append(result, Guild{
					ID:          g.ID,
					Name:        g.Name,
					Icon:        g.Icon,
					Owner:       g.Owner,
					Permissions: g.Permissions,
				})
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newChannelsCmd() *cobra.Command {
	var channelType string

	cmd := &cobra.Command{
		Use:   "channels [guild-id]",
		Short: "List channels in a guild",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			guildID := args[0]
			reqURL := fmt.Sprintf("%s/guilds/%s/channels", baseURL, guildID)
			data, err := doRequest("GET", reqURL, token, nil)
			if err != nil {
				return err
			}

			var channels []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Type     int    `json:"type"`
				Topic    string `json:"topic"`
				Position int    `json:"position"`
				ParentID string `json:"parent_id"`
				NSFW     bool   `json:"nsfw"`
			}

			if err := json.Unmarshal(data, &channels); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			// Filter by type if specified
			typeFilter := -1
			switch channelType {
			case "text":
				typeFilter = 0
			case "voice":
				typeFilter = 2
			case "category":
				typeFilter = 4
			case "announcement", "news":
				typeFilter = 5
			case "forum":
				typeFilter = 15
			}

			result := make([]Channel, 0, len(channels))
			for _, c := range channels {
				if typeFilter >= 0 && c.Type != typeFilter {
					continue
				}

				result = append(result, Channel{
					ID:       c.ID,
					Name:     c.Name,
					Type:     channelTypeName(c.Type),
					Topic:    truncate(c.Topic, 100),
					Position: c.Position,
					ParentID: c.ParentID,
					NSFW:     c.NSFW,
				})
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVarP(&channelType, "type", "t", "", "Filter by type: text, voice, category, announcement, forum")

	return cmd
}

func newMessagesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "messages [channel-id]",
		Short: "Get recent messages from a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			// Clamp limit to Discord's max of 100
			if limit > 100 {
				limit = 100
			}
			if limit < 1 {
				limit = 50
			}

			channelID := args[0]
			reqURL := fmt.Sprintf("%s/channels/%s/messages?limit=%d", baseURL, channelID, limit)
			data, err := doRequest("GET", reqURL, token, nil)
			if err != nil {
				return err
			}

			var messages []struct {
				ID              string `json:"id"`
				Content         string `json:"content"`
				Timestamp       string `json:"timestamp"`
				EditedTimestamp string `json:"edited_timestamp"`
				Pinned          bool   `json:"pinned"`
				Author          struct {
					ID            string `json:"id"`
					Username      string `json:"username"`
					Discriminator string `json:"discriminator"`
					GlobalName    string `json:"global_name"`
				} `json:"author"`
				Mentions []struct {
					ID       string `json:"id"`
					Username string `json:"username"`
				} `json:"mentions"`
			}

			if err := json.Unmarshal(data, &messages); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			result := make([]Message, 0, len(messages))
			for _, m := range messages {
				authorName := m.Author.GlobalName
				if authorName == "" {
					authorName = m.Author.Username
				}

				mentions := make([]string, 0, len(m.Mentions))
				for _, mention := range m.Mentions {
					mentions = append(mentions, mention.Username)
				}

				result = append(result, Message{
					ID:        m.ID,
					Content:   m.Content,
					Author:    authorName,
					AuthorID:  m.Author.ID,
					Timestamp: formatTime(m.Timestamp),
					Edited:    formatTime(m.EditedTimestamp),
					Pinned:    m.Pinned,
					Mentions:  mentions,
				})
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of messages (1-100)")

	return cmd
}

func newSendCmd() *cobra.Command {
	var channelID string

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message to a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			if channelID == "" {
				return output.PrintError("missing_channel", "Channel ID is required (use --channel)", map[string]any{
					"hint": "Use 'pocket discord channels <guild-id>' to list available channels",
				})
			}

			content := args[0]
			if len(content) > 2000 {
				return output.PrintError("message_too_long", "Message exceeds 2000 character limit", map[string]any{
					"length": len(content),
					"max":    2000,
				})
			}

			reqURL := fmt.Sprintf("%s/channels/%s/messages", baseURL, channelID)
			payload := map[string]string{"content": content}
			data, err := doRequest("POST", reqURL, token, payload)
			if err != nil {
				return err
			}

			var resp struct {
				ID        string `json:"id"`
				ChannelID string `json:"channel_id"`
				Content   string `json:"content"`
				Timestamp string `json:"timestamp"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			return output.Print(SentMessage{
				ID:        resp.ID,
				ChannelID: resp.ChannelID,
				Content:   resp.Content,
				Timestamp: formatTime(resp.Timestamp),
			})
		},
	}

	cmd.Flags().StringVarP(&channelID, "channel", "c", "", "Channel ID (required)")
	cmd.MarkFlagRequired("channel")

	return cmd
}

func newDMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dm [user-id] [message]",
		Short: "Send a direct message to a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			userID := args[0]
			content := args[1]

			if len(content) > 2000 {
				return output.PrintError("message_too_long", "Message exceeds 2000 character limit", map[string]any{
					"length": len(content),
					"max":    2000,
				})
			}

			// First, create/get the DM channel
			dmURL := fmt.Sprintf("%s/users/@me/channels", baseURL)
			dmPayload := map[string]string{"recipient_id": userID}
			dmData, err := doRequest("POST", dmURL, token, dmPayload)
			if err != nil {
				return err
			}

			var dmChannel struct {
				ID   string `json:"id"`
				Type int    `json:"type"`
			}

			if err := json.Unmarshal(dmData, &dmChannel); err != nil {
				return output.PrintError("parse_failed", "Failed to create DM channel: "+err.Error(), nil)
			}

			// Now send the message to the DM channel
			msgURL := fmt.Sprintf("%s/channels/%s/messages", baseURL, dmChannel.ID)
			msgPayload := map[string]string{"content": content}
			msgData, err := doRequest("POST", msgURL, token, msgPayload)
			if err != nil {
				return err
			}

			var resp struct {
				ID        string `json:"id"`
				ChannelID string `json:"channel_id"`
				Content   string `json:"content"`
				Timestamp string `json:"timestamp"`
			}

			if err := json.Unmarshal(msgData, &resp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"status":     "sent",
				"message_id": resp.ID,
				"channel_id": resp.ChannelID,
				"user_id":    userID,
				"content":    resp.Content,
				"timestamp":  formatTime(resp.Timestamp),
			})
		},
	}

	return cmd
}

func getToken() (string, error) {
	token, err := config.Get("discord_token")
	if err != nil || token == "" {
		return "", output.PrintError("setup_required", "Discord bot token not configured", map[string]any{
			"missing":   []string{"discord_token"},
			"setup_cmd": "pocket setup show discord",
			"hint":      "Run 'pocket setup show discord' for setup instructions",
		})
	}
	return token, nil
}

func doRequest(method, reqURL, token string, payload any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, output.PrintError("request_failed", "Failed to encode payload: "+err.Error(), nil)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, output.PrintError("request_failed", err.Error(), nil)
	}

	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, output.PrintError("request_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, output.PrintError("read_failed", err.Error(), nil)
	}

	// Handle rate limiting
	if resp.StatusCode == 429 {
		var rateLimitResp struct {
			Message    string  `json:"message"`
			RetryAfter float64 `json:"retry_after"`
			Global     bool    `json:"global"`
		}
		json.Unmarshal(data, &rateLimitResp)

		return nil, output.PrintError("rate_limited", "Discord rate limit exceeded", map[string]any{
			"retry_after_seconds": rateLimitResp.RetryAfter,
			"global":              rateLimitResp.Global,
			"hint":                fmt.Sprintf("Wait %.1f seconds before retrying", rateLimitResp.RetryAfter),
		})
	}

	// Handle authentication errors
	if resp.StatusCode == 401 {
		return nil, output.PrintError("auth_failed", "Invalid Discord bot token", map[string]any{
			"hint": "Check your discord_token configuration. Ensure you're using a Bot token, not a user token.",
		})
	}

	// Handle forbidden
	if resp.StatusCode == 403 {
		var errResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.Unmarshal(data, &errResp)

		return nil, output.PrintError("forbidden", "Access denied: "+errResp.Message, map[string]any{
			"discord_code": errResp.Code,
			"hint":         "The bot may lack required permissions for this action",
		})
	}

	// Handle not found
	if resp.StatusCode == 404 {
		var errResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.Unmarshal(data, &errResp)

		return nil, output.PrintError("not_found", errResp.Message, map[string]any{
			"discord_code": errResp.Code,
		})
	}

	// Handle other errors
	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Message != "" {
			return nil, output.PrintError("api_error", errResp.Message, map[string]any{
				"status":       resp.StatusCode,
				"discord_code": errResp.Code,
			})
		}
		return nil, output.PrintError("api_error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status), nil)
	}

	return data, nil
}

func channelTypeName(t int) string {
	switch t {
	case 0:
		return "text"
	case 1:
		return "dm"
	case 2:
		return "voice"
	case 3:
		return "group_dm"
	case 4:
		return "category"
	case 5:
		return "announcement"
	case 10:
		return "announcement_thread"
	case 11:
		return "public_thread"
	case 12:
		return "private_thread"
	case 13:
		return "stage"
	case 14:
		return "directory"
	case 15:
		return "forum"
	case 16:
		return "media"
	default:
		return strconv.Itoa(t)
	}
}

func formatTime(isoTime string) string {
	if isoTime == "" {
		return ""
	}

	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		// Try parsing without timezone offset
		t, err = time.Parse("2006-01-02T15:04:05.999999", isoTime)
		if err != nil {
			return isoTime
		}
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours < 1 {
			mins := int(diff.Minutes())
			if mins < 1 {
				return "now"
			}
			return fmt.Sprintf("%dm ago", mins)
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	if diff < 30*24*time.Hour {
		return fmt.Sprintf("%dw ago", int(diff.Hours()/(24*7)))
	}
	if diff < 365*24*time.Hour {
		return fmt.Sprintf("%dmo ago", int(diff.Hours()/(24*30)))
	}
	return fmt.Sprintf("%dy ago", int(diff.Hours()/(24*365)))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
