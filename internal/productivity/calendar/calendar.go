package calendar

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "calendar",
		Aliases: []string{"cal", "gcal"},
		Short:   "Calendar commands (Google Calendar)",
	}

	cmd.AddCommand(newEventsCmd())
	cmd.AddCommand(newTodayCmd())
	cmd.AddCommand(newCreateCmd())

	return cmd
}

func newEventsCmd() *cobra.Command {
	var days int
	var limit int

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List upcoming events",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("google_cred_path")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Calendar events coming soon",
				"days":    days,
				"limit":   limit,
			})
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 7, "Days to look ahead")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum events")

	return cmd
}

func newTodayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "today",
		Short: "List today's events",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("google_cred_path")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "Calendar today coming soon",
			})
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var title string
	var start string
	var end string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an event",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("google_cred_path")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":      "not_implemented",
				"message":     "Calendar create coming soon",
				"title":       title,
				"start":       start,
				"end":         end,
				"description": description,
			})
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Event title (required)")
	cmd.Flags().StringVar(&start, "start", "", "Start time (required)")
	cmd.Flags().StringVar(&end, "end", "", "End time (required)")
	cmd.Flags().StringVar(&description, "desc", "", "Description")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("start")
	cmd.MarkFlagRequired("end")

	return cmd
}
