package imessage

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

// Chat represents an iMessage conversation
type Chat struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	LastMessage string `json:"last_message,omitempty"`
	LastDate    string `json:"last_date,omitempty"`
	UnreadCount int    `json:"unread_count,omitempty"`
}

// Message represents an iMessage message
type Message struct {
	ID          int64  `json:"id"`
	Text        string `json:"text"`
	IsFromMe    bool   `json:"is_from_me"`
	Date        string `json:"date"`
	ChatID      string `json:"chat_id,omitempty"`
	Sender      string `json:"sender,omitempty"`
	Service     string `json:"service,omitempty"`
	IsDelivered bool   `json:"is_delivered,omitempty"`
	IsRead      bool   `json:"is_read,omitempty"`
}

// appleEpoch is January 1, 2001 00:00:00 UTC (Apple's Core Data timestamp reference)
var appleEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// NewCmd creates the imessage command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "imessage",
		Aliases: []string{"imsg", "messages", "sms"},
		Short:   "iMessage commands (macOS only)",
		Long:    `Interact with iMessage via AppleScript (send) and SQLite (read history). Only available on macOS.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return output.PrintError("platform_unsupported",
					"iMessage is only available on macOS",
					map[string]string{
						"current_platform": runtime.GOOS,
						"required":         "darwin (macOS)",
					})
			}
			return nil
		},
	}

	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newChatsCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newUnreadCmd())

	return cmd
}

// getChatDBPath returns the path to the iMessage chat database
func getChatDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, "Library", "Messages", "chat.db")
}

// openChatDB opens a read-only connection to the chat database
func openChatDB() (*sql.DB, error) {
	dbPath := getChatDBPath()
	if dbPath == "" {
		return nil, fmt.Errorf("could not determine home directory")
	}

	// Check if the file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chat database not found at %s", dbPath)
	}

	// Open with read-only mode and immutable flag for safety
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&immutable=1", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open chat database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		if strings.Contains(err.Error(), "unable to open database") ||
			strings.Contains(err.Error(), "authorization denied") {
			return nil, fmt.Errorf("permission denied: Full Disk Access may be required. " +
				"Go to System Settings > Privacy & Security > Full Disk Access and add your terminal")
		}
		return nil, fmt.Errorf("failed to connect to chat database: %w", err)
	}

	return db, nil
}

// convertAppleTimestamp converts Apple's Core Data timestamp to a readable format
// Apple uses nanoseconds since 2001-01-01 00:00:00 UTC
func convertAppleTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}

	// Apple timestamps can be in nanoseconds (newer) or seconds (older)
	// If the timestamp is very large (> 1 billion), it's likely in nanoseconds
	var t time.Time
	if timestamp > 1000000000000000000 {
		// Nanoseconds since Apple epoch
		t = appleEpoch.Add(time.Duration(timestamp) * time.Nanosecond)
	} else if timestamp > 1000000000000000 {
		// Microseconds since Apple epoch (some versions)
		t = appleEpoch.Add(time.Duration(timestamp) * time.Microsecond)
	} else if timestamp > 1000000000000 {
		// Milliseconds since Apple epoch
		t = appleEpoch.Add(time.Duration(timestamp) * time.Millisecond)
	} else {
		// Seconds since Apple epoch
		t = appleEpoch.Add(time.Duration(timestamp) * time.Second)
	}

	return t.Local().Format("2006-01-02 15:04:05")
}

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// runAppleScript executes an AppleScript and returns the output
func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		if strings.Contains(errMsg, "not allowed assistive access") ||
			strings.Contains(errMsg, "osascript is not allowed") {
			return "", fmt.Errorf("permission denied: Automation access required. " +
				"Go to System Settings > Privacy & Security > Automation and enable access")
		}
		return "", fmt.Errorf("%s", strings.TrimSpace(errMsg))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// newSendCmd creates the send command
func newSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [recipient] [message]",
		Short: "Send an iMessage",
		Long: `Send an iMessage to a recipient. The recipient can be:
- A phone number (e.g., +1234567890, 123-456-7890)
- An email address (e.g., user@example.com)
- A contact name (as saved in Contacts)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			recipient := args[0]
			message := args[1]

			// Determine the service based on recipient format
			service := "iMessage"

			script := fmt.Sprintf(`
tell application "Messages"
	set targetService to 1st service whose service type = %s
	set targetBuddy to buddy "%s" of targetService
	send "%s" to targetBuddy
end tell
`, service, escapeAppleScript(recipient), escapeAppleScript(message))

			_, err := runAppleScript(script)
			if err != nil {
				// Try alternative approach - sending directly to participant
				altScript := fmt.Sprintf(`
tell application "Messages"
	send "%s" to participant "%s"
end tell
`, escapeAppleScript(message), escapeAppleScript(recipient))

				_, err = runAppleScript(altScript)
				if err != nil {
					return output.PrintError("send_failed", err.Error(), map[string]string{
						"recipient": recipient,
						"hint":      "Make sure Messages.app is running and signed in",
					})
				}
			}

			return output.Print(map[string]any{
				"success":   true,
				"message":   "Message sent successfully",
				"recipient": recipient,
				"text":      message,
			})
		},
	}

	return cmd
}

