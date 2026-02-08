package sentry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://sentry.io/api/0"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Project is LLM-friendly Sentry project output
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Platform    string `json:"platform,omitempty"`
	DateCreated string `json:"date_created"`
	Status      string `json:"status"`
}

// SentryIssue is LLM-friendly Sentry issue output
type SentryIssue struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Culprit   string `json:"culprit,omitempty"`
	Level     string `json:"level"`
	Status    string `json:"status"`
	Count     string `json:"count"`
	UserCount int    `json:"user_count"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
	Permalink string `json:"permalink"`
}

// IssueDetail is LLM-friendly Sentry issue detail output
type IssueDetail struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Culprit   string            `json:"culprit,omitempty"`
	Level     string            `json:"level"`
	Status    string            `json:"status"`
	Count     string            `json:"count"`
	UserCount int               `json:"user_count"`
	FirstSeen string            `json:"first_seen"`
	LastSeen  string            `json:"last_seen"`
	Permalink string            `json:"permalink"`
	Logger    string            `json:"logger,omitempty"`
	Type      string            `json:"type,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Event is LLM-friendly Sentry event output
type Event struct {
	ID       string     `json:"id"`
	EventID  string     `json:"event_id"`
	Title    string     `json:"title"`
	Message  string     `json:"message,omitempty"`
	Platform string     `json:"platform,omitempty"`
	DateTime string     `json:"date_time"`
	Tags     []EventTag `json:"tags,omitempty"`
}

// EventTag is a key-value tag on a Sentry event
type EventTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewCmd returns the sentry parent command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sentry",
		Aliases: []string{"sr"},
		Short:   "Sentry error tracking commands",
	}

	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newIssuesCmd())
	cmd.AddCommand(newIssueCmd())
	cmd.AddCommand(newEventsCmd())

	return cmd
}

func getToken() (string, error) {
	token, err := config.Get("sentry_auth_token")
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", output.PrintError("missing_config", "Sentry auth token not configured", map[string]string{
			"setup": "Run: pocket config set sentry_auth_token YOUR_TOKEN",
		})
	}
	return token, nil
}

func getOrg(flagOrg string) string {
	if flagOrg != "" {
		return flagOrg
	}
	org, err := config.Get("sentry_org")
	if err == nil && org != "" {
		return org
	}
	return ""
}

func newProjectsCmd() *cobra.Command {
	var limit int
	var org string

	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List Sentry projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			apiURL := fmt.Sprintf("%s/projects/", baseURL)

			var raw []map[string]any
			if err := sentryGet(token, apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			orgSlug := getOrg(org)

			result := make([]Project, 0, limit)
			for _, p := range raw {
				if len(result) >= limit {
					break
				}

				// Filter by org if specified
				if orgSlug != "" {
					if orgMap, ok := p["organization"].(map[string]any); ok {
						if getString(orgMap, "slug") != orgSlug {
							continue
						}
					}
				}

				result = append(result, Project{
					ID:          getString(p, "id"),
					Name:        getString(p, "name"),
					Slug:        getString(p, "slug"),
					Platform:    getString(p, "platform"),
					DateCreated: getString(p, "dateCreated"),
					Status:      getString(p, "status"),
				})
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of projects to show")
	cmd.Flags().StringVarP(&org, "org", "o", "", "Filter by organization slug")

	return cmd
}

func newIssuesCmd() *cobra.Command {
	var limit int
	var query string
	var org string

	cmd := &cobra.Command{
		Use:   "issues [project-slug]",
		Short: "List issues for a Sentry project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			orgSlug := getOrg(org)
			if orgSlug == "" {
				return output.PrintError("missing_org", "Organization slug required (use --org or set sentry_org in config)", map[string]string{
					"setup": "Run: pocket config set sentry_org YOUR_ORG_SLUG",
				})
			}

			projectSlug := args[0]
			apiURL := fmt.Sprintf("%s/projects/%s/%s/issues/", baseURL, url.PathEscape(orgSlug), url.PathEscape(projectSlug))

			if query != "" {
				apiURL += "?query=" + url.QueryEscape(query)
			}

			var raw []map[string]any
			if err := sentryGet(token, apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			result := make([]SentryIssue, 0, limit)
			for _, i := range raw {
				if len(result) >= limit {
					break
				}
				result = append(result, toSentryIssue(i))
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of issues to show")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query")
	cmd.Flags().StringVarP(&org, "org", "o", "", "Organization slug (falls back to config)")

	return cmd
}

func newIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue [issue-id]",
		Short: "Get Sentry issue details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			issueID := args[0]
			apiURL := fmt.Sprintf("%s/issues/%s/", baseURL, url.PathEscape(issueID))

			var raw map[string]any
			if err := sentryGet(token, apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			detail := IssueDetail{
				ID:        getString(raw, "id"),
				Title:     getString(raw, "title"),
				Culprit:   getString(raw, "culprit"),
				Level:     getString(raw, "level"),
				Status:    getString(raw, "status"),
				Count:     getString(raw, "count"),
				UserCount: getInt(raw, "userCount"),
				FirstSeen: getString(raw, "firstSeen"),
				LastSeen:  getString(raw, "lastSeen"),
				Permalink: getString(raw, "permalink"),
				Logger:    getString(raw, "logger"),
				Type:      getString(raw, "type"),
			}

			if meta, ok := raw["metadata"].(map[string]any); ok {
				detail.Metadata = make(map[string]string)
				for k, v := range meta {
					if s, ok := v.(string); ok {
						detail.Metadata[k] = s
					}
				}
			}

			return output.Print(detail)
		},
	}

	return cmd
}

func newEventsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "events [issue-id]",
		Short: "List events for a Sentry issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			issueID := args[0]
			apiURL := fmt.Sprintf("%s/issues/%s/events/", baseURL, url.PathEscape(issueID))

			var raw []map[string]any
			if err := sentryGet(token, apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			result := make([]Event, 0, limit)
			for _, e := range raw {
				if len(result) >= limit {
					break
				}

				event := Event{
					ID:       getString(e, "id"),
					EventID:  getString(e, "eventID"),
					Title:    getString(e, "title"),
					Message:  getString(e, "message"),
					Platform: getString(e, "platform"),
					DateTime: getString(e, "dateTime"),
				}

				if tags, ok := e["tags"].([]any); ok {
					for _, t := range tags {
						if tag, ok := t.(map[string]any); ok {
							event.Tags = append(event.Tags, EventTag{
								Key:   getString(tag, "key"),
								Value: getString(tag, "value"),
							})
						}
					}
				}

				result = append(result, event)
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 5, "Number of events to show")

	return cmd
}

func sentryGet(token, apiURL string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if detail := getString(errResp, "detail"); detail != "" {
				return fmt.Errorf("%s", detail)
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func toSentryIssue(i map[string]any) SentryIssue {
	return SentryIssue{
		ID:        getString(i, "id"),
		Title:     getString(i, "title"),
		Culprit:   getString(i, "culprit"),
		Level:     getString(i, "level"),
		Status:    getString(i, "status"),
		Count:     getString(i, "count"),
		UserCount: getInt(i, "userCount"),
		FirstSeen: getString(i, "firstSeen"),
		LastSeen:  getString(i, "lastSeen"),
		Permalink: getString(i, "permalink"),
	}
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
