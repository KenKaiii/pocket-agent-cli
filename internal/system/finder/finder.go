package finder

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

// FileInfo represents information about a file or folder
type FileInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Type         string `json:"type"`
	Size         int64  `json:"size"`
	SizeHuman    string `json:"size_human"`
	Created      string `json:"created,omitempty"`
	Modified     string `json:"modified,omitempty"`
	Accessed     string `json:"accessed,omitempty"`
	Permissions  string `json:"permissions,omitempty"`
	IsHidden     bool   `json:"is_hidden"`
	IsExecutable bool   `json:"is_executable,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
}

// DirectoryEntry represents an entry in a directory listing
type DirectoryEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Type      string `json:"type"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"size_human"`
	Modified  string `json:"modified"`
	IsHidden  bool   `json:"is_hidden"`
}

// SearchResult represents a Spotlight search result
type SearchResult struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Modified    string `json:"modified,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// NewCmd creates the finder command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "finder",
		Aliases: []string{"find", "fs"},
		Short:   "Finder/filesystem commands (macOS only)",
		Long:    `Interact with Finder and the filesystem on macOS. Uses AppleScript and shell commands.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return output.PrintError("platform_unsupported",
					"Finder integration is only available on macOS",
					map[string]string{
						"current_platform": runtime.GOOS,
						"required":         "darwin (macOS)",
					})
			}
			return nil
		},
	}

	cmd.AddCommand(newOpenCmd())
	cmd.AddCommand(newRevealCmd())
	cmd.AddCommand(newInfoCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newTagsCmd())
	cmd.AddCommand(newTagCmd())
	cmd.AddCommand(newUntagCmd())
	cmd.AddCommand(newTrashCmd())
	cmd.AddCommand(newSearchCmd())

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
		return "", fmt.Errorf("%s", strings.TrimSpace(errMsg))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// runCommand executes a shell command and returns the output
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
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

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// resolvePath resolves and expands a file path
func resolvePath(path string) (string, error) {
	// Expand home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

// humanReadableSize converts bytes to human-readable format
func humanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// newOpenCmd opens a file or folder in Finder
func newOpenCmd() *cobra.Command {
	var withApp string

	cmd := &cobra.Command{
		Use:   "open [path]",
		Short: "Open a file or folder in Finder",
		Long:  `Opens a file or folder in Finder. Optionally specify an application to open with.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolvePath(args[0])
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}

			// Check if path exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", path),
					map[string]string{"path": path})
			}

			var openCmd *exec.Cmd
			if withApp != "" {
				openCmd = exec.Command("open", "-a", withApp, path)
			} else {
				openCmd = exec.Command("open", path)
			}

			if err := openCmd.Run(); err != nil {
				return output.PrintError("open_failed", err.Error(),
					map[string]string{"path": path})
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Opened successfully",
				"path":    path,
				"app":     withApp,
			})
		},
	}

	cmd.Flags().StringVarP(&withApp, "app", "a", "", "Application to open with")

	return cmd
}

// newRevealCmd reveals a file in Finder
func newRevealCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reveal [path]",
		Short: "Reveal a file in Finder",
		Long:  `Opens a Finder window and selects the specified file or folder.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolvePath(args[0])
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}

			// Check if path exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", path),
					map[string]string{"path": path})
			}

			script := fmt.Sprintf(`
tell application "Finder"
	reveal POSIX file "%s"
	activate
end tell`, escapeAppleScript(path))

			_, err = runAppleScript(script)
			if err != nil {
				return output.PrintError("reveal_failed", err.Error(),
					map[string]string{"path": path})
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "File revealed in Finder",
				"path":    path,
			})
		},
	}

	return cmd
}

// newInfoCmd gets file/folder information
func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [path]",
		Short: "Get file or folder information",
		Long:  `Returns detailed information about a file or folder including size, dates, and type.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolvePath(args[0])
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}

			// Check if path exists
			stat, err := os.Stat(path)
			if os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", path),
					map[string]string{"path": path})
			}
			if err != nil {
				return output.PrintError("stat_failed", err.Error(),
					map[string]string{"path": path})
			}

			info := FileInfo{
				Name:        stat.Name(),
				Path:        path,
				Size:        stat.Size(),
				SizeHuman:   humanReadableSize(stat.Size()),
				Modified:    stat.ModTime().Format(time.RFC3339),
				Permissions: stat.Mode().Perm().String(),
				IsHidden:    strings.HasPrefix(stat.Name(), "."),
			}

			if stat.IsDir() {
				info.Type = "directory"
				// Calculate directory size (capped at 100k files to prevent blocking)
				var size int64
				var fileCount int
				filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					fileCount++
					if fileCount >= 100000 {
						return fmt.Errorf("limit reached")
					}
					if !info.IsDir() {
						size += info.Size()
					}
					return nil
				})
				info.Size = size
				info.SizeHuman = humanReadableSize(size)
			} else {
				info.Type = "file"
				info.IsExecutable = stat.Mode().Perm()&0111 != 0
			}

			// Get extended metadata using mdls (outputs fields in alphabetical order)
			mdlsOutput, err := runCommand("mdls", "-name", "kMDItemContentType",
				"-name", "kMDItemContentCreationDate",
				"-name", "kMDItemLastUsedDate", path)
			if err == nil {
				for _, line := range strings.Split(mdlsOutput, "\n") {
					line = strings.TrimSpace(line)
					if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						val := strings.TrimSpace(parts[1])
						val = strings.Trim(val, "\"")
						if val == "(null)" {
							continue
						}
						switch key {
						case "kMDItemContentType":
							info.ContentType = val
						case "kMDItemContentCreationDate":
							if t, err := time.Parse("2006-01-02 15:04:05 -0700", val); err == nil {
								info.Created = t.Format(time.RFC3339)
							}
						case "kMDItemLastUsedDate":
							if t, err := time.Parse("2006-01-02 15:04:05 -0700", val); err == nil {
								info.Accessed = t.Format(time.RFC3339)
							}
						}
					}
				}
			}

			return output.Print(info)
		},
	}

	return cmd
}