// newChatsCmd creates the chats command to list recent conversations
func newChatsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "chats",
		Short: "List recent conversations",
		Long:  `List recent iMessage conversations from the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openChatDB()
			if err != nil {
				return output.PrintError("db_error", err.Error(), map[string]string{
					"path": getChatDBPath(),
				})
			}
			defer db.Close()

			// Query to get recent chats with their last message
			query := `
				SELECT
					c.ROWID,
					c.chat_identifier,
					c.display_name,
					c.service_name,
					(SELECT m.text FROM message m
					 JOIN chat_message_join cmj ON m.ROWID = cmj.message_id
					 WHERE cmj.chat_id = c.ROWID
					 ORDER BY m.date DESC LIMIT 1) as last_message,
					(SELECT m.date FROM message m
					 JOIN chat_message_join cmj ON m.ROWID = cmj.message_id
					 WHERE cmj.chat_id = c.ROWID
					 ORDER BY m.date DESC LIMIT 1) as last_date
				FROM chat c
				WHERE c.chat_identifier IS NOT NULL
				ORDER BY last_date DESC NULLS LAST
				LIMIT ?
			`

			rows, err := db.Query(query, limit)
			if err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}
			defer rows.Close()

			var chats []Chat
			for rows.Next() {
				var rowID int64
				var chatIdentifier string
				var displayName sql.NullString
				var serviceName sql.NullString
				var lastMessage sql.NullString
				var lastDate sql.NullInt64

				err := rows.Scan(&rowID, &chatIdentifier, &displayName, &serviceName, &lastMessage, &lastDate)
				if err != nil {
					continue
				}

				chat := Chat{
					ID: chatIdentifier,
				}

				// Use display_name if available, otherwise use chat_identifier
				if displayName.Valid && displayName.String != "" {
					chat.DisplayName = displayName.String
				} else {
					chat.DisplayName = chatIdentifier
				}

				if lastMessage.Valid {
					// Truncate long messages
					text := lastMessage.String
					if len(text) > 100 {
						text = text[:97] + "..."
					}
					chat.LastMessage = text
				}

				if lastDate.Valid && lastDate.Int64 > 0 {
					chat.LastDate = convertAppleTimestamp(lastDate.Int64)
				}

				chats = append(chats, chat)
			}
			if err := rows.Err(); err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"chats": chats,
				"count": len(chats),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of conversations to show")

	return cmd
}

// newReadCmd creates the read command to read messages from a conversation
func newReadCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "read [contact]",
		Short: "Read recent messages from a conversation",
		Long: `Read recent messages from a conversation with a specific contact.
