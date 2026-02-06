package safari

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"

	_ "github.com/mattn/go-sqlite3"
)

// Tab represents a Safari tab
type Tab struct {
	WindowIndex int    `json:"window_index"`
	TabIndex    int    `json:"tab_index"`
	Title       string `json:"title"`
	URL         string `json:"url"`
}

// Bookmark represents a Safari bookmark
type Bookmark struct {
	Title  string `json:"title"`
	URL    string `json:"url,omitempty"`
	Folder string `json:"folder,omitempty"`
}

// ReadingListItem represents a Safari Reading List item
type ReadingListItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Preview     string `json:"preview,omitempty"`
	DateAdded   string `json:"date_added,omitempty"`
	DateVisited string `json:"date_visited,omitempty"`
}

// HistoryItem represents a Safari history entry
type HistoryItem struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	VisitTime     string `json:"visit_time"`
	VisitCount    int    `json:"visit_count,omitempty"`
	LastVisitTime string `json:"last_visit_time,omitempty"`
}

// NewCmd creates the Safari command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "safari",
		Aliases: []string{"saf"},
		Short:   "Safari browser commands (macOS only)",
		Long:    `Interact with Safari browser via AppleScript. Only available on macOS.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return output.PrintError("platform_unsupported",
					"Safari is only available on macOS",
					map[string]string{
						"current_platform": runtime.GOOS,
						"required":         "darwin (macOS)",
					})
			}
			return nil
		},
	}

	cmd.AddCommand(newTabsCmd())
	cmd.AddCommand(newURLCmd())
	cmd.AddCommand(newTitleCmd())
	cmd.AddCommand(newOpenCmd())
	cmd.AddCommand(newCloseCmd())
	cmd.AddCommand(newBookmarksCmd())
	cmd.AddCommand(newReadingListCmd())
	cmd.AddCommand(newAddReadingCmd())
	cmd.AddCommand(newHistoryCmd())

	return cmd
}

// runAppleScript executes an AppleScript and returns the output
func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Check if Safari is not running
		if strings.Contains(errMsg, "Application isn't running") ||
			strings.Contains(errMsg, "Connection is invalid") {
			return "", fmt.Errorf("Safari is not running. Please launch Safari first")
		}
		return "", fmt.Errorf("%s", strings.TrimSpace(errMsg))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// isSafariRunning checks if Safari is currently running
func isSafariRunning() bool {
	script := `tell application "System Events" to (name of processes) contains "Safari"`
	result, err := runAppleScript(script)
	if err != nil {
		return false
	}
	return result == "true"
}

// newTabsCmd lists all open tabs
func newTabsCmd() *cobra.Command {
	var windowIndex int

	cmd := &cobra.Command{
		Use:   "tabs",
		Short: "List all open tabs in Safari",
		Long:  "List all open tabs across all Safari windows. Optionally filter by window index.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isSafariRunning() {
				return output.PrintError("safari_not_running",
					"Safari is not running",
					map[string]string{"suggestion": "Launch Safari first"})
			}

			script := `
tell application "Safari"
	set tabList to {}
	set windowCount to count of windows
	repeat with w from 1 to windowCount
		set tabCount to count of tabs of window w
		repeat with t from 1 to tabCount
			set theTab to tab t of window w
			set tabTitle to name of theTab
			set tabURL to URL of theTab
			if tabURL is missing value then set tabURL to ""
			if tabTitle is missing value then set tabTitle to ""
			set end of tabList to (w as string) & "|||" & (t as string) & "|||" & tabTitle & "|||" & tabURL
		end repeat
	end repeat
	set AppleScript's text item delimiters to ":::"
	return tabList as text
