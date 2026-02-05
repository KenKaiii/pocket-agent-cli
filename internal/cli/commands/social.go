package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/social/mastodon"
	"github.com/unstablemind/pocket/internal/social/reddit"
	"github.com/unstablemind/pocket/internal/social/twitter"
	"github.com/unstablemind/pocket/internal/social/youtube"
)

func NewSocialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "social",
		Aliases: []string{"s"},
		Short:   "Social media commands",
		Long:    `Interact with social media platforms: Twitter/X, Reddit, Mastodon, YouTube, etc.`,
	}

	cmd.AddCommand(twitter.NewCmd())
	cmd.AddCommand(reddit.NewCmd())
	cmd.AddCommand(mastodon.NewCmd())
	cmd.AddCommand(youtube.NewCmd())

	return cmd
}