The contact can be a phone number, email, or name as it appears in your chats.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contact := args[0]

			db, err := openChatDB()
			if err != nil {
				return output.PrintError("db_error", err.Error(), map[string]string{
					"path": getChatDBPath(),
				})
			}
			defer db.Close()

			// Query to find messages for a specific chat
			// We search by chat_identifier which can be phone number or email
			query := `
				SELECT
					m.ROWID,
					m.text,
					m.is_from_me,
					m.date,
					m.service,
					m.is_delivered,
					m.is_read,
					c.chat_identifier,
					h.id as sender_id
				FROM message m
				JOIN chat_message_join cmj ON m.ROWID = cmj.message_id
				JOIN chat c ON cmj.chat_id = c.ROWID
				LEFT JOIN handle h ON m.handle_id = h.ROWID
				WHERE c.chat_identifier LIKE ?
				   OR c.display_name LIKE ?
				ORDER BY m.date DESC
				LIMIT ?
			`

			searchPattern := "%" + contact + "%"
			rows, err := db.Query(query, searchPattern, searchPattern, limit)
			if err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}
			defer rows.Close()

			var messages []Message
			for rows.Next() {
				var rowID int64
				var text sql.NullString
				var isFromMe int
				var date sql.NullInt64
				var service sql.NullString
				var isDelivered sql.NullInt64
				var isRead sql.NullInt64
				var chatIdentifier string
				var senderID sql.NullString

				err := rows.Scan(&rowID, &text, &isFromMe, &date, &service, &isDelivered, &isRead, &chatIdentifier, &senderID)
				if err != nil {
					continue
				}

				msg := Message{
					ID:       rowID,
					IsFromMe: isFromMe == 1,
					ChatID:   chatIdentifier,
				}

				if text.Valid {
					msg.Text = text.String
				}

				if date.Valid && date.Int64 > 0 {
					msg.Date = convertAppleTimestamp(date.Int64)
				}

				if service.Valid {
					msg.Service = service.String
				}

				if isDelivered.Valid {
					msg.IsDelivered = isDelivered.Int64 == 1
				}

				if isRead.Valid {
					msg.IsRead = isRead.Int64 == 1
				}

				if senderID.Valid && !msg.IsFromMe {
					msg.Sender = senderID.String
				} else if msg.IsFromMe {
					msg.Sender = "me"
				}

				messages = append(messages, msg)
			}
			if err := rows.Err(); err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}

			if len(messages) == 0 {
				return output.Print(map[string]any{
					"contact":  contact,
					"messages": []Message{},
					"count":    0,
					"hint":     "No messages found. Try using the exact phone number or email as shown in 'pocket system imessage chats'",
				})
			}

			// Reverse to show oldest first (chronological order)
			for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
				messages[i], messages[j] = messages[j], messages[i]
			}

			return output.Print(map[string]any{
				"contact":  contact,
				"messages": messages,
				"count":    len(messages),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of messages to show")

	return cmd
}

// newSearchCmd creates the search command to search messages by content
func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search messages by content",
		Long:  `Search all iMessage messages for text content.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			searchQuery := args[0]

			db, err := openChatDB()
			if err != nil {
				return output.PrintError("db_error", err.Error(), map[string]string{
					"path": getChatDBPath(),
				})
			}
			defer db.Close()

			// Query to search messages by content
			query := `
				SELECT
					m.ROWID,
					m.text,
					m.is_from_me,
					m.date,
					m.service,
					c.chat_identifier,
					c.display_name,
					h.id as sender_id
				FROM message m
				JOIN chat_message_join cmj ON m.ROWID = cmj.message_id
				JOIN chat c ON cmj.chat_id = c.ROWID
				LEFT JOIN handle h ON m.handle_id = h.ROWID
				WHERE m.text LIKE ?
				ORDER BY m.date DESC
				LIMIT ?
			`

			searchPattern := "%" + searchQuery + "%"
			rows, err := db.Query(query, searchPattern, limit)
			if err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}
			defer rows.Close()

			var messages []Message
			for rows.Next() {
				var rowID int64
				var text sql.NullString
				var isFromMe int
				var date sql.NullInt64
				var service sql.NullString
				var chatIdentifier string
				var displayName sql.NullString
				var senderID sql.NullString

				err := rows.Scan(&rowID, &text, &isFromMe, &date, &service, &chatIdentifier, &displayName, &senderID)
				if err != nil {
					continue
				}

				msg := Message{
					ID:       rowID,
					IsFromMe: isFromMe == 1,
					ChatID:   chatIdentifier,
				}

				if text.Valid {
					msg.Text = text.String
				}

				if date.Valid && date.Int64 > 0 {
					msg.Date = convertAppleTimestamp(date.Int64)
				}

				if service.Valid {
					msg.Service = service.String
				}

				if senderID.Valid && !msg.IsFromMe {
					msg.Sender = senderID.String
				} else if msg.IsFromMe {
					msg.Sender = "me"
				}

				messages = append(messages, msg)
			}
			if err := rows.Err(); err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"query":    searchQuery,
				"messages": messages,
				"count":    len(messages),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of results")

	return cmd
}

// newUnreadCmd creates the unread command to show unread message count
func newUnreadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unread",
		Short: "Show unread message count",
		Long:  `Show the count of unread messages in iMessage.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openChatDB()
			if err != nil {
				return output.PrintError("db_error", err.Error(), map[string]string{
					"path": getChatDBPath(),
				})
			}
			defer db.Close()

			// Query to count unread messages
			// is_read = 0 means unread, is_from_me = 0 means received message
			query := `
				SELECT COUNT(*)
				FROM message
				WHERE is_read = 0 AND is_from_me = 0
			`

			var count int
			err = db.QueryRow(query).Scan(&count)
			if err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}

			// Also get unread per chat
			chatQuery := `
				SELECT
					c.chat_identifier,
					c.display_name,
					COUNT(*) as unread_count
				FROM message m
				JOIN chat_message_join cmj ON m.ROWID = cmj.message_id
				JOIN chat c ON cmj.chat_id = c.ROWID
				WHERE m.is_read = 0 AND m.is_from_me = 0
				GROUP BY c.ROWID
				ORDER BY unread_count DESC
				LIMIT 10
			`

			rows, err := db.Query(chatQuery)
			if err != nil {
				// If chat query fails, just return the total count
				return output.Print(map[string]any{
					"total_unread": count,
				})
			}
			defer rows.Close()

			var unreadByChat []map[string]any
			for rows.Next() {
				var chatIdentifier string
				var displayName sql.NullString
				var unreadCount int

				err := rows.Scan(&chatIdentifier, &displayName, &unreadCount)
				if err != nil {
					continue
				}

				name := chatIdentifier
				if displayName.Valid && displayName.String != "" {
					name = displayName.String
				}

				unreadByChat = append(unreadByChat, map[string]any{
					"chat":    name,
					"chat_id": chatIdentifier,
					"count":   unreadCount,
				})
			}
			if err := rows.Err(); err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"total_unread":   count,
				"unread_by_chat": unreadByChat,
			})
		},
	}

	return cmd
}
