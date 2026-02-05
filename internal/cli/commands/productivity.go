package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/productivity/calendar"
	"github.com/unstablemind/pocket/internal/productivity/notion"
	"github.com/unstablemind/pocket/internal/productivity/todoist"
)

func NewProductivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "productivity",
		Aliases: []string{"p", "prod"},
		Short:   "Productivity tool commands",
		Long:    `Interact with productivity tools: Calendar, Notion, Todoist, etc.`,
	}

	cmd.AddCommand(calendar.NewCmd())
	cmd.AddCommand(notion.NewCmd())
	cmd.AddCommand(todoist.NewCmd())

	return cmd
}
