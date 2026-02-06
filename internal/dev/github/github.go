package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://api.github.com"

var httpClient = &http.Client{}

// Repo is LLM-friendly repo output
type Repo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Desc     string `json:"desc,omitempty"`
	Private  bool   `json:"private"`
	Stars    int    `json:"stars"`
	Forks    int    `json:"forks"`
	Language string `json:"lang,omitempty"`
	Updated  string `json:"updated"`
	URL      string `json:"url"`
}

// Issue is LLM-friendly issue output
type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Author string   `json:"author"`
	Labels []string `json:"labels,omitempty"`
	Age    string   `json:"age"`
	URL    string   `json:"url"`
	IsPR   bool     `json:"is_pr,omitempty"`
	Body   string   `json:"body,omitempty"`
}

// PR is LLM-friendly PR output
type PR struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Author    string   `json:"author"`
	Labels    []string `json:"labels,omitempty"`
	Draft     bool     `json:"draft,omitempty"`
	Mergeable string   `json:"mergeable,omitempty"`
	Age       string   `json:"age"`
	URL       string   `json:"url"`
	Body      string   `json:"body,omitempty"`
}

// Notification is LLM-friendly notification output
type Notification struct {
	ID      string `json:"id"`
	Reason  string `json:"reason"`
	Title   string `json:"title"`
	Repo    string `json:"repo"`
	Type    string `json:"type"`
	Unread  bool   `json:"unread"`
	Updated string `json:"updated"`
	URL     string `json:"url"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "github",
		Aliases: []string{"gh"},
		Short:   "GitHub commands",
	}

	cmd.AddCommand(newReposCmd())
	cmd.AddCommand(newRepoCmd())
	cmd.AddCommand(newIssuesCmd())
	cmd.AddCommand(newIssueCmd())
	cmd.AddCommand(newPRsCmd())
	cmd.AddCommand(newPRCmd())
	cmd.AddCommand(newNotificationsCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

func newReposCmd() *cobra.Command {
	var limit int
	var sort string
	var user string

	cmd := &cobra.Command{
		Use:   "repos",
		Short: "List repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/user/repos?per_page=%d&sort=%s", baseURL, limit, sort)
			if user != "" {
				url = fmt.Sprintf("%s/users/%s/repos?per_page=%d&sort=%s", baseURL, user, limit, sort)
			}

			var repos []map[string]any
			if err := ghGet(token, url, &repos); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			result := make([]Repo, 0, len(repos))
			for _, r := range repos {
				result = append(result, toRepo(r))
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of repos")
	cmd.Flags().StringVarP(&sort, "sort", "s", "updated", "Sort: updated, created, pushed, full_name")
	cmd.Flags().StringVarP(&user, "user", "u", "", "User (default: authenticated user)")

	return cmd
}

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo [owner/name]",
		Short: "Get repository details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/repos/%s", baseURL, args[0])

			var repo map[string]any
			if err := ghGet(token, url, &repo); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			return output.Print(toRepo(repo))
		},
	}

	return cmd
}

func newIssuesCmd() *cobra.Command {
	var repo string
	var state string
	var limit int
	var labels string

	cmd := &cobra.Command{
		Use:   "issues",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			var url string
			if repo != "" {
				url = fmt.Sprintf("%s/repos/%s/issues?state=%s&per_page=%d", baseURL, repo, state, limit)
			} else {
				url = fmt.Sprintf("%s/issues?state=%s&per_page=%d&filter=all", baseURL, state, limit)
			}

			if labels != "" {
				url += "&labels=" + labels
			}

			var issues []map[string]any
			if err := ghGet(token, url, &issues); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			result := make([]Issue, 0, len(issues))
			for _, i := range issues {
				// Skip PRs when listing issues (GitHub API returns both)
				if _, ok := i["pull_request"]; !ok {
					result = append(result, toIssue(i, false))
				}
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository (owner/name)")
	cmd.Flags().StringVarP(&state, "state", "s", "open", "State: open, closed, all")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of issues")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")

	return cmd
}

func newIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue [owner/repo] [number]",
		Short: "Get issue details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/repos/%s/issues/%s", baseURL, args[0], args[1])

			var issue map[string]any
			if err := ghGet(token, url, &issue); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			return output.Print(toIssue(issue, true))
		},
	}

	return cmd
}

func newPRsCmd() *cobra.Command {
	var repo string
	var state string
	var limit int

	cmd := &cobra.Command{
		Use:     "prs",
		Aliases: []string{"pulls"},
		Short:   "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			if repo == "" {
				return output.PrintError("missing_repo", "Repository required for PRs (use -r owner/repo)", nil)
			}

			url := fmt.Sprintf("%s/repos/%s/pulls?state=%s&per_page=%d", baseURL, repo, state, limit)

			var prs []map[string]any
			if err := ghGet(token, url, &prs); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			result := make([]PR, 0, len(prs))
			for _, p := range prs {
				result = append(result, toPR(p, false))
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository (owner/name) - required")
	cmd.Flags().StringVarP(&state, "state", "s", "open", "State: open, closed, all")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of PRs")

	return cmd
}

func newPRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr [owner/repo] [number]",
		Short: "Get PR details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/repos/%s/pulls/%s", baseURL, args[0], args[1])

			var pr map[string]any
			if err := ghGet(token, url, &pr); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			return output.Print(toPR(pr, true))
		},
	}

	return cmd
}

func newNotificationsCmd() *cobra.Command {
	var limit int
	var all bool

	cmd := &cobra.Command{
		Use:     "notifications",
		Aliases: []string{"notifs"},
		Short:   "List notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/notifications?per_page=%d", baseURL, limit)
			if all {
				url += "&all=true"
			}

			var notifs []map[string]any
			if err := ghGet(token, url, &notifs); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			result := make([]Notification, 0, len(notifs))
			for _, n := range notifs {
				result = append(result, toNotification(n))
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 30, "Number of notifications")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include read notifications")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var searchType string
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search GitHub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.MustGet("github_token")
			if err != nil {
				return err
			}

			query := strings.ReplaceAll(args[0], " ", "+")
			url := fmt.Sprintf("%s/search/%s?q=%s&per_page=%d", baseURL, searchType, query, limit)

			var result map[string]any
			if err := ghGet(token, url, &result); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			items, _ := result["items"].([]any)
			count := int(result["total_count"].(float64))

			switch searchType {
			case "repositories":
				repos := make([]Repo, 0, len(items))
				for _, item := range items {
					if r, ok := item.(map[string]any); ok {
						repos = append(repos, toRepo(r))
					}
				}
				return output.Print(map[string]any{"total": count, "items": repos})

			case "issues":
				issues := make([]Issue, 0, len(items))
				for _, item := range items {
					if i, ok := item.(map[string]any); ok {
						issues = append(issues, toIssue(i, false))
					}
				}
				return output.Print(map[string]any{"total": count, "items": issues})

			default:
				return output.Print(map[string]any{"total": count, "items": items})
			}
		},
	}

	cmd.Flags().StringVarP(&searchType, "type", "t", "repositories", "Type: repositories, issues, code, users")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")

	return cmd
}

func ghGet(token, url string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			if msg, _ := errResp["message"].(string); msg != "" {
				return fmt.Errorf("%s", msg)
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func toRepo(r map[string]any) Repo {
	repo := Repo{
		Name:     getString(r, "name"),
		FullName: getString(r, "full_name"),
		Private:  getBool(r, "private"),
		Stars:    getInt(r, "stargazers_count"),
		Forks:    getInt(r, "forks_count"),
		Language: getString(r, "language"),
		URL:      getString(r, "html_url"),
	}

	if desc := getString(r, "description"); desc != "" {
		repo.Desc = truncate(desc, 120)
	}

	if updated := getString(r, "updated_at"); updated != "" {
		repo.Updated = parseTimeAgo(updated)
	}

	return repo
}

func toIssue(i map[string]any, includeBody bool) Issue {
	issue := Issue{
		Number: getInt(i, "number"),
		Title:  getString(i, "title"),
		State:  getString(i, "state"),
		URL:    getString(i, "html_url"),
	}

	if user, ok := i["user"].(map[string]any); ok {
		issue.Author = getString(user, "login")
	}

	if labels, ok := i["labels"].([]any); ok && len(labels) > 0 {
		for _, l := range labels {
			if label, ok := l.(map[string]any); ok {
				issue.Labels = append(issue.Labels, getString(label, "name"))
			}
		}
	}

	if created := getString(i, "created_at"); created != "" {
		issue.Age = parseTimeAgo(created)
	}

	if _, ok := i["pull_request"]; ok {
		issue.IsPR = true
	}

	if includeBody {
		if body := getString(i, "body"); body != "" {
			issue.Body = truncate(body, 500)
		}
	}

	return issue
}

func toPR(p map[string]any, includeBody bool) PR {
	pr := PR{
		Number: getInt(p, "number"),
		Title:  getString(p, "title"),
		State:  getString(p, "state"),
		Draft:  getBool(p, "draft"),
		URL:    getString(p, "html_url"),
	}

	if user, ok := p["user"].(map[string]any); ok {
		pr.Author = getString(user, "login")
	}

	if labels, ok := p["labels"].([]any); ok && len(labels) > 0 {
		for _, l := range labels {
			if label, ok := l.(map[string]any); ok {
				pr.Labels = append(pr.Labels, getString(label, "name"))
			}
		}
	}

	if created := getString(p, "created_at"); created != "" {
		pr.Age = parseTimeAgo(created)
	}

	if mergeable, ok := p["mergeable"].(bool); ok {
		if mergeable {
			pr.Mergeable = "yes"
		} else {
			pr.Mergeable = "no"
		}
	}

	if includeBody {
		if body := getString(p, "body"); body != "" {
			pr.Body = truncate(body, 500)
		}
	}

	return pr
}

func toNotification(n map[string]any) Notification {
	notif := Notification{
		ID:     getString(n, "id"),
		Reason: getString(n, "reason"),
		Unread: getBool(n, "unread"),
	}

	if subject, ok := n["subject"].(map[string]any); ok {
		notif.Title = getString(subject, "title")
		notif.Type = getString(subject, "type")
		notif.URL = getString(subject, "url")
		// Convert API URL to web URL
		notif.URL = strings.Replace(notif.URL, "api.github.com/repos", "github.com", 1)
		notif.URL = strings.Replace(notif.URL, "/pulls/", "/pull/", 1)
	}

	if repo, ok := n["repository"].(map[string]any); ok {
		notif.Repo = getString(repo, "full_name")
	}

	if updated := getString(n, "updated_at"); updated != "" {
		notif.Updated = parseTimeAgo(updated)
	}

	return notif
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func parseTimeAgo(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return timeAgo(t)
}

func timeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
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

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
