package reddit

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reddit",
		Aliases: []string{"rd"},
		Short:   "Reddit commands",
	}

	cmd.AddCommand(newFeedCmd())
	cmd.AddCommand(newSubredditCmd())
	cmd.AddCommand(newPostCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

func newFeedCmd() *cobra.Command {
	var limit int
	var sort string

	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Get home feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("reddit_client_id")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Reddit feed coming soon",
				"limit":   limit,
				"sort":    sort,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of posts")
	cmd.Flags().StringVarP(&sort, "sort", "s", "hot", "Sort: hot, new, top, rising")

	return cmd
}

func newSubredditCmd() *cobra.Command {
	var limit int
	var sort string

	cmd := &cobra.Command{
		Use:   "subreddit [name]",
		Short: "Get subreddit posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("reddit_client_id")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":    "not_implemented",
				"message":   "Reddit subreddit coming soon",
				"subreddit": args[0],
				"limit":     limit,
				"sort":      sort,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of posts")
	cmd.Flags().StringVarP(&sort, "sort", "s", "hot", "Sort: hot, new, top, rising")

	return cmd
}

func newPostCmd() *cobra.Command {
	var subreddit string
	var title string

	cmd := &cobra.Command{
		Use:   "post [content]",
		Short: "Create a post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("reddit_client_id")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":    "not_implemented",
				"message":   "Reddit post coming soon",
				"subreddit": subreddit,
				"title":     title,
				"content":   args[0],
			})
		},
	}

	cmd.Flags().StringVarP(&subreddit, "subreddit", "r", "", "Subreddit to post to (required)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Post title (required)")
	cmd.MarkFlagRequired("subreddit")
	cmd.MarkFlagRequired("title")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int
	var subreddit string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Reddit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("reddit_client_id")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":    "not_implemented",
				"message":   "Reddit search coming soon",
				"query":     args[0],
				"subreddit": subreddit,
				"limit":     limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of results")
	cmd.Flags().StringVarP(&subreddit, "subreddit", "r", "", "Limit to subreddit")

	return cmd
}
