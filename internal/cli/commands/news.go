package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/news/feeds"
	"github.com/unstablemind/pocket/internal/news/hackernews"
	"github.com/unstablemind/pocket/internal/news/newsapi"
)

func NewNewsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "news",
		Aliases: []string{"n"},
		Short:   "News and content commands",
		Long:    `Fetch news and content: RSS feeds, Hacker News, NewsAPI, etc.`,
	}

	cmd.AddCommand(feeds.NewCmd())
	cmd.AddCommand(hackernews.NewCmd())
	cmd.AddCommand(newsapi.NewCmd())

	return cmd
}
