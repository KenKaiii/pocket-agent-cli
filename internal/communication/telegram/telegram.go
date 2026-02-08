package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://api.telegram.org/bot"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// BotInfo is LLM-friendly bot information
type BotInfo struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	Username                string `json:"username"`
	CanJoinGroups           bool   `json:"can_join_groups"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries"`
}

// Chat is LLM-friendly chat information
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// User is LLM-friendly user information
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Message is LLM-friendly message information
type Message struct {
	MessageID int64  `json:"message_id"`
	Date      string `json:"date"`
	Chat      Chat   `json:"chat"`
	From      *User  `json:"from,omitempty"`
	Text      string `json:"text,omitempty"`
	Caption   string `json:"caption,omitempty"`
}

// Update is LLM-friendly update information
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// SendResult is LLM-friendly send result
type SendResult struct {
	MessageID int64  `json:"message_id"`
	ChatID    int64  `json:"chat_id"`
	Date      string `json:"date"`
	Text      string `json:"text"`
}

// ForwardResult is LLM-friendly forward result
type ForwardResult struct {
	MessageID         int64  `json:"message_id"`
	FromChatID        int64  `json:"from_chat_id"`
	ToChatID          int64  `json:"to_chat_id"`
	OriginalMessageID int64  `json:"original_message_id"`
	Date              string `json:"date"`
}

// API response structures
type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	Description string          `json:"description,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Parameters  *responseParams `json:"parameters,omitempty"`
}

type responseParams struct {
	RetryAfter int `json:"retry_after,omitempty"`
}

type apiUser struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name,omitempty"`
	Username                string `json:"username,omitempty"`
	CanJoinGroups           bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries,omitempty"`
}

type apiChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type apiMessage struct {
	MessageID int64    `json:"message_id"`
	Date      int64    `json:"date"`
	Chat      apiChat  `json:"chat"`
	From      *apiUser `json:"from,omitempty"`
	Text      string   `json:"text,omitempty"`
	Caption   string   `json:"caption,omitempty"`
}

