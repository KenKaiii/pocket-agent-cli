package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://slack.com/api"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Channel is LLM-friendly channel output
type Channel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsPrivate  bool   `json:"is_private"`
	IsArchived bool   `json:"is_archived,omitempty"`
	IsMember   bool   `json:"is_member,omitempty"`
	NumMembers int    `json:"num_members,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Purpose    string `json:"purpose,omitempty"`
}

// Message is LLM-friendly message output
type Message struct {
	TS       string `json:"ts"`
	User     string `json:"user,omitempty"`
	UserName string `json:"user_name,omitempty"`
	Text     string `json:"text"`
	Type     string `json:"type,omitempty"`
	Time     string `json:"time,omitempty"`
	ThreadTS string `json:"thread_ts,omitempty"`
	Edited   bool   `json:"edited,omitempty"`
}

// User is LLM-friendly user output
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RealName    string `json:"real_name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
	IsBot       bool   `json:"is_bot,omitempty"`
	IsAdmin     bool   `json:"is_admin,omitempty"`
	Status      string `json:"status,omitempty"`
	StatusEmoji string `json:"status_emoji,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
}

// SearchResult is LLM-friendly search result output
type SearchResult struct {
	Channel   string `json:"channel"`
	User      string `json:"user"`
	UserName  string `json:"user_name,omitempty"`
	Text      string `json:"text"`
	TS        string `json:"ts"`
	Time      string `json:"time,omitempty"`
	Permalink string `json:"permalink,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slack",
		Short: "Slack commands",
	}

	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newUsersCmd())
	cmd.AddCommand(newDMCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

func newChannelsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "channels",
		Short: "List channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("limit", fmt.Sprintf("%d", limit))
			params.Set("types", "public_channel,private_channel")
			params.Set("exclude_archived", "true")

			var resp struct {
				OK       bool   `json:"ok"`
				Error    string `json:"error,omitempty"`
				Channels []struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					IsPrivate  bool   `json:"is_private"`
					IsArchived bool   `json:"is_archived"`
					IsMember   bool   `json:"is_member"`
					NumMembers int    `json:"num_members"`
					Topic      struct {
						Value string `json:"value"`
					} `json:"topic"`
					Purpose struct {
						Value string `json:"value"`
					} `json:"purpose"`
				} `json:"channels"`
			}

			if err := slackGet(token, "conversations.list", params, &resp); err != nil {
				return err
			}

			if !resp.OK {
				return output.PrintError("api_error", resp.Error, map[string]any{
					"hint": getErrorHint(resp.Error),
				})
			}

			channels := make([]Channel, 0, len(resp.Channels))
			for _, ch := range resp.Channels {
				channels = append(channels, Channel{
					ID:         ch.ID,
					Name:       ch.Name,
					IsPrivate:  ch.IsPrivate,
					IsArchived: ch.IsArchived,
					IsMember:   ch.IsMember,
					NumMembers: ch.NumMembers,
					Topic:      truncate(ch.Topic.Value, 100),
					Purpose:    truncate(ch.Purpose.Value, 100),
				})
			}

			return output.Print(channels)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Number of channels")

	return cmd
}

func newMessagesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "messages [channel]",
		Short: "Get channel messages",
		Long:  "Get messages from a channel. Channel can be channel ID (C...) or name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			channelID := args[0]

			// If it doesn't look like a channel ID, try to resolve the name
			if len(channelID) > 0 && channelID[0] != 'C' && channelID[0] != 'G' && channelID[0] != 'D' {
				resolved, err := resolveChannelID(token, channelID)
				if err != nil {
					return output.PrintError("channel_not_found", "Could not find channel: "+channelID, map[string]any{
						"hint": "Use channel ID (starts with C) or exact channel name",
					})
				}
				channelID = resolved
			}

			params := url.Values{}
			params.Set("channel", channelID)
			params.Set("limit", fmt.Sprintf("%d", limit))

			var resp struct {
				OK       bool   `json:"ok"`
				Error    string `json:"error,omitempty"`
				Messages []struct {
					Type     string `json:"type"`
					User     string `json:"user"`
					Text     string `json:"text"`
					TS       string `json:"ts"`
					ThreadTS string `json:"thread_ts,omitempty"`
					Edited   *struct {
						TS string `json:"ts"`
					} `json:"edited,omitempty"`
				} `json:"messages"`
			}

			if err := slackGet(token, "conversations.history", params, &resp); err != nil {
				return err
			}

			if !resp.OK {
				return output.PrintError("api_error", resp.Error, map[string]any{
					"hint": getErrorHint(resp.Error),
				})
			}

			// Fetch user info to get display names
			userCache := make(map[string]string)

			messages := make([]Message, 0, len(resp.Messages))
			for _, msg := range resp.Messages {
				userName := msg.User
				if msg.User != "" {
					if cached, ok := userCache[msg.User]; ok {
						userName = cached
					} else if name, err := getUserName(token, msg.User); err == nil {
						userName = name
						userCache[msg.User] = name
					}
				}

				messages = append(messages, Message{
					TS:       msg.TS,
					User:     msg.User,
					UserName: userName,
					Text:     msg.Text,
					Type:     msg.Type,
					Time:     formatSlackTime(msg.TS),
					ThreadTS: msg.ThreadTS,
					Edited:   msg.Edited != nil,
				})
			}

			return output.Print(messages)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of messages")

	return cmd
}

