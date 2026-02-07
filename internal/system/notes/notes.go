package notes

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

// Note represents a note in Apple Notes
type Note struct {
	Name       string `json:"name"`
	Body       string `json:"body,omitempty"`
	Folder     string `json:"folder,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
}

// Folder represents a folder in Apple Notes
type Folder struct {
	Name  string `json:"name"`
	ID    string `json:"id,omitempty"`
	Count int    `json:"count,omitempty"`
}

// NewCmd creates the notes command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notes",
		Aliases: []string{"note", "applenotes"},
		Short:   "Apple Notes commands (macOS only)",
		Long:    `Interact with Apple Notes via AppleScript. Only available on macOS.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return output.PrintError("platform_unsupported",
					"Apple Notes is only available on macOS",
					map[string]string{"current_platform": runtime.GOOS})
			}
			return nil
		},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newFoldersCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newAppendCmd())

	return cmd
}

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	// Escape backslashes first, then quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
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
		return "", fmt.Errorf("%s", strings.TrimSpace(errMsg))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// newListCmd lists all notes
func newListCmd() *cobra.Command {
	var folder string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			var script string

			if folder != "" {
				// List notes in a specific folder
				script = fmt.Sprintf(`
tell application "Notes"
	set noteList to {}
	set folderName to "%s"
	repeat with theFolder in folders
		if name of theFolder is folderName then
			repeat with theNote in notes of theFolder
				set noteName to name of theNote
				set noteModDate to modification date of theNote as string
				set end of noteList to noteName & "|||" & folderName & "|||" & noteModDate
			end repeat
		end if
	end repeat
	set AppleScript's text item delimiters to ":::"
	return noteList as text
end tell`, escapeAppleScript(folder))
			} else {
				// List all notes by iterating through each folder
				script = `
tell application "Notes"
	set noteList to {}
	repeat with theFolder in folders
		set folderName to name of theFolder
		if folderName is not "Recently Deleted" then
			repeat with theNote in notes of theFolder
				set noteName to name of theNote
				set noteModDate to modification date of theNote as string
				set end of noteList to noteName & "|||" & folderName & "|||" & noteModDate
			end repeat
		end if
	end repeat
	set AppleScript's text item delimiters to ":::"
	return noteList as text
end tell`
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("list_failed", err.Error(), nil)
			}

			if result == "" {
				return output.Print([]Note{})
			}

			// Parse the result
			var notes []Note
			items := strings.Split(result, ":::")
			count := 0
			for _, item := range items {
				if limit > 0 && count >= limit {
					break
				}
				parts := strings.Split(item, "|||")
				if len(parts) >= 3 {
					notes = append(notes, Note{
						Name:       strings.TrimSpace(parts[0]),
						Folder:     strings.TrimSpace(parts[1]),
						ModifiedAt: strings.TrimSpace(parts[2]),
					})
					count++
				}
			}

			return output.Print(notes)
		},
	}

	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Filter by folder name")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of notes (0 = no limit)")

	return cmd
}

// newFoldersCmd lists all folders
func newFoldersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folders",
		Short: "List all folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			script := `
tell application "Notes"
	set folderList to {}
	repeat with theFolder in folders
		set folderName to name of theFolder
		set noteCount to count of notes of theFolder
		set end of folderList to folderName & "|||" & noteCount
	end repeat
	set AppleScript's text item delimiters to ":::"
	return folderList as text
end tell`

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("folders_failed", err.Error(), nil)
			}

			if result == "" {
				return output.Print([]Folder{})
			}

			// Parse the result
			var folders []Folder
			items := strings.Split(result, ":::")
			for _, item := range items {
				parts := strings.Split(item, "|||")
				if len(parts) >= 2 {
					count := 0
					fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &count)
					folders = append(folders, Folder{
						Name:  strings.TrimSpace(parts[0]),
						Count: count,
					})
				}
			}

			return output.Print(folders)
		},
	}

	return cmd
}

// newReadCmd reads a note by name
func newReadCmd() *cobra.Command {
	var folder string

	cmd := &cobra.Command{
		Use:   "read [name]",
		Short: "Read a note by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteName := args[0]

			var script string
			if folder != "" {
				script = fmt.Sprintf(`
tell application "Notes"
	set targetName to "%s"
	set folderName to "%s"
	repeat with theFolder in folders
		if name of theFolder is folderName then
			repeat with theNote in notes of theFolder
				if name of theNote is targetName then
					set noteBody to plaintext of theNote
					set noteCreated to creation date of theNote as string
					set noteModified to modification date of theNote as string
					return targetName & "|||" & noteBody & "|||" & folderName & "|||" & noteCreated & "|||" & noteModified
				end if
			end repeat
		end if
	end repeat
	return "NOT_FOUND"
end tell`, escapeAppleScript(noteName), escapeAppleScript(folder))
			} else {
				script = fmt.Sprintf(`
tell application "Notes"
	set targetName to "%s"
	repeat with theFolder in folders
		set folderName to name of theFolder
		if folderName is not "Recently Deleted" then
			repeat with theNote in notes of theFolder
				if name of theNote is targetName then
					set noteBody to plaintext of theNote
					set noteCreated to creation date of theNote as string
					set noteModified to modification date of theNote as string
					return targetName & "|||" & noteBody & "|||" & folderName & "|||" & noteCreated & "|||" & noteModified
				end if
			end repeat
		end if
	end repeat
	return "NOT_FOUND"