type apiUpdate struct {
	UpdateID int64       `json:"update_id"`
	Message  *apiMessage `json:"message,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "telegram",
		Aliases: []string{"tg"},
		Short:   "Telegram Bot API commands",
	}

	cmd.AddCommand(newMeCmd())
	cmd.AddCommand(newUpdatesCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newChatsCmd())
	cmd.AddCommand(newForwardCmd())

	return cmd
}

func newMeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Get bot info",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			resp, err := callAPI(token, "getMe", nil)
			if err != nil {
				return err
			}

			var user apiUser
			if err := json.Unmarshal(resp.Result, &user); err != nil {
				return output.PrintError("parse_failed", "Failed to parse bot info", map[string]any{
					"error": err.Error(),
				})
			}

			return output.Print(BotInfo{
				ID:                      user.ID,
				IsBot:                   user.IsBot,
				FirstName:               user.FirstName,
				Username:                user.Username,
				CanJoinGroups:           user.CanJoinGroups,
				CanReadAllGroupMessages: user.CanReadAllGroupMessages,
				SupportsInlineQueries:   user.SupportsInlineQueries,
			})
		},
	}

	return cmd
}

func newUpdatesCmd() *cobra.Command {
	var limit int
	var offset int64

	cmd := &cobra.Command{
		Use:   "updates",
		Short: "Get recent updates/messages sent to the bot",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			params := map[string]any{
				"limit": limit,
			}
			if offset > 0 {
				params["offset"] = offset
			}

			resp, err := callAPI(token, "getUpdates", params)
			if err != nil {
				return err
			}

			var apiUpdates []apiUpdate
			if err := json.Unmarshal(resp.Result, &apiUpdates); err != nil {
				return output.PrintError("parse_failed", "Failed to parse updates", map[string]any{
					"error": err.Error(),
				})
			}

			updates := make([]Update, 0, len(apiUpdates))
			for _, u := range apiUpdates {
				update := Update{
					UpdateID: u.UpdateID,
				}
				if u.Message != nil {
					update.Message = convertMessage(u.Message)
				}
				updates = append(updates, update)
			}

			return output.Print(updates)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Number of updates to retrieve (1-100)")
	cmd.Flags().Int64Var(&offset, "offset", 0, "Identifier of first update to return")

	return cmd
}

func newSendCmd() *cobra.Command {
	var chatID string
	var parseMode string
	var disableNotification bool

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message to a chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			if chatID == "" {
				return output.PrintError("missing_chat", "Chat ID is required", map[string]any{
					"hint": "Use --chat flag to specify the chat ID. Get chat IDs from 'pocket telegram chats'",
				})
			}

			params := map[string]any{
				"chat_id": chatID,
				"text":    args[0],
			}

			if parseMode != "" {
				params["parse_mode"] = parseMode
			}
			if disableNotification {
				params["disable_notification"] = true
			}

			resp, err := callAPI(token, "sendMessage", params)
			if err != nil {
				return err
			}

			var msg apiMessage
			if err := json.Unmarshal(resp.Result, &msg); err != nil {
				return output.PrintError("parse_failed", "Failed to parse sent message", map[string]any{
					"error": err.Error(),
				})
			}

			return output.Print(SendResult{
				MessageID: msg.MessageID,
				ChatID:    msg.Chat.ID,
				Date:      formatUnixTime(msg.Date),
				Text:      msg.Text,
			})
		},
	}

	cmd.Flags().StringVarP(&chatID, "chat", "c", "", "Chat ID (required)")
	cmd.Flags().StringVarP(&parseMode, "parse", "p", "", "Parse mode: Markdown, MarkdownV2, or HTML")
	cmd.Flags().BoolVarP(&disableNotification, "silent", "s", false, "Send silently without notification")
	cmd.MarkFlagRequired("chat")

	return cmd
}

func newChatsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "chats",
		Short: "List recent chats from updates",
		Long:  "Lists unique chats that have sent messages to the bot. Uses getUpdates to extract chat information.",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			params := map[string]any{
				"limit": 100, // Get max updates to find more chats
			}

			resp, err := callAPI(token, "getUpdates", params)
			if err != nil {
				return err
			}

			var apiUpdates []apiUpdate
			if err := json.Unmarshal(resp.Result, &apiUpdates); err != nil {
				return output.PrintError("parse_failed", "Failed to parse updates", map[string]any{
					"error": err.Error(),
				})
			}

			// Extract unique chats
			seen := make(map[int64]bool)
			chats := make([]Chat, 0)

			for _, u := range apiUpdates {
				if u.Message == nil {
					continue
				}
				chatID := u.Message.Chat.ID
				if seen[chatID] {
					continue
				}
				seen[chatID] = true

				c := u.Message.Chat
				chats = append(chats, Chat{
					ID:        c.ID,
					Type:      c.Type,
					Title:     c.Title,
					Username:  c.Username,
					FirstName: c.FirstName,
					LastName:  c.LastName,
				})

				if len(chats) >= limit {
					break
				}
			}

			return output.Print(chats)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of chats to return")

	return cmd
}

func newForwardCmd() *cobra.Command {
	var disableNotification bool

	cmd := &cobra.Command{
		Use:   "forward [from-chat-id] [message-id] [to-chat-id]",
		Short: "Forward a message from one chat to another",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			fromChatID := args[0]
			messageID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return output.PrintError("invalid_message_id", "Message ID must be a number", map[string]any{
					"provided": args[1],
				})
			}
			toChatID := args[2]

			params := map[string]any{
				"chat_id":      toChatID,
				"from_chat_id": fromChatID,
				"message_id":   messageID,
			}

			if disableNotification {
				params["disable_notification"] = true
			}

			resp, err := callAPI(token, "forwardMessage", params)
			if err != nil {
				return err
			}

			var msg apiMessage
			if err := json.Unmarshal(resp.Result, &msg); err != nil {
				return output.PrintError("parse_failed", "Failed to parse forwarded message", map[string]any{
					"error": err.Error(),
				})
			}

			// Parse from and to chat IDs for the result
			fromID, _ := strconv.ParseInt(fromChatID, 10, 64)
			toID, _ := strconv.ParseInt(toChatID, 10, 64)

			return output.Print(ForwardResult{
				MessageID:         msg.MessageID,
				FromChatID:        fromID,
				ToChatID:          toID,
				OriginalMessageID: messageID,
				Date:              formatUnixTime(msg.Date),
			})
		},
	}

	cmd.Flags().BoolVarP(&disableNotification, "silent", "s", false, "Forward silently without notification")

	return cmd
}

// getToken retrieves the Telegram token from config
func getToken() (string, error) {
	token, err := config.MustGet("telegram_token")
	if err != nil {
		return "", output.PrintError("token_missing", "Telegram bot token not configured", map[string]any{
			"setup_cmd": "pocket config set telegram_token <your-bot-token>",
			"hint":      "Create a bot via @BotFather on Telegram to get a token",
			"docs":      "https://core.telegram.org/bots#how-do-i-create-a-bot",
		})
	}
	return token, nil
}

// callAPI makes a request to the Telegram Bot API
func callAPI(token, method string, params map[string]any) (*apiResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url := baseURL + token + "/" + method

	var reqBody []byte
	var err error

	if params != nil {
		reqBody, err = json.Marshal(params)
		if err != nil {
			return nil, output.PrintError("request_failed", "Failed to encode request", map[string]any{
				"error": err.Error(),
			})
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, output.PrintError("request_failed", "Failed to create request", map[string]any{
			"error": err.Error(),
		})
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, output.PrintError("request_failed", "Failed to connect to Telegram API", map[string]any{
			"error": err.Error(),
			"hint":  "Check your internet connection",
		})
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, output.PrintError("parse_failed", "Failed to parse API response", map[string]any{
			"error":       err.Error(),
			"status_code": resp.StatusCode,
		})
	}

	if !apiResp.OK {
		// Handle rate limiting
		if apiResp.ErrorCode == 429 && apiResp.Parameters != nil && apiResp.Parameters.RetryAfter > 0 {
			return nil, output.PrintError("rate_limited", "Rate limited by Telegram", map[string]any{
				"retry_after_seconds": apiResp.Parameters.RetryAfter,
				"hint":                fmt.Sprintf("Wait %d seconds before retrying", apiResp.Parameters.RetryAfter),
			})
		}

		// Handle unauthorized (bad token)
		if apiResp.ErrorCode == 401 {
			return nil, output.PrintError("unauthorized", "Invalid or expired bot token", map[string]any{
				"description": apiResp.Description,
				"hint":        "Check your telegram_token is correct. Get a new token from @BotFather if needed.",
			})
		}

		// Handle chat not found
		if apiResp.ErrorCode == 400 && (apiResp.Description == "Bad Request: chat not found" ||
			apiResp.Description == "Bad Request: CHAT_ID_INVALID") {
			return nil, output.PrintError("chat_not_found", "Chat not found", map[string]any{
				"description": apiResp.Description,
				"hint":        "The chat ID may be wrong, or the bot hasn't been added to this chat yet",
			})
		}

		return nil, output.PrintError("api_error", apiResp.Description, map[string]any{
			"error_code": apiResp.ErrorCode,
		})
	}

	return &apiResp, nil
}

// convertMessage converts API message to LLM-friendly format
func convertMessage(m *apiMessage) *Message {
	if m == nil {
		return nil
	}

	msg := &Message{
		MessageID: m.MessageID,
		Date:      formatUnixTime(m.Date),
		Chat: Chat{
			ID:        m.Chat.ID,
			Type:      m.Chat.Type,
			Title:     m.Chat.Title,
			Username:  m.Chat.Username,
			FirstName: m.Chat.FirstName,
			LastName:  m.Chat.LastName,
		},
		Text:    m.Text,
		Caption: m.Caption,
	}

	if m.From != nil {
		msg.From = &User{
			ID:        m.From.ID,
			IsBot:     m.From.IsBot,
			FirstName: m.From.FirstName,
			LastName:  m.From.LastName,
			Username:  m.From.Username,
		}
	}

	return msg
}

// formatUnixTime converts Unix timestamp to ISO 8601 format
func formatUnixTime(ts int64) string {
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
