package calendar

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

// Calendar represents an Apple Calendar
type Calendar struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Event represents a calendar event
type Event struct {
	Title       string `json:"title"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Location    string `json:"location,omitempty"`
	Notes       string `json:"notes,omitempty"`
	AllDay      bool   `json:"all_day"`
	Calendar    string `json:"calendar"`
	ID          string `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
}

// checkPlatform returns an error if not running on macOS
func checkPlatform() error {
	if runtime.GOOS != "darwin" {
		return output.PrintError(
			"unsupported_platform",
			"Apple Calendar integration is only available on macOS",
			map[string]any{
				"current_platform": runtime.GOOS,
				"supported":        "darwin (macOS)",
				"suggestion":       "Use Google Calendar integration for cross-platform support",
			},
		)
	}
	return nil
}

// runAppleScript executes an AppleScript and returns the output
func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("AppleScript error: %v - %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// formatDateForAppleScript formats a time.Time for AppleScript
// Uses short numeric format which is more reliable across locales
func formatDateForAppleScript(t time.Time) string {
	// Format: MM/DD/YYYY HH:MM:SS (24h) - works reliably with AppleScript
	return t.Format("1/2/2006 15:04:05")
}

// parseDateTime parses various date/time formats
func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
		"01/02/2006 15:04:05",
		"01/02/2006 15:04",
		"01/02/2006",
		"Jan 2, 2006 3:04 PM",
		"January 2, 2006 3:04 PM",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date/time: %s", s)
}

// NewCmd returns the Apple Calendar command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apple-calendar",
		Aliases: []string{"acal", "applecal", "ical"},
		Short:   "Apple Calendar commands (macOS only)",
		Long:    `Interact with Apple Calendar on macOS using AppleScript. Requires macOS.`,
	}

	cmd.AddCommand(newCalendarsCmd())
	cmd.AddCommand(newTodayCmd())
	cmd.AddCommand(newEventsCmd())
	cmd.AddCommand(newEventCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpcomingCmd())
	cmd.AddCommand(newWeekCmd())

	return cmd
}

// newCalendarsCmd lists all calendars
func newCalendarsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendars",
		Short: "List all calendars",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkPlatform(); err != nil {
				return err
			}

			script := `
tell application "Calendar"
	set calList to {}
	repeat with c in calendars
		set end of calList to name of c
	end repeat
	return calList
end tell
`
			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", "Failed to list calendars", map[string]any{
					"error": err.Error(),
				})
			}

			calendars := []Calendar{}
			if result != "" {
				items := strings.Split(result, ", ")
				for _, item := range items {
					name := strings.TrimSpace(item)
					if name != "" {
						calendars = append(calendars, Calendar{
							Name: name,
						})
					}
				}
			}

			return output.Print(map[string]any{
				"calendars": calendars,
				"count":     len(calendars),
			})
		},
	}

	return cmd
}

// newTodayCmd shows today's events
func newTodayCmd() *cobra.Command {
	var calendarName string

	cmd := &cobra.Command{
		Use:   "today",
		Short: "Show today's events",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkPlatform(); err != nil {
				return err
			}

			today := time.Now()
			startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
			endOfDay := startOfDay.Add(24 * time.Hour)

			return fetchEvents(calendarName, startOfDay, endOfDay, "today")
		},
	}

	cmd.Flags().StringVarP(&calendarName, "calendar", "c", "", "Filter by calendar name")

	return cmd
}

// newEventsCmd lists events within a date range
func newEventsCmd() *cobra.Command {
	var calendarName string
	var days int

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List events",
		Long:  "List events from calendars. By default shows events for the next 7 days.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkPlatform(); err != nil {
				return err
			}

			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endDate := startOfDay.Add(time.Duration(days) * 24 * time.Hour)

			return fetchEvents(calendarName, startOfDay, endDate, fmt.Sprintf("next %d days", days))
		},
	}

	cmd.Flags().StringVarP(&calendarName, "calendar", "c", "", "Filter by calendar name")
	cmd.Flags().IntVarP(&days, "days", "d", 7, "Number of days to look ahead")

	return cmd
}

