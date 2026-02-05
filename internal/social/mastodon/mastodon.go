package mastodon

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mastodon",
		Aliases: []string{"masto", "fedi"},
		Short:   "Mastodon/Fediverse commands",
	}

	cmd.AddCommand(newTimelineCmd())
	cmd.AddCommand(newPostCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

func newTimelineCmd() *cobra.Command {
	var limit int
	var timeline string

	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Get timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("mastodon_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":   "not_implemented",
				"message":  "Mastodon timeline coming soon",
				"timeline": timeline,
				"limit":    limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of posts")
	cmd.Flags().StringVarP(&timeline, "type", "t", "home", "Timeline: home, local, public")

	return cmd
}

func newPostCmd() *cobra.Command {
	var visibility string

	cmd := &cobra.Command{
		Use:   "post [content]",
		Short: "Post a toot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("mastodon_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":     "not_implemented",
				"message":    "Mastodon post coming soon",
				"content":    args[0],
				"visibility": visibility,
			})
		},
	}

	cmd.Flags().StringVarP(&visibility, "visibility", "V", "public", "Visibility: public, unlisted, private, direct")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int
	var searchType string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Mastodon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("mastodon_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Mastodon search coming soon",
				"query":   args[0],
				"type":    searchType,
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")
	cmd.Flags().StringVarP(&searchType, "type", "t", "all", "Type: accounts, hashtags, statuses, all")

	return cmd
}
