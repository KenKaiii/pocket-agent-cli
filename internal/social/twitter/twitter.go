package twitter

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "twitter",
		Aliases: []string{"tw", "x"},
		Short:   "Twitter/X commands",
	}

	cmd.AddCommand(newTimelineCmd())
	cmd.AddCommand(newPostCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newUserCmd())

	return cmd
}

func newTimelineCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Get home timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("twitter_api_key")
			if err != nil {
				return err
			}

			// TODO: Implement Twitter API call
			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Twitter timeline fetch coming soon",
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of tweets to fetch")

	return cmd
}

func newPostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post [message]",
		Short: "Post a tweet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("twitter_api_key")
			if err != nil {
				return err
			}

			// TODO: Implement Twitter API call
			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Twitter post coming soon",
				"content": args[0],
			})
		},
	}

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search tweets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("twitter_api_key")
			if err != nil {
				return err
			}

			// TODO: Implement Twitter API call
			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Twitter search coming soon",
				"query":   args[0],
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")

	return cmd
}

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user [username]",
		Short: "Get user info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("twitter_api_key")
			if err != nil {
				return err
			}

			// TODO: Implement Twitter API call
			return output.Print(map[string]any{
				"status":   "not_implemented",
				"message":  "Twitter user lookup coming soon",
				"username": args[0],
			})
		},
	}

	return cmd
}