// newListCmd lists directory contents
func newListCmd() *cobra.Command {
	var showHidden bool
	var sortBy string
	var limit int

	cmd := &cobra.Command{
		Use:   "list [path]",
		Short: "List directory contents",
		Long:  `Lists the contents of a directory with details including size and modification date.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			resolvedPath, err := resolvePath(path)
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}

			// Check if path exists and is a directory
			stat, err := os.Stat(resolvedPath)
			if os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", resolvedPath),
					map[string]string{"path": resolvedPath})
			}
			if err != nil {
				return output.PrintError("stat_failed", err.Error(),
					map[string]string{"path": resolvedPath})
			}
			if !stat.IsDir() {
				return output.PrintError("not_directory",
					fmt.Sprintf("Path is not a directory: %s", resolvedPath),
					map[string]string{"path": resolvedPath})
			}

			entries, err := os.ReadDir(resolvedPath)
			if err != nil {
				return output.PrintError("read_dir_failed", err.Error(),
					map[string]string{"path": resolvedPath})
			}

			var results []DirectoryEntry
			for _, entry := range entries {
				name := entry.Name()

				// Skip hidden files unless requested
				if !showHidden && strings.HasPrefix(name, ".") {
					continue
				}

				info, err := entry.Info()
				if err != nil {
					continue
				}

				entryType := "file"
				if entry.IsDir() {
					entryType = "directory"
				} else if info.Mode()&os.ModeSymlink != 0 {
					entryType = "symlink"
				}

				results = append(results, DirectoryEntry{
					Name:      name,
					Path:      filepath.Join(resolvedPath, name),
					Type:      entryType,
					Size:      info.Size(),
					SizeHuman: humanReadableSize(info.Size()),
					Modified:  info.ModTime().Format(time.RFC3339),
					IsHidden:  strings.HasPrefix(name, "."),
				})

				if limit > 0 && len(results) >= limit {
					break
				}
			}

			return output.Print(map[string]any{
				"path":    resolvedPath,
				"entries": results,
				"count":   len(results),
			})
		},
	}

	cmd.Flags().BoolVarP(&showHidden, "hidden", "a", false, "Show hidden files")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "name", "Sort by: name, size, modified")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of entries (0 = no limit)")

	return cmd
}

// newTagsCmd gets Finder tags for a file
func newTagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags [path]",
		Short: "Get Finder tags for a file",
		Long:  `Returns the Finder tags (labels) associated with a file or folder.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolvePath(args[0])
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}

			// Check if path exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", path),
					map[string]string{"path": path})
			}

			// Get tags using mdls
			mdlsOutput, err := runCommand("mdls", "-name", "kMDItemUserTags", "-raw", path)
			if err != nil {
				return output.PrintError("mdls_failed", err.Error(),
					map[string]string{"path": path})
			}

			var tags []string
			if mdlsOutput != "(null)" && mdlsOutput != "" {
				// Parse the plist-style array output
				mdlsOutput = strings.Trim(mdlsOutput, "()")
				if mdlsOutput != "" {
					parts := strings.Split(mdlsOutput, ",")
					for _, part := range parts {
						tag := strings.TrimSpace(part)
						tag = strings.Trim(tag, "\"")
						// Remove color suffix if present (e.g., "Red\n6")
						if idx := strings.Index(tag, "\n"); idx != -1 {
							tag = tag[:idx]
						}
						if tag != "" {
							tags = append(tags, tag)
						}
					}
				}
			}

			return output.Print(map[string]any{
				"path":  path,
				"tags":  tags,
				"count": len(tags),
			})
		},
	}

	return cmd
}

