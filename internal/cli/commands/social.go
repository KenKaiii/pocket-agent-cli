package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/social/twitter"
	"github.com/unstablemind/pocket/internal/social/reddit"
	"github.com/unstablemind/pocket/internal/social/mastodon"
)

func NewSocialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "social",
		Aliases: []string{"s"},
		Short:   "Social media commands",
		Long:    `Interact with social media platforms: Twitter/X, Reddit, Mastodon, etc.`,
	}

	// Twitter/X subcommands
	cmd.AddCommand(twitter.NewCmd())

	// Reddit subcommands
	cmd.AddCommand(reddit.NewCmd())

	// Mastodon subcommands
	cmd.AddCommand(mastodon.NewCmd())

	return cmd
}