end tell`

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("tabs_failed", err.Error(), nil)
			}

			if result == "" {
				return output.Print(map[string]any{
					"tabs":  []Tab{},
					"count": 0,
				})
			}

			var tabs []Tab
			items := strings.Split(result, ":::")
			for _, item := range items {
				parts := strings.Split(item, "|||")
				if len(parts) >= 4 {
					wIdx, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
					tIdx, _ := strconv.Atoi(strings.TrimSpace(parts[1]))

					// Filter by window if specified
					if windowIndex > 0 && wIdx != windowIndex {
						continue
					}

					tabs = append(tabs, Tab{
						WindowIndex: wIdx,
						TabIndex:    tIdx,
						Title:       strings.TrimSpace(parts[2]),
						URL:         strings.TrimSpace(parts[3]),
					})
				}
			}

			return output.Print(map[string]any{
				"tabs":  tabs,
				"count": len(tabs),
			})
		},
	}

	cmd.Flags().IntVarP(&windowIndex, "window", "w", 0, "Filter by window index (1-based, 0 = all windows)")

	return cmd
}

// newURLCmd gets the URL of the current tab
func newURLCmd() *cobra.Command {
	var windowIndex int
	var tabIndex int

	cmd := &cobra.Command{
		Use:   "url",
		Short: "Get URL of the current or specified tab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isSafariRunning() {
				return output.PrintError("safari_not_running",
					"Safari is not running",
					map[string]string{"suggestion": "Launch Safari first"})
			}

			var script string
			if windowIndex > 0 && tabIndex > 0 {
				script = fmt.Sprintf(`
tell application "Safari"
	set theTab to tab %d of window %d
	set tabURL to URL of theTab
	set tabTitle to name of theTab
	if tabURL is missing value then set tabURL to ""
	if tabTitle is missing value then set tabTitle to ""
	return tabTitle & "|||" & tabURL
end tell`, tabIndex, windowIndex)
			} else {
				script = `
tell application "Safari"
	set theTab to current tab of front window
	set tabURL to URL of theTab
	set tabTitle to name of theTab
	if tabURL is missing value then set tabURL to ""
	if tabTitle is missing value then set tabTitle to ""
	return tabTitle & "|||" & tabURL
end tell`
			}

			result, err := runAppleScript(script)
			if err != nil {
				if strings.Contains(err.Error(), "Can't get window") {
					return output.PrintError("no_window", "No Safari window is open", nil)
				}
				return output.PrintError("url_failed", err.Error(), nil)
			}

			parts := strings.Split(result, "|||")
			if len(parts) < 2 {
				return output.PrintError("parse_failed", "Failed to parse tab data", nil)
			}

			return output.Print(map[string]any{
				"title": strings.TrimSpace(parts[0]),
				"url":   strings.TrimSpace(parts[1]),
			})
		},
	}

	cmd.Flags().IntVarP(&windowIndex, "window", "w", 0, "Window index (1-based)")
	cmd.Flags().IntVarP(&tabIndex, "tab", "t", 0, "Tab index (1-based)")

	return cmd
}

// newTitleCmd gets the title of the current tab
func newTitleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "title",
		Short: "Get title of the current tab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isSafariRunning() {
				return output.PrintError("safari_not_running",
					"Safari is not running",
					map[string]string{"suggestion": "Launch Safari first"})
			}

			script := `
tell application "Safari"
	set theTab to current tab of front window
	set tabTitle to name of theTab
	set tabURL to URL of theTab
	if tabTitle is missing value then set tabTitle to ""
	if tabURL is missing value then set tabURL to ""
	return tabTitle & "|||" & tabURL
end tell`

			result, err := runAppleScript(script)
			if err != nil {
				if strings.Contains(err.Error(), "Can't get window") {
					return output.PrintError("no_window", "No Safari window is open", nil)
				}
				return output.PrintError("title_failed", err.Error(), nil)
			}

			parts := strings.Split(result, "|||")
			if len(parts) < 2 {
				return output.PrintError("parse_failed", "Failed to parse tab data", nil)
			}

			return output.Print(map[string]any{
				"title": strings.TrimSpace(parts[0]),
				"url":   strings.TrimSpace(parts[1]),
			})
		},
	}

	return cmd
}

// newOpenCmd opens a URL in a new tab
func newOpenCmd() *cobra.Command {
	var newWindow bool

	cmd := &cobra.Command{
		Use:   "open [url]",
		Short: "Open URL in a new tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]

			// Add https:// if no protocol specified
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "https://" + url
			}

			var script string
			if newWindow {
				script = fmt.Sprintf(`