end tell`, escapeAppleScript(noteName))
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("read_failed", err.Error(), nil)
			}

			if result == "NOT_FOUND" {
				return output.PrintError("note_not_found",
					fmt.Sprintf("Note not found: %s", noteName),
					map[string]string{"name": noteName, "folder": folder})
			}

			parts := strings.Split(result, "|||")
			if len(parts) < 5 {
				return output.PrintError("parse_failed", "Failed to parse note data", nil)
			}

			note := Note{
				Name:       strings.TrimSpace(parts[0]),
				Body:       strings.TrimSpace(parts[1]),
				Folder:     strings.TrimSpace(parts[2]),
				CreatedAt:  strings.TrimSpace(parts[3]),
				ModifiedAt: strings.TrimSpace(parts[4]),
			}

			return output.Print(note)
		},
	}

	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Folder to search in")

	return cmd
}

// newCreateCmd creates a new note
func newCreateCmd() *cobra.Command {
	var folder string

	cmd := &cobra.Command{
		Use:   "create [name] [body]",
		Short: "Create a new note",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteName := args[0]
			noteBody := args[1]

			// Format the body with HTML (Notes uses HTML internally)
			// Convert newlines to <br> for proper formatting
			htmlBody := strings.ReplaceAll(escapeAppleScript(noteBody), "\\n", "<br>")

			var script string
			if folder != "" {
				script = fmt.Sprintf(`
tell application "Notes"
	set theFolder to folder "%s"
	set theNote to make new note at theFolder with properties {name:"%s", body:"%s"}
	return name of theNote
end tell`, escapeAppleScript(folder), escapeAppleScript(noteName), htmlBody)
			} else {
				// Create in default folder
				script = fmt.Sprintf(`
tell application "Notes"
	set theNote to make new note with properties {name:"%s", body:"%s"}
	return name of theNote
end tell`, escapeAppleScript(noteName), htmlBody)
			}

			result, err := runAppleScript(script)
			if err != nil {
				if strings.Contains(err.Error(), "Can't get folder") {
					return output.PrintError("folder_not_found",
						fmt.Sprintf("Folder not found: %s", folder),
						map[string]string{"folder": folder})
				}
				return output.PrintError("create_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Note created successfully",
				"name":    result,
				"folder":  folder,
			})
		},
	}

	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Folder to create note in (default: Notes folder)")

	return cmd
}

// newSearchCmd searches notes
func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search notes by name or content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.ToLower(args[0])

			script := fmt.Sprintf(`
tell application "Notes"
	set matchingNotes to {}
	set searchQuery to "%s"
	repeat with theFolder in folders
		set folderName to name of theFolder
		if folderName is not "Recently Deleted" then
			repeat with theNote in notes of theFolder
				set noteName to name of theNote
				set noteBody to plaintext of theNote
				set lowerName to do shell script "echo " & quoted form of noteName & " | tr '[:upper:]' '[:lower:]'"
				set lowerBody to do shell script "echo " & quoted form of noteBody & " | tr '[:upper:]' '[:lower:]'"
				if lowerName contains searchQuery or lowerBody contains searchQuery then
					set noteModDate to modification date of theNote as string
					set end of matchingNotes to noteName & "|||" & folderName & "|||" & noteModDate
				end if
			end repeat
		end if
	end repeat
	set AppleScript's text item delimiters to ":::"
	return matchingNotes as text
end tell`, escapeAppleScript(query))

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("search_failed", err.Error(), nil)
			}

			if result == "" {
				return output.Print(map[string]any{
					"query":   query,
					"results": []Note{},
					"count":   0,
				})
			}

			// Parse the result
			var notes []Note
			items := strings.Split(result, ":::")
			count := 0
			for _, item := range items {
				if limit > 0 && count >= limit {
					break
				}
				parts := strings.Split(item, "|||")
				if len(parts) >= 3 {
					notes = append(notes, Note{
						Name:       strings.TrimSpace(parts[0]),
						Folder:     strings.TrimSpace(parts[1]),
						ModifiedAt: strings.TrimSpace(parts[2]),
					})
					count++
				}
			}

			return output.Print(map[string]any{
				"query":   query,
				"results": notes,
				"count":   len(notes),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of results (0 = no limit)")

	return cmd
}

// newAppendCmd appends text to an existing note
func newAppendCmd() *cobra.Command {
	var folder string

	cmd := &cobra.Command{
		Use:   "append [name] [text]",
		Short: "Append text to an existing note",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteName := args[0]
			textToAppend := args[1]

			// Convert newlines to <br> for HTML formatting
			htmlText := strings.ReplaceAll(escapeAppleScript(textToAppend), "\\n", "<br>")

			var script string
			if folder != "" {
				script = fmt.Sprintf(`
tell application "Notes"
	set theFolder to folder "%s"
	set theNote to first note of theFolder whose name is "%s"
	set currentBody to body of theNote
	set body of theNote to currentBody & "<br>" & "%s"
	return name of theNote
end tell`, escapeAppleScript(folder), escapeAppleScript(noteName), htmlText)
			} else {
				script = fmt.Sprintf(`
tell application "Notes"
	set theNote to first note whose name is "%s"
	set currentBody to body of theNote
	set body of theNote to currentBody & "<br>" & "%s"
	return name of theNote
end tell`, escapeAppleScript(noteName), htmlText)
			}

			result, err := runAppleScript(script)
			if err != nil {
				if strings.Contains(err.Error(), "Can't get") {
					return output.PrintError("note_not_found",
						fmt.Sprintf("Note not found: %s", noteName),
						map[string]string{"name": noteName, "folder": folder})
				}
				return output.PrintError("append_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Text appended successfully",
				"name":    result,
			})
		},
	}

	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Folder to search in")

	return cmd
}