// newEventCmd gets event details by title
func newEventCmd() *cobra.Command {
	var calendarName string

	cmd := &cobra.Command{
		Use:   "event [title]",
		Short: "Get event details by title",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkPlatform(); err != nil {
				return err
			}

			title := args[0]

			calFilter := ""
			if calendarName != "" {
				calFilter = fmt.Sprintf(`of calendar "%s"`, calendarName)
			}

			script := fmt.Sprintf(`
tell application "Calendar"
	set matchingEvents to {}
	set searchTitle to "%s"

	repeat with c in calendars
		try
			set evts to (every event %s whose summary contains searchTitle)
			repeat with e in evts
				set eventInfo to (summary of e) & "|||" & (start date of e as string) & "|||" & (end date of e as string) & "|||"
				try
					set eventInfo to eventInfo & (location of e)
				end try
				set eventInfo to eventInfo & "|||"
				try
					set eventInfo to eventInfo & (description of e)
				end try
				set eventInfo to eventInfo & "|||" & (allday event of e as string) & "|||" & (name of c)
				set end of matchingEvents to eventInfo
			end repeat
		end try
	end repeat
	return matchingEvents
end tell
`, escapeAppleScriptString(title), calFilter)

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", "Failed to search for event", map[string]any{
					"error": err.Error(),
					"title": title,
				})
			}

			events := parseEventResults(result)

			if len(events) == 0 {
				return output.Print(map[string]any{
					"events":  events,
					"count":   0,
					"message": fmt.Sprintf("No events found matching '%s'", title),
				})
			}

			return output.Print(map[string]any{
				"events": events,
				"count":  len(events),
			})
		},
	}

	cmd.Flags().StringVarP(&calendarName, "calendar", "c", "", "Filter by calendar name")

	return cmd
}

// newCreateCmd creates a new event
func newCreateCmd() *cobra.Command {
	var calendarName string
	var startTime string
	var endTime string
	var location string
	var notes string
	var allDay bool

	cmd := &cobra.Command{
		Use:   "create [title]",
		Short: "Create a new event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkPlatform(); err != nil {
				return err
			}

			title := args[0]

			// Parse start time
			start, err := parseDateTime(startTime)
			if err != nil {
				return output.PrintError("invalid_start_time", "Failed to parse start time", map[string]any{
					"input": startTime,
					"error": err.Error(),
					"hint":  "Use format: 2006-01-02 15:04 or 2006-01-02T15:04:05",
				})
			}

			// Parse end time
			end, err := parseDateTime(endTime)
			if err != nil {
				return output.PrintError("invalid_end_time", "Failed to parse end time", map[string]any{
					"input": endTime,
					"error": err.Error(),
					"hint":  "Use format: 2006-01-02 15:04 or 2006-01-02T15:04:05",
				})
			}

			// Determine which calendar to use
			targetCalendar := calendarName
			if targetCalendar == "" {
				// Get the first calendar as default
				listScript := `
tell application "Calendar"
	return name of first calendar
end tell
`
				result, err := runAppleScript(listScript)
				if err != nil {
					return output.PrintError("no_calendar", "Failed to find a calendar", map[string]any{
						"error": err.Error(),
						"hint":  "Specify a calendar with --calendar flag",
					})
				}
				targetCalendar = result
			}

			// Build the create script
			locationPart := ""
			if location != "" {
				locationPart = fmt.Sprintf(`set location of newEvent to "%s"`, escapeAppleScriptString(location))
			}

			notesPart := ""
			if notes != "" {
				notesPart = fmt.Sprintf(`set description of newEvent to "%s"`, escapeAppleScriptString(notes))
			}

			allDayPart := ""
			if allDay {
				allDayPart = `set allday event of newEvent to true`
			}

			script := fmt.Sprintf(`
tell application "Calendar"
	tell calendar "%s"
		set startDate to date "%s"
		set endDate to date "%s"
		set newEvent to make new event with properties {summary:"%s", start date:startDate, end date:endDate}
		%s
		%s
		%s
		return summary of newEvent
	end tell
end tell
`, escapeAppleScriptString(targetCalendar),
				formatDateForAppleScript(start),
				formatDateForAppleScript(end),
				escapeAppleScriptString(title),
				locationPart,
				notesPart,
				allDayPart,
			)

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("create_failed", "Failed to create event", map[string]any{
					"error": err.Error(),
				})
			}

			return output.Print(map[string]any{
				"success":  true,
				"message":  "Event created successfully",
				"title":    result,
				"calendar": targetCalendar,
				"start":    startTime,
				"end":      endTime,
				"location": location,
				"notes":    notes,
				"all_day":  allDay,
			})
		},
	}

	cmd.Flags().StringVarP(&calendarName, "calendar", "c", "", "Calendar name (uses first calendar if not specified)")
	cmd.Flags().StringVarP(&startTime, "start", "s", "", "Start time (required)")
	cmd.Flags().StringVarP(&endTime, "end", "e", "", "End time (required)")
	cmd.Flags().StringVarP(&location, "location", "l", "", "Event location")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "Event notes/description")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "All-day event")
	cmd.MarkFlagRequired("start")
	cmd.MarkFlagRequired("end")

	return cmd
}