tell application "Safari"
	activate
	make new document with properties {URL:"%s"}
	delay 0.5
	set theTab to current tab of front window
	set tabTitle to name of theTab
	set tabURL to URL of theTab
	if tabTitle is missing value then set tabTitle to ""
	if tabURL is missing value then set tabURL to ""
	return tabTitle & "|||" & tabURL
end tell`, escapeAppleScript(url))
			} else {
				// Check if Safari is running and has windows
				if !isSafariRunning() {
					script = fmt.Sprintf(`
tell application "Safari"
	activate
	make new document with properties {URL:"%s"}
	delay 0.5
	set theTab to current tab of front window
	set tabTitle to name of theTab
	set tabURL to URL of theTab
	if tabTitle is missing value then set tabTitle to ""
	if tabURL is missing value then set tabURL to ""
	return tabTitle & "|||" & tabURL
end tell`, escapeAppleScript(url))
				} else {
					script = fmt.Sprintf(`
tell application "Safari"
	activate
	tell front window
		set newTab to make new tab with properties {URL:"%s"}
		set current tab to newTab
	end tell
	delay 0.5
	set theTab to current tab of front window
	set tabTitle to name of theTab
	set tabURL to URL of theTab
	if tabTitle is missing value then set tabTitle to ""
	if tabURL is missing value then set tabURL to ""
	return tabTitle & "|||" & tabURL
end tell`, escapeAppleScript(url))
				}
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("open_failed", err.Error(), nil)
			}

			parts := strings.Split(result, "|||")
			title := ""
			actualURL := url
			if len(parts) >= 2 {
				title = strings.TrimSpace(parts[0])
				actualURL = strings.TrimSpace(parts[1])
			}

			return output.Print(map[string]any{
				"success":    true,
				"message":    "URL opened successfully",
				"url":        actualURL,
				"title":      title,
				"new_window": newWindow,
			})
		},
	}

	cmd.Flags().BoolVarP(&newWindow, "new-window", "n", false, "Open in a new window instead of a new tab")

	return cmd
}

// newCloseCmd closes the current tab
func newCloseCmd() *cobra.Command {
	var windowIndex int
	var tabIndex int
	var closeWindow bool

	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close the current or specified tab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isSafariRunning() {
				return output.PrintError("safari_not_running",
					"Safari is not running",
					map[string]string{"suggestion": "Launch Safari first"})
			}

			var script string
			if closeWindow {
				if windowIndex > 0 {
					script = fmt.Sprintf(`
tell application "Safari"
	close window %d
	return "Window closed"
end tell`, windowIndex)
				} else {
					script = `
tell application "Safari"
	close front window
	return "Window closed"
end tell`
				}
			} else if windowIndex > 0 && tabIndex > 0 {
				script = fmt.Sprintf(`
tell application "Safari"
	set theTab to tab %d of window %d
	set tabTitle to name of theTab
	close theTab
	return tabTitle
end tell`, tabIndex, windowIndex)
			} else {
				script = `
tell application "Safari"
	set theTab to current tab of front window
	set tabTitle to name of theTab
	close theTab
	return tabTitle
end tell`
			}

			result, err := runAppleScript(script)
			if err != nil {
				if strings.Contains(err.Error(), "Can't get window") {
					return output.PrintError("no_window", "No Safari window is open", nil)
				}
				return output.PrintError("close_failed", err.Error(), nil)
			}

			if closeWindow {
				return output.Print(map[string]any{
					"success": true,
					"message": "Window closed",
				})
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Tab closed",
				"title":   strings.TrimSpace(result),
			})
		},
	}

	cmd.Flags().IntVarP(&windowIndex, "window", "w", 0, "Window index (1-based)")
	cmd.Flags().IntVarP(&tabIndex, "tab", "t", 0, "Tab index (1-based)")
	cmd.Flags().BoolVar(&closeWindow, "window-close", false, "Close the entire window instead of just the tab")

	return cmd
}

// newBookmarksCmd lists bookmarks
func newBookmarksCmd() *cobra.Command {
	var folder string
	var limit int

	cmd := &cobra.Command{
		Use:   "bookmarks",
		Short: "List Safari bookmarks (Favorites and other folders)",
		Long:  "List Safari bookmarks. By default lists items from Favorites Bar. Use --folder to specify a different folder.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Safari bookmarks are stored in a plist, we'll use AppleScript to access them
			var script string
			if folder != "" {
				script = fmt.Sprintf(`
