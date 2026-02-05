package hackernews

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

const (
	baseURL    = "https://hacker-news.firebaseio.com/v0"
	maxWorkers = 10
)

var client = &http.Client{Timeout: 10 * time.Second}

// Item represents a HN item (story, comment, etc)
type Item struct {
	ID          int    `json:"id"`
	Type        string `json:"type,omitempty"`
	By          string `json:"by,omitempty"`
	Time        int64  `json:"time,omitempty"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
	Text        string `json:"text,omitempty"`
	Score       int    `json:"score,omitempty"`
	Descendants int    `json:"descendants,omitempty"`
	Kids        []int  `json:"kids,omitempty"`
	Parent      int    `json:"parent,omitempty"`
}

// Story is the LLM-friendly output for a story
type Story struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"`
	Points   int    `json:"points"`
	Author   string `json:"author"`
	Comments int    `json:"comments"`
	Age      string `json:"age"`
}

// Comment is the LLM-friendly output for a comment
type Comment struct {
	ID     int    `json:"id"`
	Author string `json:"author"`
	Text   string `json:"text"`
	Age    string `json:"age"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hackernews",
		Aliases: []string{"hn"},
		Short:   "Hacker News commands",
	}

	cmd.AddCommand(newTopCmd())
	cmd.AddCommand(newNewCmd())
	cmd.AddCommand(newBestCmd())
	cmd.AddCommand(newAskCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newItemCmd())

	return cmd
}

func newTopCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "top",
		Short: "Top stories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchStories("topstories", limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of stories")
	return cmd
}

func newNewCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "new",
		Short: "New stories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchStories("newstories", limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of stories")
	return cmd
}

func newBestCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "best",
		Short: "Best stories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchStories("beststories", limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of stories")
	return cmd
}

func newAskCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask HN stories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchStories("askstories", limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of stories")
	return cmd
}

func newShowCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show HN stories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchStories("showstories", limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of stories")
	return cmd
}

func newItemCmd() *cobra.Command {
	var commentsLimit int

	cmd := &cobra.Command{
		Use:   "item [id]",
		Short: "Get item with comments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var id int
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return output.PrintError("invalid_id", "ID must be a number", nil)
			}
			return fetchItem(id, commentsLimit)
		},
	}

	cmd.Flags().IntVarP(&commentsLimit, "comments", "c", 5, "Number of top comments")
	return cmd
}

func fetchStories(endpoint string, limit int) error {
	ids, err := getStoryIDs(endpoint)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	if limit > len(ids) {
		limit = len(ids)
	}
	ids = ids[:limit]

	stories := fetchItemsConcurrent(ids)

	result := make([]Story, 0, len(stories))
	for _, item := range stories {
		if item != nil {
			result = append(result, toStory(item))
		}
	}

	return output.Print(result)
}

func fetchItem(id int, commentsLimit int) error {
	item, err := getItem(id)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	if item.Type == "story" || item.Type == "job" {
		story := toStory(item)

		// Fetch top comments if any
		var comments []Comment
		if len(item.Kids) > 0 {
			if commentsLimit > len(item.Kids) {
				commentsLimit = len(item.Kids)
			}
			commentItems := fetchItemsConcurrent(item.Kids[:commentsLimit])
			for _, c := range commentItems {
				if c != nil && c.Type == "comment" {
					comments = append(comments, toComment(c))
				}
			}
		}

		return output.Print(map[string]any{
			"story":    story,
			"comments": comments,
		})
	}

	// It's a comment
	return output.Print(toComment(item))
}

func getStoryIDs(endpoint string) ([]int, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s.json", baseURL, endpoint))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func getItem(id int) (*Item, error) {
	resp, err := client.Get(fmt.Sprintf("%s/item/%d.json", baseURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var item Item
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

func fetchItemsConcurrent(ids []int) []*Item {
	items := make([]*Item, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	for i, id := range ids {
		wg.Add(1)
		go func(idx, itemID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			item, err := getItem(itemID)
			if err == nil {
				items[idx] = item
			}
		}(i, id)
	}

	wg.Wait()
	return items
}

func toStory(item *Item) Story {
	return Story{
		ID:       item.ID,
		Title:    item.Title,
		URL:      item.URL,
		Points:   item.Score,
		Author:   item.By,
		Comments: item.Descendants,
		Age:      timeAgo(item.Time),
	}
}

func toComment(item *Item) Comment {
	return Comment{
		ID:     item.ID,
		Author: item.By,
		Text:   cleanHTML(item.Text),
		Age:    timeAgo(item.Time),
	}
}

func cleanHTML(s string) string {
	// Replace <p> with newlines
	s = strings.ReplaceAll(s, "<p>", "\n\n")
	// Remove all other HTML tags
	s = htmlTagRe.ReplaceAllString(s, "")
	// Decode HTML entities
	s = html.UnescapeString(s)
	// Trim whitespace
	s = strings.TrimSpace(s)
	return s
}

func timeAgo(unix int64) string {
	diff := time.Since(time.Unix(unix, 0))

	switch {
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		m := int(diff.Minutes())
		return fmt.Sprintf("%dm", m)
	case diff < 24*time.Hour:
		h := int(diff.Hours())
		return fmt.Sprintf("%dh", h)
	default:
		d := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd", d)
	}
}