func newSendCmd() *cobra.Command {
	var channel string
	var threadTS string

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message to a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			channelID := channel

			// If it doesn't look like a channel ID, try to resolve the name
			if len(channelID) > 0 && channelID[0] != 'C' && channelID[0] != 'G' && channelID[0] != 'D' {
				resolved, err := resolveChannelID(token, channelID)
				if err != nil {
					return output.PrintError("channel_not_found", "Could not find channel: "+channelID, map[string]any{
						"hint": "Use channel ID (starts with C) or exact channel name",
					})
				}
				channelID = resolved
			}

			body := map[string]string{
				"channel": channelID,
				"text":    args[0],
			}
			if threadTS != "" {
				body["thread_ts"] = threadTS
			}

			var resp struct {
				OK      bool   `json:"ok"`
				Error   string `json:"error,omitempty"`
				Channel string `json:"channel"`
				TS      string `json:"ts"`
				Message struct {
					Text string `json:"text"`
					TS   string `json:"ts"`
				} `json:"message"`
			}

			if err := slackPost(token, "chat.postMessage", body, &resp); err != nil {
				return err
			}

			if !resp.OK {
				return output.PrintError("api_error", resp.Error, map[string]any{
					"hint": getErrorHint(resp.Error),
				})
			}

			return output.Print(map[string]any{
				"status":    "sent",
				"channel":   resp.Channel,
				"ts":        resp.TS,
				"text":      resp.Message.Text,
				"thread_ts": threadTS,
			})
		},
	}

	cmd.Flags().StringVarP(&channel, "channel", "c", "", "Channel ID or name (required)")
	cmd.Flags().StringVarP(&threadTS, "thread", "t", "", "Thread timestamp (for replies)")
	cmd.MarkFlagRequired("channel")

	return cmd
}

func newUsersCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "users",
		Short: "List workspace users",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("limit", fmt.Sprintf("%d", limit))

			var resp struct {
				OK      bool   `json:"ok"`
				Error   string `json:"error,omitempty"`
				Members []struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					RealName string `json:"real_name"`
					Deleted  bool   `json:"deleted"`
					IsBot    bool   `json:"is_bot"`
					IsAdmin  bool   `json:"is_admin"`
					Profile  struct {
						DisplayName string `json:"display_name"`
						Email       string `json:"email"`
						StatusText  string `json:"status_text"`
						StatusEmoji string `json:"status_emoji"`
					} `json:"profile"`
					TZ string `json:"tz"`
				} `json:"members"`
			}

			if err := slackGet(token, "users.list", params, &resp); err != nil {
				return err
			}

			if !resp.OK {
				return output.PrintError("api_error", resp.Error, map[string]any{
					"hint": getErrorHint(resp.Error),
				})
			}

			users := make([]User, 0, len(resp.Members))
			for _, m := range resp.Members {
				if m.Deleted {
					continue
				}
				users = append(users, User{
					ID:          m.ID,
					Name:        m.Name,
					RealName:    m.RealName,
					DisplayName: m.Profile.DisplayName,
					Email:       m.Profile.Email,
					IsBot:       m.IsBot,
					IsAdmin:     m.IsAdmin,
					Status:      m.Profile.StatusText,
					StatusEmoji: m.Profile.StatusEmoji,
					Timezone:    m.TZ,
				})
			}

			return output.Print(users)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Number of users")

	return cmd
}

func newDMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dm [user-id] [message]",
		Short: "Send a direct message",
		Long:  "Send a direct message to a user. User can be user ID (U...) or username.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			userID := args[0]
			message := args[1]

			// If it doesn't look like a user ID, try to resolve the username
			if len(userID) > 0 && userID[0] != 'U' && userID[0] != 'W' {
				resolved, err := resolveUserID(token, userID)
				if err != nil {
					return output.PrintError("user_not_found", "Could not find user: "+userID, map[string]any{
						"hint": "Use user ID (starts with U) or exact username",
					})
				}
				userID = resolved
			}

			// Open DM channel
			openBody := map[string]string{
				"users": userID,
			}

			var openResp struct {
				OK      bool   `json:"ok"`
				Error   string `json:"error,omitempty"`
				Channel struct {
					ID string `json:"id"`
				} `json:"channel"`
			}

			if err := slackPost(token, "conversations.open", openBody, &openResp); err != nil {
				return err
			}

			if !openResp.OK {
				return output.PrintError("api_error", openResp.Error, map[string]any{
					"hint": getErrorHint(openResp.Error),
				})
			}

			// Send message
			sendBody := map[string]string{
				"channel": openResp.Channel.ID,
				"text":    message,
			}

			var sendResp struct {
				OK      bool   `json:"ok"`
				Error   string `json:"error,omitempty"`
				Channel string `json:"channel"`
				TS      string `json:"ts"`
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
			}

			if err := slackPost(token, "chat.postMessage", sendBody, &sendResp); err != nil {
				return err
			}

			if !sendResp.OK {
				return output.PrintError("api_error", sendResp.Error, map[string]any{
					"hint": getErrorHint(sendResp.Error),
				})
			}

			return output.Print(map[string]any{
				"status":  "sent",
				"user_id": userID,
				"channel": sendResp.Channel,
				"ts":      sendResp.TS,
				"text":    sendResp.Message.Text,
			})
		},
	}

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search messages",
		Long:  "Search messages in the workspace. Requires a user token (xoxp-*) with search:read scope.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("query", args[0])
			params.Set("count", fmt.Sprintf("%d", limit))
			params.Set("sort", "timestamp")
			params.Set("sort_dir", "desc")

			var resp struct {
				OK       bool   `json:"ok"`
				Error    string `json:"error,omitempty"`
				Messages struct {
					Total   int `json:"total"`
					Matches []struct {
						Channel struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"channel"`
						User      string `json:"user"`
						Username  string `json:"username"`
						Text      string `json:"text"`
						TS        string `json:"ts"`
						Permalink string `json:"permalink"`
					} `json:"matches"`
				} `json:"messages"`
			}

			if err := slackGet(token, "search.messages", params, &resp); err != nil {
				return err
			}

			if !resp.OK {
				return output.PrintError("api_error", resp.Error, map[string]any{
					"hint": getErrorHint(resp.Error),
				})
			}

			results := make([]SearchResult, 0, len(resp.Messages.Matches))
			for _, m := range resp.Messages.Matches {
				results = append(results, SearchResult{
					Channel:   m.Channel.Name,
					User:      m.User,
					UserName:  m.Username,
					Text:      truncate(m.Text, 300),
					TS:        m.TS,
					Time:      formatSlackTime(m.TS),
					Permalink: m.Permalink,
				})
			}

			return output.Print(map[string]any{
				"total":   resp.Messages.Total,
				"results": results,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")

	return cmd
}

// getToken retrieves the Slack token from config
func getToken() (string, error) {
	token, err := config.MustGet("slack_token")
	if err != nil {
		return "", output.PrintError("auth_required", "Slack token not configured", map[string]any{
			"setup_cmd": "pocket config set slack_token <your-token>",
			"hint":      "Get a Bot token (xoxb-*) or User token (xoxp-*) from https://api.slack.com/apps",
		})
	}
	return token, nil
}

// slackGet makes a GET request to the Slack API
func slackGet(token, method string, params url.Values, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/%s?%s", baseURL, method, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return output.PrintError("request_failed", err.Error(), nil)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("request_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		return output.PrintError("rate_limited", "Slack API rate limit exceeded", map[string]any{
			"retry_after": retryAfter,
			"hint":        fmt.Sprintf("Wait %s seconds before retrying", retryAfter),
		})
	}

	if resp.StatusCode >= 400 {
		return output.PrintError("http_error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status), nil)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// slackPost makes a POST request to the Slack API
func slackPost(token, method string, body map[string]string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/%s", baseURL, method)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return output.PrintError("request_failed", err.Error(), nil)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return output.PrintError("request_failed", err.Error(), nil)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("request_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		return output.PrintError("rate_limited", "Slack API rate limit exceeded", map[string]any{
			"retry_after": retryAfter,
			"hint":        fmt.Sprintf("Wait %s seconds before retrying", retryAfter),
		})
	}

	if resp.StatusCode >= 400 {
		return output.PrintError("http_error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status), nil)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// resolveChannelID resolves a channel name to its ID
func resolveChannelID(token, name string) (string, error) {
	params := url.Values{}
	params.Set("limit", "1000")
	params.Set("types", "public_channel,private_channel")

	var resp struct {
		OK       bool `json:"ok"`
		Channels []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"channels"`
	}

	if err := slackGet(token, "conversations.list", params, &resp); err != nil {
		return "", err
	}

	for _, ch := range resp.Channels {
		if ch.Name == name {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel not found: %s", name)
}

// resolveUserID resolves a username to its ID
func resolveUserID(token, name string) (string, error) {
	params := url.Values{}
	params.Set("limit", "1000")

	var resp struct {
		OK      bool `json:"ok"`
		Members []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"members"`
	}

	if err := slackGet(token, "users.list", params, &resp); err != nil {
		return "", err
	}

	for _, u := range resp.Members {
		if u.Name == name {
			return u.ID, nil
		}
	}

	return "", fmt.Errorf("user not found: %s", name)
}

// getUserName gets the display name for a user ID
func getUserName(token, userID string) (string, error) {
	params := url.Values{}
	params.Set("user", userID)

	var resp struct {
		OK   bool `json:"ok"`
		User struct {
			Name     string `json:"name"`
			RealName string `json:"real_name"`
			Profile  struct {
				DisplayName string `json:"display_name"`
			} `json:"profile"`
		} `json:"user"`
	}

	if err := slackGet(token, "users.info", params, &resp); err != nil {
		return "", err
	}

	if !resp.OK {
		return "", fmt.Errorf("user not found")
	}

	// Prefer display name, then real name, then username
	if resp.User.Profile.DisplayName != "" {
		return resp.User.Profile.DisplayName, nil
	}
	if resp.User.RealName != "" {
		return resp.User.RealName, nil
	}
	return resp.User.Name, nil
}

// formatSlackTime converts a Slack timestamp to a readable format
func formatSlackTime(ts string) string {
	var secs float64
	_, err := fmt.Sscanf(ts, "%f", &secs)
	if err != nil {
		return ""
	}

	t := time.Unix(int64(secs), 0)
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return t.Format("Mon 15:04")
	default:
		return t.Format("Jan 2, 15:04")
	}
}

// getErrorHint provides helpful hints for common Slack API errors
func getErrorHint(errCode string) string {
	hints := map[string]string{
		"not_authed":              "Token is missing or invalid. Check your slack_token configuration.",
		"invalid_auth":            "Token is invalid or expired. Generate a new token from https://api.slack.com/apps",
		"token_revoked":           "Token has been revoked. Generate a new token from https://api.slack.com/apps",
		"channel_not_found":       "Channel does not exist or bot is not a member. Invite the bot to the channel first.",
		"not_in_channel":          "Bot is not a member of this channel. Invite the bot to the channel first.",
		"is_archived":             "This channel has been archived and cannot be modified.",
		"user_not_found":          "User does not exist in this workspace.",
		"missing_scope":           "Token is missing required OAuth scopes. Check app permissions at https://api.slack.com/apps",
		"ratelimited":             "Rate limited by Slack. Wait before retrying.",
		"no_permission":           "Token does not have permission for this action. Check app permissions.",
		"cant_dm_bot":             "Cannot send direct messages to bots.",
		"method_not_allowed":      "This method is not allowed for your token type.",
		"ekm_access_denied":       "Enterprise Key Management prevented this request.",
		"org_login_required":      "Organization login required. Check your workspace settings.",
		"team_access_not_granted": "Workspace admin has not granted access to this app.",
	}

	if hint, ok := hints[errCode]; ok {
		return hint
	}
	return "Check Slack API documentation for error: " + errCode
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