tell application "Safari"
	set bookmarkList to {}
	try
		set targetFolder to bookmark folder "%s"
		repeat with b in bookmark items of targetFolder
			try
				set bName to name of b
				set bURL to URL of b
				if bURL is not missing value then
					set end of bookmarkList to bName & "|||" & bURL & "|||%s"
				end if
			end try
		end repeat
	end try
	set AppleScript's text item delimiters to ":::"
	return bookmarkList as text
end tell`, escapeAppleScript(folder), escapeAppleScript(folder))
			} else {
				// Get bookmarks from Favorites Bar
				script = `
tell application "Safari"
	set bookmarkList to {}
	try
		set favFolder to bookmark folder "Favorites"
		repeat with b in bookmark items of favFolder
			try
				set bName to name of b
				set bURL to URL of b
				if bURL is not missing value then
					set end of bookmarkList to bName & "|||" & bURL & "|||Favorites"
				end if
			end try
		end repeat
	end try
	set AppleScript's text item delimiters to ":::"
	return bookmarkList as text
end tell`
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("bookmarks_failed", err.Error(), nil)
			}

			if result == "" {
				return output.Print(map[string]any{
					"bookmarks": []Bookmark{},
					"count":     0,
					"folder":    folder,
				})
			}

			var bookmarks []Bookmark
			items := strings.Split(result, ":::")
			count := 0
			for _, item := range items {
				if limit > 0 && count >= limit {
					break
				}
				parts := strings.Split(item, "|||")
				if len(parts) >= 3 {
					bookmarks = append(bookmarks, Bookmark{
						Title:  strings.TrimSpace(parts[0]),
						URL:    strings.TrimSpace(parts[1]),
						Folder: strings.TrimSpace(parts[2]),
					})
					count++
				}
			}

			folderName := folder
			if folderName == "" {
				folderName = "Favorites"
			}

			return output.Print(map[string]any{
				"bookmarks": bookmarks,
				"count":     len(bookmarks),
				"folder":    folderName,
			})
		},
	}

	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Bookmark folder to list (default: Favorites)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of bookmarks (0 = no limit)")

	return cmd
}

// newReadingListCmd lists Reading List items
func newReadingListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "reading-list",
		Short: "List Safari Reading List items",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Reading List items are stored in the Bookmarks.plist file
			// We'll read it directly since AppleScript access is limited
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return output.PrintError("home_dir_failed", "Failed to get home directory", nil)
			}

			bookmarksPlist := filepath.Join(homeDir, "Library", "Safari", "Bookmarks.plist")

			// Use plutil to convert plist to JSON for easier parsing
			cmd2 := exec.Command("plutil", "-convert", "json", "-o", "-", bookmarksPlist)
			var stdout, stderr bytes.Buffer
			cmd2.Stdout = &stdout
			cmd2.Stderr = &stderr

			if err := cmd2.Run(); err != nil {
				// Try alternative method using defaults command
				script := `
tell application "Safari"
	set readingList to {}
	try
		set rlFolder to bookmark folder "com.apple.ReadingList"
		repeat with b in bookmark items of rlFolder
			try
				set bName to name of b
				set bURL to URL of b
				if bURL is not missing value then
					set end of readingList to bName & "|||" & bURL
				end if
			end try
		end repeat
	end try
	set AppleScript's text item delimiters to ":::"
	return readingList as text