// newTagCmd adds a tag to a file
func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag [path] [tag]",
		Short: "Add a tag to a file",
		Long:  `Adds a Finder tag (label) to a file or folder.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolvePath(args[0])
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}
			tag := args[1]

			// Check if path exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", path),
					map[string]string{"path": path})
			}

			// Use xattr to add the tag
			// First get existing tags
			existingOutput, _ := runCommand("mdls", "-name", "kMDItemUserTags", "-raw", path)
			var existingTags []string
			if existingOutput != "(null)" && existingOutput != "" {
				existingOutput = strings.Trim(existingOutput, "()")
				if existingOutput != "" {
					parts := strings.Split(existingOutput, ",")
					for _, part := range parts {
						t := strings.TrimSpace(part)
						t = strings.Trim(t, "\"")
						if idx := strings.Index(t, "\n"); idx != -1 {
							t = t[:idx]
						}
						if t != "" {
							existingTags = append(existingTags, t)
						}
					}
				}
			}

			// Check if tag already exists
			for _, t := range existingTags {
				if t == tag {
					return output.Print(map[string]any{
						"success": true,
						"message": "Tag already exists",
						"path":    path,
						"tag":     tag,
					})
				}
			}

			// Add the new tag using xattr with plist format
			existingTags = append(existingTags, tag)
			plistTags := "<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\"><plist version=\"1.0\"><array>"
			for _, t := range existingTags {
				plistTags += fmt.Sprintf("<string>%s</string>", html.EscapeString(t))
			}
			plistTags += "</array></plist>"

			xattrCmd := exec.Command("xattr", "-w", "com.apple.metadata:_kMDItemUserTags", plistTags, path)
			if err := xattrCmd.Run(); err != nil {
				return output.PrintError("tag_failed", err.Error(),
					map[string]string{"path": path, "tag": tag})
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Tag added successfully",
				"path":    path,
				"tag":     tag,
			})
		},
	}

	return cmd
}

// newUntagCmd removes a tag from a file
func newUntagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "untag [path] [tag]",
		Short: "Remove a tag from a file",
		Long:  `Removes a Finder tag (label) from a file or folder.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolvePath(args[0])
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}
			tag := args[1]

			// Check if path exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", path),
					map[string]string{"path": path})
			}

			// Get existing tags
			existingOutput, _ := runCommand("mdls", "-name", "kMDItemUserTags", "-raw", path)
			var existingTags []string
			if existingOutput != "(null)" && existingOutput != "" {
				existingOutput = strings.Trim(existingOutput, "()")
				if existingOutput != "" {
					parts := strings.Split(existingOutput, ",")
					for _, part := range parts {
						t := strings.TrimSpace(part)
						t = strings.Trim(t, "\"")
						if idx := strings.Index(t, "\n"); idx != -1 {
							t = t[:idx]
						}
						if t != "" {
							existingTags = append(existingTags, t)
						}
					}
				}
			}

			// Remove the tag
			var newTags []string
			found := false
			for _, t := range existingTags {
				if t == tag {
					found = true
				} else {
					newTags = append(newTags, t)
				}
			}

			if !found {
				return output.Print(map[string]any{
					"success": true,
					"message": "Tag not found on file",
					"path":    path,
					"tag":     tag,
				})
			}

			// Update tags using xattr
			if len(newTags) == 0 {
				// Remove the attribute entirely
				xattrCmd := exec.Command("xattr", "-d", "com.apple.metadata:_kMDItemUserTags", path)
				xattrCmd.Run() // Ignore error if attribute doesn't exist
			} else {
				plistTags := "<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\"><plist version=\"1.0\"><array>"
				for _, t := range newTags {
					plistTags += fmt.Sprintf("<string>%s</string>", html.EscapeString(t))
				}
				plistTags += "</array></plist>"

				xattrCmd := exec.Command("xattr", "-w", "com.apple.metadata:_kMDItemUserTags", plistTags, path)
				if err := xattrCmd.Run(); err != nil {
					return output.PrintError("untag_failed", err.Error(),
						map[string]string{"path": path, "tag": tag})
				}
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Tag removed successfully",
				"path":    path,
				"tag":     tag,
			})
		},
	}

	return cmd
}

// newTrashCmd moves a file to trash
func newTrashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trash [path]",
		Short: "Move a file to trash",
		Long:  `Moves a file or folder to the Trash using Finder.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolvePath(args[0])
			if err != nil {
				return output.PrintError("invalid_path", err.Error(), nil)
			}

			// Check if path exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return output.PrintError("path_not_found",
					fmt.Sprintf("Path does not exist: %s", path),
					map[string]string{"path": path})
			}

			script := fmt.Sprintf(`
