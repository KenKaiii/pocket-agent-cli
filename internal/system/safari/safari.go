package safari

import (
	"bytes"
	"database/sql"
	"encoding/json"
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
		Long:  "List Safari bookmarks from Bookmarks.plist. Requires Full Disk Access for your terminal.",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return output.PrintError("home_dir_failed", "Failed to get home directory", nil)
			}

			bookmarksPlist := filepath.Join(homeDir, "Library", "Safari", "Bookmarks.plist")

			// Convert plist to JSON for parsing
			plistCmd := exec.Command("plutil", "-convert", "json", "-o", "-", bookmarksPlist)
			var stdout, stderr bytes.Buffer
			plistCmd.Stdout = &stdout
			plistCmd.Stderr = &stderr

			if err := plistCmd.Run(); err != nil {
				errMsg := stderr.String()
				if strings.Contains(errMsg, "permission") || strings.Contains(errMsg, "couldn't be opened") {
					return output.PrintError("permission_denied",
						"Full Disk Access required to read Safari bookmarks",
						map[string]string{
							"path":       bookmarksPlist,
							"suggestion": "Go to System Settings > Privacy & Security > Full Disk Access and add your terminal",
						})
				}
				return output.PrintError("bookmarks_failed",
					"Failed to read bookmarks plist",
					map[string]string{"error": errMsg})
			}

			// Parse the bookmarks using shell script to extract from JSON
			// The plist has a nested structure: Children[] -> Children[] -> URLString, URIDictionary.title
			targetFolder := folder
			if targetFolder == "" {
				targetFolder = "BookmarksBar"
			}

			var bookmarks []Bookmark
			bookmarks = parseBookmarksJSON(stdout.Bytes(), targetFolder, limit)

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

	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Bookmark folder to list (default: Favorites/BookmarksBar)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of bookmarks (0 = no limit)")

	return cmd
}

// plistNode represents a node in Safari's Bookmarks.plist JSON structure
type plistNode struct {
	Title           string            `json:"Title"`
	WebBookmarkType string            `json:"WebBookmarkType"`
	URLString       string            `json:"URLString"`
	URIDictionary   map[string]string `json:"URIDictionary"`
	Children        []plistNode       `json:"Children"`
	ReadingList     map[string]any    `json:"ReadingList"`
}

// parseBookmarksJSON extracts bookmarks from the plist JSON data
func parseBookmarksJSON(data []byte, targetFolder string, limit int) []Bookmark {
	var root plistNode
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}

	var bookmarks []Bookmark
	collectBookmarks(&root, targetFolder, &bookmarks, limit)
	return bookmarks
}

func collectBookmarks(node *plistNode, targetFolder string, bookmarks *[]Bookmark, limit int) {
	if limit > 0 && len(*bookmarks) >= limit {
		return
	}

	// Check if this is the target folder
	isTarget := false
	if node.WebBookmarkType == "WebBookmarkTypeList" {
		if targetFolder == "BookmarksBar" && node.Title == "BookmarksBar" {
			isTarget = true
		} else if targetFolder == "BookmarksMenu" && node.Title == "BookmarksMenu" {
			isTarget = true
		} else if node.Title == targetFolder {
			isTarget = true
		}
	}

	if isTarget {
		for _, child := range node.Children {
			if limit > 0 && len(*bookmarks) >= limit {
				return
			}
			if child.WebBookmarkType == "WebBookmarkTypeLeaf" && child.URLString != "" {
				title := ""
				if child.URIDictionary != nil {
					title = child.URIDictionary["title"]
				}
				if title == "" {
					title = child.Title
				}
				*bookmarks = append(*bookmarks, Bookmark{
					Title:  title,
					URL:    child.URLString,
					Folder: node.Title,
				})
			}
		}
		return
	}

	// Recurse into children
	for i := range node.Children {
		collectBookmarks(&node.Children[i], targetFolder, bookmarks, limit)
	}
}

// parseReadingListJSON extracts reading list items from the plist JSON data
func parseReadingListJSON(data []byte, limit int) []ReadingListItem {
	var root plistNode
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}

	var items []ReadingListItem
	collectReadingList(&root, &items, limit)
	return items
}

func collectReadingList(node *plistNode, items *[]ReadingListItem, limit int) {
	if limit > 0 && len(*items) >= limit {
		return
	}

	// Find the com.apple.ReadingList folder
	if node.WebBookmarkType == "WebBookmarkTypeList" && node.Title == "com.apple.ReadingList" {
		for _, child := range node.Children {
			if limit > 0 && len(*items) >= limit {
				return
			}
			if child.URLString != "" {
				title := ""
				if child.URIDictionary != nil {
					title = child.URIDictionary["title"]
				}
				if title == "" {
					title = child.Title
				}
				item := ReadingListItem{
					Title: title,
					URL:   child.URLString,
				}
				if child.ReadingList != nil {
					if preview, ok := child.ReadingList["PreviewText"].(string); ok {
						item.Preview = preview
					}
					if dateAdded, ok := child.ReadingList["DateAdded"].(string); ok {
						item.DateAdded = dateAdded
					}
				}
				*items = append(*items, item)
			}
		}
		return
	}

	for i := range node.Children {
		collectReadingList(&node.Children[i], items, limit)
	}
}

// newReadingListCmd lists Reading List items
func newReadingListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "reading-list",
		Short: "List Safari Reading List items",
		Long:  "List Safari Reading List items from Bookmarks.plist. Requires Full Disk Access for your terminal.",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return output.PrintError("home_dir_failed", "Failed to get home directory", nil)
			}

			bookmarksPlist := filepath.Join(homeDir, "Library", "Safari", "Bookmarks.plist")

			// Convert plist to JSON for parsing
			plistCmd := exec.Command("plutil", "-convert", "json", "-o", "-", bookmarksPlist)
			var stdout, stderr bytes.Buffer
			plistCmd.Stdout = &stdout
			plistCmd.Stderr = &stderr

			if err := plistCmd.Run(); err != nil {
				errMsg := stderr.String()
				if strings.Contains(errMsg, "permission") || strings.Contains(errMsg, "couldn't be opened") {
					return output.PrintError("permission_denied",
						"Full Disk Access required to read Safari Reading List",
						map[string]string{
							"path":       bookmarksPlist,
							"suggestion": "Go to System Settings > Privacy & Security > Full Disk Access and add your terminal",
						})
				}
				return output.PrintError("reading_list_failed",
					"Failed to read bookmarks plist",
					map[string]string{"error": errMsg})
			}

			items := parseReadingListJSON(stdout.Bytes(), limit)

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
				os.Remove(tmpDB)
				return output.PrintError("permission_denied",
					"Full Disk Access required to read Safari history",
					map[string]string{
						"path":       historyDB,
						"suggestion": "Go to System Settings > Privacy & Security > Full Disk Access and add your terminal",
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
