package feeds

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var (
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
	parser    = gofeed.NewParser()
)

// FeedItem is the LLM-friendly output for a feed item
type FeedItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Author  string `json:"author,omitempty"`
	Summary string `json:"summary,omitempty"`
	Age     string `json:"age"`
}

// FeedInfo is the LLM-friendly output for feed metadata
type FeedInfo struct {
	Title   string     `json:"title"`
	URL     string     `json:"url"`
	Desc    string     `json:"desc,omitempty"`
	Items   []FeedItem `json:"items"`
	Updated string     `json:"updated,omitempty"`
}

// SavedFeed represents a saved feed in config
type SavedFeed struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "feeds",
		Aliases: []string{"rss", "feed"},
		Short:   "RSS/Atom feed commands",
	}

	cmd.AddCommand(newFetchCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newReadCmd())

	return cmd
}

func newFetchCmd() *cobra.Command {
	var limit int
	var summaryLen int

	cmd := &cobra.Command{
		Use:   "fetch [url]",
		Short: "Fetch and parse a feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchFeed(args[0], limit, summaryLen)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of items")
	cmd.Flags().IntVarP(&summaryLen, "summary", "s", 150, "Summary max length (0 = none)")

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			feeds, err := loadSavedFeeds()
			if err != nil {
				return output.PrintError("load_failed", err.Error(), nil)
			}
			if len(feeds) == 0 {
				return output.Print([]SavedFeed{})
			}
			return output.Print(feeds)
		},
	}

	return cmd
}

func newAddCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "add [url]",
		Short: "Save a feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]

			// If no name provided, fetch the feed to get its title
			if name == "" {
				feed, err := parser.ParseURL(url)
				if err != nil {
					return output.PrintError("fetch_failed", err.Error(), nil)
				}
				name = feed.Title
			}

			feeds, _ := loadSavedFeeds()

			// Check for duplicates
			for _, f := range feeds {
				if f.URL == url {
					return output.PrintError("duplicate", "Feed already saved", nil)
				}
			}

			feeds = append(feeds, SavedFeed{Name: name, URL: url})

			if err := saveSavedFeeds(feeds); err != nil {
				return output.PrintError("save_failed", err.Error(), nil)
			}

			return output.Print(map[string]string{
				"status": "added",
				"name":   name,
				"url":    url,
			})
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Feed name (auto-detected if not set)")

	return cmd
}

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove [name-or-url]",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a saved feed",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			feeds, err := loadSavedFeeds()
			if err != nil {
				return output.PrintError("load_failed", err.Error(), nil)
			}

			var newFeeds []SavedFeed
			var removed *SavedFeed

			for _, f := range feeds {
				if f.Name == query || f.URL == query {
					removed = &f
				} else {
					newFeeds = append(newFeeds, f)
				}
			}

			if removed == nil {
				return output.PrintError("not_found", "Feed not found", nil)
			}

			if err := saveSavedFeeds(newFeeds); err != nil {
				return output.PrintError("save_failed", err.Error(), nil)
			}

			return output.Print(map[string]string{
				"status": "removed",
				"name":   removed.Name,
				"url":    removed.URL,
			})
		},
	}

	return cmd
}

func newReadCmd() *cobra.Command {
	var limit int
	var summaryLen int

	cmd := &cobra.Command{
		Use:   "read [name]",
		Short: "Fetch a saved feed by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			feeds, err := loadSavedFeeds()
			if err != nil {
				return output.PrintError("load_failed", err.Error(), nil)
			}

			for _, f := range feeds {
				if f.Name == name || f.URL == name {
					return fetchFeed(f.URL, limit, summaryLen)
				}
			}

			return output.PrintError("not_found", "Feed not found: "+name, nil)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of items")
	cmd.Flags().IntVarP(&summaryLen, "summary", "s", 150, "Summary max length (0 = none)")

	return cmd
}

func fetchFeed(url string, limit, summaryLen int) error {
	feed, err := parser.ParseURL(url)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	if limit > len(feed.Items) {
		limit = len(feed.Items)
	}

	items := make([]FeedItem, 0, limit)
	for i := 0; i < limit; i++ {
		item := feed.Items[i]
		items = append(items, toFeedItem(item, summaryLen))
	}

	info := FeedInfo{
		Title: feed.Title,
		URL:   url,
		Desc:  truncate(cleanHTML(feed.Description), 200),
		Items: items,
	}

	if feed.UpdatedParsed != nil {
		info.Updated = timeAgo(*feed.UpdatedParsed)
	}

	return output.Print(info)
}

func toFeedItem(item *gofeed.Item, summaryLen int) FeedItem {
	fi := FeedItem{
		Title: cleanHTML(item.Title),
		URL:   item.Link,
	}

	// Author
	if item.Author != nil {
		fi.Author = item.Author.Name
	}

	// Summary
	if summaryLen > 0 {
		desc := item.Description
		if desc == "" {
			desc = item.Content
		}
		fi.Summary = truncate(cleanHTML(desc), summaryLen)
	}

	// Age
	if item.PublishedParsed != nil {
		fi.Age = timeAgo(*item.PublishedParsed)
	} else if item.UpdatedParsed != nil {
		fi.Age = timeAgo(*item.UpdatedParsed)
	}

	return fi
}

func cleanHTML(s string) string {
	s = strings.ReplaceAll(s, "<p>", " ")
	s = strings.ReplaceAll(s, "<br>", " ")
	s = strings.ReplaceAll(s, "<br/>", " ")
	s = strings.ReplaceAll(s, "<br />", " ")
	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.Join(strings.Fields(s), " ") // normalize whitespace
	return strings.TrimSpace(s)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func timeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < 0:
		return "future"
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh", int(diff.Hours()))
	default:
		return fmt.Sprintf("%dd", int(diff.Hours()/24))
	}
}

func feedsFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pocket", "feeds.json")
}

func loadSavedFeeds() ([]SavedFeed, error) {
	path := feedsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SavedFeed{}, nil
		}
		return nil, err
	}

	var feeds []SavedFeed
	if err := json.Unmarshal(data, &feeds); err != nil {
		return nil, err
	}
	return feeds, nil
}

func saveSavedFeeds(feeds []SavedFeed) error {
	path := feedsFilePath()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(feeds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
