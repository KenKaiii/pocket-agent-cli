package linear

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "linear",
		Short: "Linear commands",
	}

	cmd.AddCommand(newIssuesCmd())
	cmd.AddCommand(newTeamsCmd())
	cmd.AddCommand(newCreateCmd())

	return cmd
}

func newIssuesCmd() *cobra.Command {
	var team string
	var status string
	var limit int

	cmd := &cobra.Command{
		Use:   "issues",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("linear_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Linear issues coming soon",
				"team":    team,
				"state":   status,
				"limit":   limit,
			})
		},
	}

	cmd.Flags().StringVarP(&team, "team", "t", "", "Team key")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Status filter")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of issues")

	return cmd
}

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teams",
		Short: "List teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("linear_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Linear teams coming soon",
			})
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var team string
	var title string

	cmd := &cobra.Command{
		Use:   "create [description]",
		Short: "Create an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("linear_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":      "not_implemented",
				"message":     "Linear create coming soon",
				"team":        team,
				"title":       title,
				"description": args[0],
			})
		},
	}

	cmd.Flags().StringVarP(&team, "team", "t", "", "Team key (required)")
	cmd.Flags().StringVar(&title, "title", "", "Issue title (required)")
	cmd.MarkFlagRequired("team")
	cmd.MarkFlagRequired("title")

	return cmd
}
