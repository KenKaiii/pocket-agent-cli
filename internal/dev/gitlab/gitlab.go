package gitlab

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gitlab",
		Aliases: []string{"gl"},
		Short:   "GitLab commands",
	}

	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newIssuesCmd())
	cmd.AddCommand(newMRsCmd())

	return cmd
}

func newProjectsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("gitlab_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "GitLab projects coming soon",
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 30, "Number of projects")

	return cmd
}

func newIssuesCmd() *cobra.Command {
	var project string
	var state string
	var limit int

	cmd := &cobra.Command{
		Use:   "issues",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("gitlab_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "GitLab issues coming soon",
				"project": project,
				"state":   state,
				"limit":   limit,
			})
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project ID or path")
	cmd.Flags().StringVarP(&state, "state", "s", "opened", "State: opened, closed, all")
	cmd.Flags().IntVarP(&limit, "limit", "l", 30, "Number of issues")

	return cmd
}

func newMRsCmd() *cobra.Command {
	var project string
	var state string
	var limit int

	cmd := &cobra.Command{
		Use:     "mrs",
		Aliases: []string{"merge-requests"},
		Short:   "List merge requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("gitlab_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "GitLab MRs coming soon",
				"project": project,
				"state":   state,
				"limit":   limit,
			})
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project ID or path")
	cmd.Flags().StringVarP(&state, "state", "s", "opened", "State: opened, closed, merged, all")
	cmd.Flags().IntVarP(&limit, "limit", "l", 30, "Number of MRs")

	return cmd
}
