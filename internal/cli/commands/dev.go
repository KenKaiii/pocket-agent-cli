package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/dev/github"
	"github.com/unstablemind/pocket/internal/dev/gitlab"
	"github.com/unstablemind/pocket/internal/dev/linear"
	"github.com/unstablemind/pocket/internal/dev/npm"
	"github.com/unstablemind/pocket/internal/dev/pypi"
)

func NewDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Aliases: []string{"d"},
		Short:   "Developer tool commands",
		Long:    `Interact with developer tools: GitHub, GitLab, Linear, Jira, etc.`,
	}

	cmd.AddCommand(github.NewCmd())
	cmd.AddCommand(gitlab.NewCmd())
	cmd.AddCommand(linear.NewCmd())
	cmd.AddCommand(npm.NewCmd())
	cmd.AddCommand(pypi.NewCmd())

	return cmd
}
