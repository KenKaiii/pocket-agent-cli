package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gitlab",
		Aliases: []string{"gl"},
		Short:   "GitLab commands",
		Long:    "GitLab API integration for projects, issues, and merge requests.",
	}

	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newIssuesCmd())
	cmd.AddCommand(newMRsCmd())
	cmd.AddCommand(newUserCmd())

	return cmd
}

type gitlabClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func newGitLabClient() (*gitlabClient, error) {
	token, err := config.MustGet("gitlab_token")
	if err != nil {
		return nil, err
	}

	baseURL, _ := config.Get("gitlab_url")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	return &gitlabClient{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}, nil
}

func (c *gitlabClient) doRequest(endpoint string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v4"+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if errResp.Message != "" {
				return nil, fmt.Errorf("GitLab API error: %s", errResp.Message)
			}
			if errResp.Error != "" {
				return nil, fmt.Errorf("GitLab API error: %s", errResp.Error)
			}
		}
		return nil, fmt.Errorf("GitLab API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func newProjectsCmd() *cobra.Command {
	var limit int
	var owned bool
	var membership bool

	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGitLabClient()
			if err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/projects?per_page=%d&order_by=updated_at", limit)
			if owned {
				endpoint += "&owned=true"
			}
			if membership {
				endpoint += "&membership=true"
			}

			body, err := client.doRequest(endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var projects []struct {
				ID                int    `json:"id"`
				Name              string `json:"name"`
				NameWithNamespace string `json:"name_with_namespace"`
				Path              string `json:"path_with_namespace"`
				Description       string `json:"description"`
				WebURL            string `json:"web_url"`
				DefaultBranch     string `json:"default_branch"`
				Visibility        string `json:"visibility"`
				StarCount         int    `json:"star_count"`
				ForksCount        int    `json:"forks_count"`
				LastActivityAt    string `json:"last_activity_at"`
			}

			if err := json.Unmarshal(body, &projects); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			result := make([]map[string]any, len(projects))
			for i, p := range projects {
				result[i] = map[string]any{
					"id":             p.ID,
					"name":           p.Name,
					"full_name":      p.NameWithNamespace,
					"path":           p.Path,
					"description":    p.Description,
					"url":            p.WebURL,
					"default_branch": p.DefaultBranch,
					"visibility":     p.Visibility,
					"stars":          p.StarCount,
					"forks":          p.ForksCount,
					"last_activity":  p.LastActivityAt,
				}
			}

			return output.Print(map[string]any{
				"count":    len(projects),
				"projects": result,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 30, "Number of projects")
	cmd.Flags().BoolVarP(&owned, "owned", "o", false, "Only show owned projects")
	cmd.Flags().BoolVarP(&membership, "member", "m", true, "Only show projects where you're a member")

	return cmd
}

func newIssuesCmd() *cobra.Command {
	var project string
	var state string
	var limit int
	var assignee string

	cmd := &cobra.Command{
		Use:   "issues",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGitLabClient()
			if err != nil {
				return err
			}

			var endpoint string
			if project != "" {
				encodedProject := url.PathEscape(project)
				endpoint = fmt.Sprintf("/projects/%s/issues?per_page=%d&state=%s", encodedProject, limit, state)
			} else {
				endpoint = fmt.Sprintf("/issues?per_page=%d&state=%s&scope=all", limit, state)
			}

			if assignee == "me" {
				endpoint += "&assignee_id=@me"
			}

			body, err := client.doRequest(endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var issues []struct {
				ID          int      `json:"id"`
				IID         int      `json:"iid"`
				Title       string   `json:"title"`
				Description string   `json:"description"`
				State       string   `json:"state"`
				WebURL      string   `json:"web_url"`
				Labels      []string `json:"labels"`
				Author      struct {
					Username string `json:"username"`
				} `json:"author"`
				Assignees []struct {
					Username string `json:"username"`
				} `json:"assignees"`
				CreatedAt string `json:"created_at"`
				UpdatedAt string `json:"updated_at"`
			}

			if err := json.Unmarshal(body, &issues); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			result := make([]map[string]any, len(issues))
			for i, issue := range issues {
				assignees := make([]string, len(issue.Assignees))
				for j, a := range issue.Assignees {
					assignees[j] = a.Username
				}

				result[i] = map[string]any{
					"id":         issue.ID,
					"iid":        issue.IID,
					"title":      issue.Title,
					"state":      issue.State,
					"url":        issue.WebURL,
					"labels":     issue.Labels,
					"author":     issue.Author.Username,
					"assignees":  assignees,
					"created_at": issue.CreatedAt,
					"updated_at": issue.UpdatedAt,
				}
			}

			return output.Print(map[string]any{
				"count":  len(issues),
				"issues": result,
			})
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project ID or path (e.g., 'group/project')")
	cmd.Flags().StringVarP(&state, "state", "s", "opened", "State: opened, closed, all")
	cmd.Flags().IntVarP(&limit, "limit", "l", 30, "Number of issues")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Filter by assignee ('me' for yourself)")

	return cmd
}

func newMRsCmd() *cobra.Command {
	var project string
	var state string
	var limit int

	cmd := &cobra.Command{
		Use:     "mrs",
		Aliases: []string{"merge-requests", "prs"},
		Short:   "List merge requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGitLabClient()
			if err != nil {
				return err
			}

			var endpoint string
			if project != "" {
				encodedProject := url.PathEscape(project)
				endpoint = fmt.Sprintf("/projects/%s/merge_requests?per_page=%d&state=%s", encodedProject, limit, state)
			} else {
				endpoint = fmt.Sprintf("/merge_requests?per_page=%d&state=%s&scope=all", limit, state)
			}

			body, err := client.doRequest(endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var mrs []struct {
				ID           int      `json:"id"`
				IID          int      `json:"iid"`
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				State        string   `json:"state"`
				WebURL       string   `json:"web_url"`
				SourceBranch string   `json:"source_branch"`
				TargetBranch string   `json:"target_branch"`
				Labels       []string `json:"labels"`
				Author       struct {
					Username string `json:"username"`
				} `json:"author"`
				MergeStatus string `json:"merge_status"`
				Draft       bool   `json:"draft"`
				CreatedAt   string `json:"created_at"`
				UpdatedAt   string `json:"updated_at"`
			}

			if err := json.Unmarshal(body, &mrs); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			result := make([]map[string]any, len(mrs))
			for i, mr := range mrs {
				result[i] = map[string]any{
					"id":            mr.ID,
					"iid":           mr.IID,
					"title":         mr.Title,
					"state":         mr.State,
					"url":           mr.WebURL,
					"source_branch": mr.SourceBranch,
					"target_branch": mr.TargetBranch,
					"labels":        mr.Labels,
					"author":        mr.Author.Username,
					"merge_status":  mr.MergeStatus,
					"draft":         mr.Draft,
					"created_at":    mr.CreatedAt,
					"updated_at":    mr.UpdatedAt,
				}
			}

			return output.Print(map[string]any{
				"count":          len(mrs),
				"merge_requests": result,
			})
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "Project ID or path")
	cmd.Flags().StringVarP(&state, "state", "s", "opened", "State: opened, closed, merged, all")
	cmd.Flags().IntVarP(&limit, "limit", "l", 30, "Number of MRs")

	return cmd
}

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Get current user info",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGitLabClient()
			if err != nil {
				return err
			}

			body, err := client.doRequest("/user")
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var user struct {
				ID        int    `json:"id"`
				Username  string `json:"username"`
				Name      string `json:"name"`
				Email     string `json:"email"`
				AvatarURL string `json:"avatar_url"`
				WebURL    string `json:"web_url"`
				State     string `json:"state"`
				IsAdmin   bool   `json:"is_admin"`
				CreatedAt string `json:"created_at"`
			}

			if err := json.Unmarshal(body, &user); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"id":         user.ID,
				"username":   user.Username,
				"name":       user.Name,
				"email":      user.Email,
				"avatar":     user.AvatarURL,
				"url":        user.WebURL,
				"state":      user.State,
				"is_admin":   user.IsAdmin,
				"created_at": user.CreatedAt,
			})
		},
	}

	return cmd
}
