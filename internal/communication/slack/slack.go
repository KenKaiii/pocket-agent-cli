package slack

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slack",
		Short: "Slack commands",
	}

	cmd.AddCommand(newChannelsCmd())
	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newSendCmd())

	return cmd
}

func newChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "List channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("slack_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Slack channels coming soon",
			})
		},
	}

	return cmd
}

func newMessagesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "messages [channel]",
		Short: "Get channel messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("slack_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Slack messages coming soon",
				"channel": args[0],
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of messages")

	return cmd
}

func newSendCmd() *cobra.Command {
	var channel string

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("slack_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Slack send coming soon",
				"channel": channel,
				"content": args[0],
			})
		},
	}

	cmd.Flags().StringVarP(&channel, "channel", "c", "", "Channel (required)")
	cmd.MarkFlagRequired("channel")

	return cmd
}