tell application "Finder"
	move POSIX file "%s" to trash
end tell`, escapeAppleScript(path))

			_, err = runAppleScript(script)
			if err != nil {
				return output.PrintError("trash_failed", err.Error(),
					map[string]string{"path": path})
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "File moved to trash",
				"path":    path,
			})
		},
	}

	return cmd
}

// newSearchCmd performs a Spotlight search
func newSearchCmd() *cobra.Command {
	var searchPath string
	var kind string
	var limit int
	var onlyIn string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Spotlight search",
		Long:  `Searches for files using Spotlight (mdfind). Supports name matching and content search.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.ReplaceAll(args[0], "'", "")

			// Build mdfind command
			mdfindArgs := []string{}

			// Add path restriction if specified
			if searchPath != "" {
				resolvedPath, err := resolvePath(searchPath)
				if err != nil {
					return output.PrintError("invalid_path", err.Error(), nil)
				}
				mdfindArgs = append(mdfindArgs, "-onlyin", resolvedPath)
			} else if onlyIn != "" {
				resolvedPath, err := resolvePath(onlyIn)
				if err != nil {
					return output.PrintError("invalid_path", err.Error(), nil)
				}
				mdfindArgs = append(mdfindArgs, "-onlyin", resolvedPath)
			}

			// Build query string
			queryStr := query
			if kind != "" {
				kindMap := map[string]string{
					"app":         "kMDItemContentType == 'com.apple.application-bundle'",
					"application": "kMDItemContentType == 'com.apple.application-bundle'",
					"folder":      "kMDItemContentType == 'public.folder'",
					"directory":   "kMDItemContentType == 'public.folder'",
					"image":       "kMDItemContentType == 'public.image'",
					"audio":       "kMDItemContentType == 'public.audio'",
					"video":       "kMDItemContentType == 'public.movie'",
					"pdf":         "kMDItemContentType == 'com.adobe.pdf'",
					"document":    "kMDItemContentType == 'public.content'",
					"text":        "kMDItemContentType == 'public.text'",
				}
				if kindQuery, ok := kindMap[strings.ToLower(kind)]; ok {
					queryStr = fmt.Sprintf("(%s) && (kMDItemDisplayName == '*%s*'wcd || kMDItemTextContent == '*%s*'wcd)",
						kindQuery, query, query)
				}
			} else {
				// Default search: name or content
				queryStr = fmt.Sprintf("kMDItemDisplayName == '*%s*'wcd || kMDItemTextContent == '*%s*'wcd",
					query, query)
			}

			mdfindArgs = append(mdfindArgs, queryStr)

			result, err := runCommand("mdfind", mdfindArgs...)
			if err != nil {
				return output.PrintError("search_failed", err.Error(),
					map[string]string{"query": query})
			}

			var results []SearchResult
			if result != "" {
				lines := strings.Split(result, "\n")
				for i, line := range lines {
					if limit > 0 && i >= limit {
						break
					}
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}

					sr := SearchResult{
						Path: line,
						Name: filepath.Base(line),
					}

					// Get additional metadata
					mdlsOutput, err := runCommand("mdls",
						"-name", "kMDItemContentType",
						"-name", "kMDItemContentModificationDate",
						"-name", "kMDItemKind",
						"-raw", line)
					if err == nil {
						mdlsLines := strings.Split(mdlsOutput, "\n")
						for j, mdLine := range mdlsLines {
							mdLine = strings.TrimSpace(mdLine)
							if mdLine == "(null)" {
								continue
							}
							switch j {
							case 0:
								sr.ContentType = mdLine
							case 1:
								if t, err := time.Parse("2006-01-02 15:04:05 -0700", mdLine); err == nil {
									sr.Modified = t.Format(time.RFC3339)
								}
							case 2:
								sr.Kind = mdLine
							}
						}
					}

					results = append(results, sr)
				}
			}

			return output.Print(map[string]any{
				"query":   query,
				"results": results,
				"count":   len(results),
				"path":    searchPath,
				"kind":    kind,
			})
		},
	}

	cmd.Flags().StringVarP(&searchPath, "path", "p", "", "Search within this path")
	cmd.Flags().StringVarP(&onlyIn, "in", "i", "", "Search within this path (alias for --path)")
	cmd.Flags().StringVarP(&kind, "kind", "k", "", "Filter by kind: app, folder, image, audio, video, pdf, document, text")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Limit number of results (default 50)")

	return cmd
}

// parseMdlsOutput parses mdls output for a single attribute
func parseMdlsOutput(output, attrName string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, attrName) {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, "\"")
				if value == "(null)" {
					return ""
				}
				return value
			}
		}
	}
	return ""
}

// parseSize parses a size string from mdls
func parseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "(null)" {
		return 0
	}
	size, _ := strconv.ParseInt(s, 10, 64)
	return size
}