end tell`

				result, err := runAppleScript(script)
				if err != nil {
					return output.PrintError("reading_list_failed",
						"Failed to access Reading List",
						map[string]string{"error": err.Error()})
				}

				if result == "" {
					return output.Print(map[string]any{
						"items": []ReadingListItem{},
						"count": 0,
					})
				}

				var items []ReadingListItem
				itemStrs := strings.Split(result, ":::")
				count := 0
				for _, item := range itemStrs {
					if limit > 0 && count >= limit {
						break
					}
					parts := strings.Split(item, "|||")
					if len(parts) >= 2 {
						items = append(items, ReadingListItem{
							Title: strings.TrimSpace(parts[0]),
							URL:   strings.TrimSpace(parts[1]),
						})
						count++
					}
				}

				return output.Print(map[string]any{
					"items": items,
					"count": len(items),
				})
			}

			// Parse JSON output to find Reading List items
			// For simplicity, we'll use the AppleScript approach as primary
			script := `
tell application "Safari"
	set readingList to {}
	try
		set rlFolder to bookmark folder "com.apple.ReadingList"
		repeat with b in bookmark items of rlFolder
			try
				set bName to name of b
				set bURL to URL of b
				if bURL is not missing value then
					set end of readingList to bName & "|||" & bURL
				end if
			end try
		end repeat
	end try
	set AppleScript's text item delimiters to ":::"
	return readingList as text
end tell`

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("reading_list_failed", err.Error(), nil)
			}

			if result == "" {
				return output.Print(map[string]any{
					"items": []ReadingListItem{},
					"count": 0,
				})
			}

			var items []ReadingListItem
			itemStrs := strings.Split(result, ":::")
			count := 0
			for _, item := range itemStrs {
				if limit > 0 && count >= limit {
					break
				}
				parts := strings.Split(item, "|||")
				if len(parts) >= 2 {
					items = append(items, ReadingListItem{
						Title: strings.TrimSpace(parts[0]),
						URL:   strings.TrimSpace(parts[1]),
					})
					count++
				}
			}

			return output.Print(map[string]any{
				"items": items,
				"count": len(items),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of items (0 = no limit)")

	return cmd
}

// newAddReadingCmd adds a URL to the Reading List
func newAddReadingCmd() *cobra.Command {
	var title string

	cmd := &cobra.Command{
		Use:   "add-reading [url]",
		Short: "Add URL to Safari Reading List",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]

			// Add https:// if no protocol specified
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "https://" + url
			}

			// Use the "Add to Reading List" functionality
			// This requires opening the URL first and then adding it
			script := fmt.Sprintf(`
tell application "Safari"
	activate
	-- Create a temporary document to add to reading list
	set tempDoc to make new document with properties {URL:"%s"}
	delay 1
	-- Use keyboard shortcut to add to reading list (Cmd+Shift+D)
	tell application "System Events"
		keystroke "d" using {command down, shift down}
	end tell
	delay 0.5
	-- Close the temporary document
	close tempDoc
	return "Added to Reading List"
end tell`, escapeAppleScript(url))

			result, err := runAppleScript(script)
			if err != nil {
				// Try alternative method using Safari's menu
				altScript := fmt.Sprintf(`
tell application "Safari"
	activate
	open location "%s"
	delay 1
end tell
tell application "System Events"
	tell process "Safari"
		click menu item "Add to Reading List" of menu "Bookmarks" of menu bar 1
	end tell
