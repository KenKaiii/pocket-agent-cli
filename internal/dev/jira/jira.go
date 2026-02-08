package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Issue is LLM-friendly issue output
type Issue struct {
	Key         string   `json:"key"`
	Summary     string   `json:"summary"`
	Status      string   `json:"status"`
	Type        string   `json:"type"`
	Priority    string   `json:"priority,omitempty"`
	Assignee    string   `json:"assignee,omitempty"`
	Reporter    string   `json:"reporter,omitempty"`
	Project     string   `json:"project"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	URL         string   `json:"url"`
}

// Project is LLM-friendly project output
type Project struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Lead   string `json:"lead,omitempty"`
	URL    string `json:"url"`
	Avatar string `json:"avatar,omitempty"`
}

// Transition is LLM-friendly transition output
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   string `json:"to"`
}

// CreateResult is the result of creating an issue
type CreateResult struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// TransitionResult is the result of transitioning an issue
type TransitionResult struct {
	Key        string `json:"key"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	URL        string `json:"url"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "jira",
		Aliases: []string{"j"},
		Short:   "Jira commands",
	}

	cmd.AddCommand(newIssuesCmd())
	cmd.AddCommand(newIssueCmd())
	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newTransitionCmd())

	return cmd
}

func newIssuesCmd() *cobra.Command {
	var project string
	var status string
	var limit int

	cmd := &cobra.Command{
		Use:   "issues",
		Short: "List issues assigned to me or in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, email, token, err := getCredentials()
			if err != nil {
				return err
			}

			// Build JQL query
			jql := "assignee = currentUser()"
			if project != "" {
				jql = fmt.Sprintf("project = %s", project)
			}
			if status != "" {
				jql += fmt.Sprintf(" AND status = \"%s\"", status)
			}
			jql += " ORDER BY updated DESC"

			apiURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=%d", baseURL, url.QueryEscape(jql), limit)

			var result map[string]any
			if err := jiraGet(baseURL, email, token, apiURL, &result); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			issues, _ := result["issues"].([]any)
			output := make([]Issue, 0, len(issues))
			for _, i := range issues {
				if issue, ok := i.(map[string]any); ok {
					output = append(output, toIssue(baseURL, issue, false))
				}
			}

			return outputPrint(output)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Filter by project key (e.g., PROJ)")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (e.g., \"In Progress\")")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of issues")

	return cmd
}

func newIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue [key]",
		Short: "Get issue details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, email, token, err := getCredentials()
			if err != nil {
				return err
			}

			apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", baseURL, args[0])

			var issue map[string]any
			if err := jiraGet(baseURL, email, token, apiURL, &issue); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			return outputPrint(toIssue(baseURL, issue, true))
		},
	}

	return cmd
}

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List accessible projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, email, token, err := getCredentials()
			if err != nil {
				return err
			}

			apiURL := fmt.Sprintf("%s/rest/api/3/project", baseURL)

			var projects []map[string]any
			if err := jiraGet(baseURL, email, token, apiURL, &projects); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			result := make([]Project, 0, len(projects))
			for _, p := range projects {
				result = append(result, toProject(baseURL, p))
			}

			return outputPrint(result)
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var project string
	var issueType string
	var description string

	cmd := &cobra.Command{
		Use:   "create [summary]",
		Short: "Create a new issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, email, token, err := getCredentials()
			if err != nil {
				return err
			}

			if project == "" {
				return output.PrintError("missing_project", "Project key required (use -p PROJECT)", nil)
			}

			if issueType == "" {
				issueType = "Task"
			}

			// Build request body
			body := map[string]any{
				"fields": map[string]any{
					"project": map[string]any{
						"key": project,
					},
					"summary": args[0],
					"issuetype": map[string]any{
						"name": issueType,
					},
				},
			}

			if description != "" {
				// Jira API v3 uses Atlassian Document Format for description
				body["fields"].(map[string]any)["description"] = map[string]any{
					"type":    "doc",
					"version": 1,
					"content": []map[string]any{
						{
							"type": "paragraph",
							"content": []map[string]any{
								{
									"type": "text",
									"text": description,
								},
							},
						},
					},
				}
			}

			apiURL := fmt.Sprintf("%s/rest/api/3/issue", baseURL)

			var result map[string]any
			if err := jiraPost(baseURL, email, token, apiURL, body, &result); err != nil {
				return output.PrintError("create_failed", err.Error(), nil)
			}

			key := getString(result, "key")
			return outputPrint(CreateResult{
				Key: key,
				URL: fmt.Sprintf("%s/browse/%s", baseURL, key),
			})
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project key (required)")
	cmd.Flags().StringVarP(&issueType, "type", "t", "Task", "Issue type (Task, Bug, Story, etc.)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Issue description")

	return cmd
}

func newTransitionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transition [key] [status]",
		Short: "Change issue status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, email, token, err := getCredentials()
			if err != nil {
				return err
			}

			issueKey := args[0]
			targetStatus := args[1]

			// First, get current issue to capture from_status
			issueURL := fmt.Sprintf("%s/rest/api/3/issue/%s", baseURL, issueKey)
			var issue map[string]any
			if err := jiraGet(baseURL, email, token, issueURL, &issue); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			fromStatus := ""
			if fields, ok := issue["fields"].(map[string]any); ok {
				if status, ok := fields["status"].(map[string]any); ok {
					fromStatus = getString(status, "name")
				}
			}

			// Get available transitions
			transitionsURL := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", baseURL, issueKey)

			var transResult map[string]any
			if err := jiraGet(baseURL, email, token, transitionsURL, &transResult); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			transitions, _ := transResult["transitions"].([]any)

			// Find matching transition
			var transitionID string
			var toStatus string
			for _, t := range transitions {
				if trans, ok := t.(map[string]any); ok {
					name := getString(trans, "name")
					if to, ok := trans["to"].(map[string]any); ok {
						toName := getString(to, "name")
						if strings.EqualFold(name, targetStatus) || strings.EqualFold(toName, targetStatus) {
							transitionID = getString(trans, "id")
							toStatus = toName
							break
						}
					}
				}
			}

			if transitionID == "" {
				// List available transitions in error
				available := make([]string, 0, len(transitions))
				for _, t := range transitions {
					if trans, ok := t.(map[string]any); ok {
						if to, ok := trans["to"].(map[string]any); ok {
							available = append(available, getString(to, "name"))
						}
					}
				}
				return output.PrintError("invalid_transition", fmt.Sprintf("Cannot transition to '%s'. Available: %s", targetStatus, strings.Join(available, ", ")), nil)
			}

			// Perform transition
			body := map[string]any{
				"transition": map[string]any{
					"id": transitionID,
				},
			}

			if err := jiraPost(baseURL, email, token, transitionsURL, body, nil); err != nil {
				return output.PrintError("transition_failed", err.Error(), nil)
			}

			return outputPrint(TransitionResult{
				Key:        issueKey,
				FromStatus: fromStatus,
				ToStatus:   toStatus,
				URL:        fmt.Sprintf("%s/browse/%s", baseURL, issueKey),
			})
		},
	}

	return cmd
}

func getCredentials() (baseURL, email, token string, err error) {
	baseURL, err = config.Get("jira_url")
	if err != nil {
		return "", "", "", err
	}
	if baseURL == "" {
		return "", "", "", output.PrintError("missing_config", "Jira URL not configured", map[string]string{
			"setup": "Run: pocket config set jira_url https://your-domain.atlassian.net",
		})
	}
	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	email, err = config.Get("jira_email")
	if err != nil {
		return "", "", "", err
	}
	if email == "" {
		return "", "", "", output.PrintError("missing_config", "Jira email not configured", map[string]string{
			"setup": "Run: pocket config set jira_email your-email@example.com",
		})
	}

	token, err = config.Get("jira_token")
	if err != nil {
		return "", "", "", err
	}
	if token == "" {
		return "", "", "", output.PrintError("missing_config", "Jira API token not configured", map[string]string{
			"setup": "Run: pocket config set jira_token YOUR_API_TOKEN",
			"docs":  "Create token at: https://id.atlassian.com/manage-profile/security/api-tokens",
		})
	}

	return baseURL, email, token, nil
}

func jiraGet(baseURL, email, token, apiURL string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}

	setAuthHeaders(req, email, token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func jiraPost(baseURL, email, token, apiURL string, body any, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	setAuthHeaders(req, email, token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}

	// Some endpoints return no content (204)
	if result != nil && resp.StatusCode != 204 {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func setAuthHeaders(req *http.Request, email, token string) {
	// Basic Auth: base64(email:token)
	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
}

func parseError(resp *http.Response) error {
	var errResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
		// Try to extract error messages
		if messages, ok := errResp["errorMessages"].([]any); ok && len(messages) > 0 {
			if msg, ok := messages[0].(string); ok {
				return fmt.Errorf("%s", msg)
			}
		}
		if errors, ok := errResp["errors"].(map[string]any); ok {
			for _, v := range errors {
				if msg, ok := v.(string); ok {
					return fmt.Errorf("%s", msg)
				}
			}
		}
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
}

func toIssue(baseURL string, i map[string]any, includeDesc bool) Issue {
	issue := Issue{
		Key: getString(i, "key"),
		URL: fmt.Sprintf("%s/browse/%s", baseURL, getString(i, "key")),
	}

	if fields, ok := i["fields"].(map[string]any); ok {
		issue.Summary = getString(fields, "summary")

		if status, ok := fields["status"].(map[string]any); ok {
			issue.Status = getString(status, "name")
		}

		if issueType, ok := fields["issuetype"].(map[string]any); ok {
			issue.Type = getString(issueType, "name")
		}

		if priority, ok := fields["priority"].(map[string]any); ok {
			issue.Priority = getString(priority, "name")
		}

		if assignee, ok := fields["assignee"].(map[string]any); ok {
			issue.Assignee = getString(assignee, "displayName")
		}

		if reporter, ok := fields["reporter"].(map[string]any); ok {
			issue.Reporter = getString(reporter, "displayName")
		}

		if project, ok := fields["project"].(map[string]any); ok {
			issue.Project = getString(project, "key")
		}

		if labels, ok := fields["labels"].([]any); ok && len(labels) > 0 {
			for _, l := range labels {
				if label, ok := l.(string); ok {
					issue.Labels = append(issue.Labels, label)
				}
			}
		}

		if created := getString(fields, "created"); created != "" {
			issue.Created = parseTimeAgo(created)
		}

		if updated := getString(fields, "updated"); updated != "" {
			issue.Updated = parseTimeAgo(updated)
		}

		if includeDesc {
			// Description in API v3 is Atlassian Document Format
			if desc, ok := fields["description"].(map[string]any); ok {
				issue.Description = truncate(extractTextFromADF(desc), 500)
			}
		}
	}

	return issue
}

func toProject(baseURL string, p map[string]any) Project {
	proj := Project{
		Key:  getString(p, "key"),
		Name: getString(p, "name"),
		URL:  fmt.Sprintf("%s/browse/%s", baseURL, getString(p, "key")),
	}

	if projectTypeKey := getString(p, "projectTypeKey"); projectTypeKey != "" {
		proj.Type = projectTypeKey
	}

	if lead, ok := p["lead"].(map[string]any); ok {
		proj.Lead = getString(lead, "displayName")
	}

	if avatarUrls, ok := p["avatarUrls"].(map[string]any); ok {
		if avatar, ok := avatarUrls["48x48"].(string); ok {
			proj.Avatar = avatar
		}
	}

	return proj
}

// extractTextFromADF extracts plain text from Atlassian Document Format
func extractTextFromADF(doc map[string]any) string {
	var text strings.Builder
	extractText(doc, &text)
	return strings.TrimSpace(text.String())
}

func extractText(node map[string]any, text *strings.Builder) {
	if t := getString(node, "text"); t != "" {
		text.WriteString(t)
	}

	if content, ok := node["content"].([]any); ok {
		for _, c := range content {
			if child, ok := c.(map[string]any); ok {
				extractText(child, text)
			}
		}
		// Add newline after block elements
		if nodeType := getString(node, "type"); nodeType == "paragraph" || nodeType == "heading" {
			text.WriteString("\n")
		}
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func parseTimeAgo(ts string) string {
	// Jira uses ISO 8601 format: 2024-01-15T10:30:00.000+0000
	layouts := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z",
		time.RFC3339,
	}

	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.Parse(layout, ts)
		if err == nil {
			break
		}
	}
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

// outputPrint is a wrapper to avoid shadowing the output package
func outputPrint(data any) error {
	return output.Print(data)
}
