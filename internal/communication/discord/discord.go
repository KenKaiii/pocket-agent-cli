package discord

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discord",
		Aliases: []string{"dc"},
		Short: "Discord commands",
	}

	cmd.AddCommand(newGuildsCmd())
	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newSendCmd())

	return cmd
}

func newGuildsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guilds",
		Short: "List guilds/servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("discord_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Discord guilds coming soon",
			})
		},
	}

	return cmd
}

func newChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels [guild-id]",
		Short: "List channels in a guild",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("discord_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":   "not_implemented",
				"message":  "Discord channels coming soon",
				"guild_id": args[0],
			})
		},
	}

	return cmd
}

func newMessagesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "messages [channel-id]",
		Short: "Get channel messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("discord_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":     "not_implemented",
				"message":    "Discord messages coming soon",
				"channel_id": args[0],
				"limit":      limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of messages")

	return cmd
}

func newSendCmd() *cobra.Command {
	var channelID string

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("discord_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":     "not_implemented",
				"message":    "Discord send coming soon",
				"channel_id": channelID,
				"content":    args[0],
			})
		},
	}

	cmd.Flags().StringVarP(&channelID, "channel", "c", "", "Channel ID (required)")
	cmd.MarkFlagRequired("channel")

	return cmd
}
