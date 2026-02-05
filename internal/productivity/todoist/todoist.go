package todoist

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "todoist",
		Aliases: []string{"todo"},
		Short:   "Todoist commands",
	}

	cmd.AddCommand(newTasksCmd())
	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newCompleteCmd())

	return cmd
}

func newTasksCmd() *cobra.Command {
	var project string
	var filter string

	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("todoist_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Todoist tasks coming soon",
				"project": project,
				"filter":  filter,
			})
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name or ID")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "Filter expression")

	return cmd
}

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("todoist_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Todoist projects coming soon",
			})
		},
	}

	return cmd
}

func newAddCmd() *cobra.Command {
	var project string
	var due string
	var priority int

	cmd := &cobra.Command{
		Use:   "add [content]",
		Short: "Add a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("todoist_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":   "not_implemented",
				"message":  "Todoist add coming soon",
				"content":  args[0],
				"project":  project,
				"due":      due,
				"priority": priority,
			})
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name or ID")
	cmd.Flags().StringVarP(&due, "due", "d", "", "Due date")
	cmd.Flags().IntVar(&priority, "priority", 1, "Priority (1-4)")

	return cmd
}

func newCompleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete [task-id]",
		Short: "Complete a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("todoist_token")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Todoist complete coming soon",
				"task_id": args[0],
			})
		},
	}

	return cmd
}