// newUpcomingCmd shows next 7 days of events
func newUpcomingCmd() *cobra.Command {
	var calendarName string

	cmd := &cobra.Command{
		Use:   "upcoming",
		Short: "Show next 7 days of events",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkPlatform(); err != nil {
				return err
			}

			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endDate := startOfDay.Add(7 * 24 * time.Hour)

			return fetchEvents(calendarName, startOfDay, endDate, "upcoming 7 days")
		},
	}

	cmd.Flags().StringVarP(&calendarName, "calendar", "c", "", "Filter by calendar name")

	return cmd
}

// newWeekCmd shows this week's events
func newWeekCmd() *cobra.Command {
	var calendarName string

	cmd := &cobra.Command{
		Use:   "week",
		Short: "Show this week's events",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkPlatform(); err != nil {
				return err
			}

			now := time.Now()
			// Find the start of the week (Sunday)
			weekday := int(now.Weekday())
			startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-weekday, 0, 0, 0, 0, now.Location())
			endOfWeek := startOfWeek.Add(7 * 24 * time.Hour)

			return fetchEvents(calendarName, startOfWeek, endOfWeek, "this week")
		},
	}

	cmd.Flags().StringVarP(&calendarName, "calendar", "c", "", "Filter by calendar name")

	return cmd
}

// fetchEvents fetches events within a date range
func fetchEvents(calendarName string, startDate, endDate time.Time, period string) error {
	// Calculate offset from current date for AppleScript
	now := time.Now()
	startOffset := int(startDate.Sub(now).Hours() / 24)
	endOffset := int(endDate.Sub(now).Hours() / 24)

	calFilter := ""
	if calendarName != "" {
		calFilter = fmt.Sprintf(` of calendar "%s"`, escapeAppleScriptString(calendarName))
	}

	// Use current date offsets instead of date string parsing for reliability
	script := fmt.Sprintf(`
tell application "Calendar"
	set today to current date
	set todayStart to today - (time of today)
	set startDate to todayStart + (%d * days)
	set endDate to todayStart + (%d * days)
	set eventList to {}

	repeat with c in calendars
		try
			set evts to (every event%s of c whose start date >= startDate and start date < endDate)
			repeat with e in evts
				set eventInfo to (summary of e) & "|||" & (start date of e as string) & "|||" & (end date of e as string) & "|||"
				try
					set eventInfo to eventInfo & (location of e)
				end try
				set eventInfo to eventInfo & "|||"
				try
					set eventInfo to eventInfo & (description of e)
				end try
				set eventInfo to eventInfo & "|||" & (allday event of e as string) & "|||" & (name of c)
				set end of eventList to eventInfo
			end repeat
		end try
	end repeat
	return eventList
end tell
`, startOffset, endOffset, calFilter)

	result, err := runAppleScript(script)
	if err != nil {
		return output.PrintError("applescript_error", "Failed to fetch events", map[string]any{
			"error":  err.Error(),
			"period": period,
		})
	}

	events := parseEventResults(result)

	return output.Print(map[string]any{
		"events":     events,
		"count":      len(events),
		"period":     period,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	})
}

// parseEventResults parses the AppleScript event output
func parseEventResults(result string) []Event {
	events := []Event{}
	if result == "" {
		return events
	}

	items := strings.Split(result, ", ")
	for _, item := range items {
		parts := strings.Split(item, "|||")
		if len(parts) >= 7 {
			event := Event{
				Title:     strings.TrimSpace(parts[0]),
				StartDate: strings.TrimSpace(parts[1]),
				EndDate:   strings.TrimSpace(parts[2]),
				Location:  strings.TrimSpace(parts[3]),
				Notes:     strings.TrimSpace(parts[4]),
				AllDay:    strings.TrimSpace(parts[5]) == "true",
				Calendar:  strings.TrimSpace(parts[6]),
			}
			events = append(events, event)
		}
	}

	return events
}

// escapeAppleScriptString escapes special characters for AppleScript strings
func escapeAppleScriptString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
