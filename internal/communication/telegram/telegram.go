package telegram

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "telegram",
		Aliases: []string{"tg"},
		Short:   "Telegram commands",
	}

	cmd.AddCommand(newChatsCmd())
	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newSendCmd())

	return cmd
}

func newChatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chats",
		Short: "List chats",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("telegram_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Telegram chats coming soon",
			})
		},
	}

	return cmd
}

func newMessagesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "messages [chat-id]",
		Short: "Get chat messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("telegram_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Telegram messages coming soon",
				"chat_id": args[0],
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of messages")

	return cmd
}

func newSendCmd() *cobra.Command {
	var chatID string

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("telegram_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Telegram send coming soon",
				"chat_id": chatID,
				"content": args[0],
			})
		},
	}

	cmd.Flags().StringVarP(&chatID, "chat", "c", "", "Chat ID (required)")
	cmd.MarkFlagRequired("chat")

	return cmd
}
