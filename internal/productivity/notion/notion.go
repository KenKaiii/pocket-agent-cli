package notion

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notion",
		Short: "Notion commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newPageCmd())
	cmd.AddCommand(newDatabaseCmd())

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Notion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("notion_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Notion search coming soon",
				"query":   args[0],
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")

	return cmd
}

func newPageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page [page-id]",
		Short: "Get page content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("notion_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Notion page coming soon",
				"page_id": args[0],
			})
		},
	}

	return cmd
}

func newDatabaseCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "database [database-id]",
		Aliases: []string{"db"},
		Short:   "Query a database",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("notion_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":      "not_implemented",
				"message":     "Notion database coming soon",
				"database_id": args[0],
				"limit":       limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Number of results")

	return cmd
}