end tell
return "Added to Reading List"`, escapeAppleScript(url))

				result, err = runAppleScript(altScript)
				if err != nil {
					return output.PrintError("add_reading_failed", err.Error(),
						map[string]string{
							"url":        url,
							"suggestion": "Make sure Safari has accessibility permissions enabled",
						})
				}
			}

			displayTitle := title
			if displayTitle == "" {
				displayTitle = url
			}

			return output.Print(map[string]any{
				"success": true,
				"message": strings.TrimSpace(result),
				"url":     url,
				"title":   displayTitle,
			})
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Title for the reading list item (optional)")

	return cmd
}

// newHistoryCmd gets recent browser history
func newHistoryCmd() *cobra.Command {
	var limit int
	var days int
	var search string

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Get recent Safari history",
		Long:  "Get recent Safari browsing history. History is read from Safari's History.db SQLite database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return output.PrintError("home_dir_failed", "Failed to get home directory", nil)
			}

			historyDB := filepath.Join(homeDir, "Library", "Safari", "History.db")

			// Check if the file exists
			if _, err := os.Stat(historyDB); os.IsNotExist(err) {
				return output.PrintError("history_not_found",
					"Safari history database not found",
					map[string]string{
						"path":       historyDB,
						"suggestion": "Make sure Safari has been used and history is enabled",
					})
			}

			// Safari may have the database locked, so we need to copy it first
			tmpFile, err := os.CreateTemp("", "safari_history_*.db")
			if err != nil {
				return output.PrintError("temp_file_failed",
					"Failed to create temporary file",
					map[string]string{"error": err.Error()})
			}
			tmpDB := tmpFile.Name()
			tmpFile.Close()
			cpCmd := exec.Command("cp", historyDB, tmpDB)
			if err := cpCmd.Run(); err != nil {
				return output.PrintError("copy_failed",
					"Failed to copy history database",
					map[string]string{
						"error":      err.Error(),
						"suggestion": "Safari may be preventing access. Try closing Safari first.",
					})
			}
			defer os.Remove(tmpDB)

			// Open the copied database
			db, err := sql.Open("sqlite3", tmpDB+"?mode=ro")
			if err != nil {
				return output.PrintError("db_open_failed",
					"Failed to open history database",
					map[string]string{"error": err.Error()})
			}
			defer db.Close()

			// Calculate the time filter
			// Safari stores timestamps as seconds since January 1, 2001 (Mac absolute time)
			// We need to convert to this format
			macEpoch := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
			cutoffTime := time.Now().AddDate(0, 0, -days)
			cutoffMacTime := cutoffTime.Sub(macEpoch).Seconds()

			// Build the query
			query := `
				SELECT
					hi.url,
					hv.title,
					hv.visit_time,
					hi.visit_count
				FROM history_items hi
				JOIN history_visits hv ON hi.id = hv.history_item
				WHERE hv.visit_time > ?
			`
			queryArgs := []interface{}{cutoffMacTime}

			if search != "" {
				query += " AND (hv.title LIKE ? OR hi.url LIKE ?)"
				searchPattern := "%" + search + "%"
				queryArgs = append(queryArgs, searchPattern, searchPattern)
			}

			query += " ORDER BY hv.visit_time DESC"

			if limit > 0 {
				query += fmt.Sprintf(" LIMIT %d", limit)
			}

			rows, err := db.Query(query, queryArgs...)
			if err != nil {
				// Try alternative query structure (schema may vary)
				altQuery := `
					SELECT
						url,
						title,
						visit_time,
						visit_count
					FROM history_items
					WHERE visit_time > ?
				`
				if search != "" {
					altQuery += " AND (title LIKE ? OR url LIKE ?)"
				}
				altQuery += " ORDER BY visit_time DESC"
				if limit > 0 {
					altQuery += fmt.Sprintf(" LIMIT %d", limit)
				}

				rows, err = db.Query(altQuery, queryArgs...)
				if err != nil {
					return output.PrintError("query_failed",
						"Failed to query history database",
						map[string]string{"error": err.Error()})
				}
			}
			defer rows.Close()

			var items []HistoryItem
			for rows.Next() {
				var url, title sql.NullString
				var visitTime sql.NullFloat64
				var visitCount sql.NullInt64

				if err := rows.Scan(&url, &title, &visitTime, &visitCount); err != nil {
					continue
				}

				// Convert Mac absolute time to human readable
				var visitTimeStr string
				if visitTime.Valid {
					unixTime := macEpoch.Add(time.Duration(visitTime.Float64) * time.Second)
					visitTimeStr = unixTime.Format(time.RFC3339)
				}

				item := HistoryItem{
					URL:       url.String,
					Title:     title.String,
					VisitTime: visitTimeStr,
				}
				if visitCount.Valid {
					item.VisitCount = int(visitCount.Int64)
				}

				items = append(items, item)
			}
			if err := rows.Err(); err != nil {
				return output.PrintError("query_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"items":  items,
				"count":  len(items),
				"days":   days,
				"search": search,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Limit number of history items")
	cmd.Flags().IntVarP(&days, "days", "d", 7, "Number of days of history to retrieve")
	cmd.Flags().StringVarP(&search, "search", "s", "", "Search term to filter history")

	return cmd
}
