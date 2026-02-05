package email

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "email",
		Aliases: []string{"mail", "gmail"},
		Short:   "Email commands (Gmail)",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var limit int
	var label string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List emails",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("gmail_cred_path")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Email list coming soon",
				"label":   label,
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of emails")
	cmd.Flags().StringVar(&label, "label", "INBOX", "Label/folder")

	return cmd
}

func newReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [message-id]",
		Short: "Read an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("gmail_cred_path")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":     "not_implemented",
				"message":    "Email read coming soon",
				"message_id": args[0],
			})
		},
	}

	return cmd
}

func newSendCmd() *cobra.Command {
	var to string
	var subject string
	var cc string

	cmd := &cobra.Command{
		Use:   "send [body]",
		Short: "Send an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("gmail_cred_path")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Email send coming soon",
				"to":      to,
				"subject": subject,
				"cc":      cc,
				"body":    args[0],
			})
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipient (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject (required)")
	cmd.Flags().StringVar(&cc, "cc", "", "CC recipients")
	cmd.MarkFlagRequired("to")
	cmd.MarkFlagRequired("subject")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search emails",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("gmail_cred_path")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Email search coming soon",
				"query":   args[0],
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")

	return cmd
}
