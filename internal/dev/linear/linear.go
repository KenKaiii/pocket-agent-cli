package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const graphqlURL = "https://api.linear.app/graphql"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "linear",
		Short: "Linear commands",
		Long:  "Linear issue tracking integration.",
	}

	cmd.AddCommand(newIssuesCmd())
	cmd.AddCommand(newTeamsCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newMeCmd())

	return cmd
}

type linearClient struct {
	token      string
	httpClient *http.Client
}

func newLinearClient() (*linearClient, error) {
	token, err := config.MustGet("linear_token")
	if err != nil {
		return nil, err
	}

	return &linearClient{
		token:      token,
		httpClient: &http.Client{},
	}, nil
}

func (c *linearClient) doQuery(query string, variables map[string]any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload := map[string]any{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("Linear API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Check for GraphQL errors
	var result struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &result) == nil && len(result.Errors) > 0 {
		return nil, fmt.Errorf("Linear API error: %s", result.Errors[0].Message)
	}

	return body, nil
}

func newIssuesCmd() *cobra.Command {
	var team string
	var state string
	var limit int
	var assignedToMe bool

	cmd := &cobra.Command{
		Use:   "issues",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newLinearClient()
			if err != nil {
				return err
			}

			query := `
				query Issues($first: Int, $filter: IssueFilter) {
					issues(first: $first, filter: $filter) {
						nodes {
							id
							identifier
							title
							description
							priority
							state { name }
							team { key name }
							assignee { name email }
							createdAt
							updatedAt
							url
						}
					}
				}
			`

			filter := make(map[string]any)
			if team != "" {
				filter["team"] = map[string]any{"key": map[string]string{"eq": team}}
			}
			if state != "" {
				filter["state"] = map[string]any{"name": map[string]string{"eq": state}}
			}
			if assignedToMe {
				filter["assignee"] = map[string]any{"isMe": map[string]bool{"eq": true}}
			}

			variables := map[string]any{
				"first": limit,
			}
			if len(filter) > 0 {
				variables["filter"] = filter
			}

			body, err := client.doQuery(query, variables)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Data struct {
					Issues struct {
						Nodes []struct {
							ID          string `json:"id"`
							Identifier  string `json:"identifier"`
							Title       string `json:"title"`
							Description string `json:"description"`
							Priority    int    `json:"priority"`
							State       struct {
								Name string `json:"name"`
							} `json:"state"`
							Team struct {
								Key  string `json:"key"`
								Name string `json:"name"`
							} `json:"team"`
							Assignee *struct {
								Name  string `json:"name"`
								Email string `json:"email"`
							} `json:"assignee"`
							CreatedAt string `json:"createdAt"`
							UpdatedAt string `json:"updatedAt"`
							URL       string `json:"url"`
						} `json:"nodes"`
					} `json:"issues"`
				} `json:"data"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			issues := make([]map[string]any, len(result.Data.Issues.Nodes))
			for i, issue := range result.Data.Issues.Nodes {
				item := map[string]any{
					"id":         issue.ID,
					"identifier": issue.Identifier,
					"title":      issue.Title,
					"priority":   issue.Priority,
					"state":      issue.State.Name,
					"team":       issue.Team.Key,
					"team_name":  issue.Team.Name,
					"created_at": issue.CreatedAt,
					"updated_at": issue.UpdatedAt,
					"url":        issue.URL,
				}
				if issue.Assignee != nil {
					item["assignee"] = issue.Assignee.Name
				}
				issues[i] = item
			}

			return output.Print(map[string]any{
				"count":  len(issues),
				"issues": issues,
			})
		},
	}

	cmd.Flags().StringVarP(&team, "team", "t", "", "Team key (e.g., 'ENG')")
	cmd.Flags().StringVarP(&state, "state", "s", "", "State name filter")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of issues")
	cmd.Flags().BoolVarP(&assignedToMe, "mine", "m", false, "Only show issues assigned to me")

	return cmd
}

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teams",
		Short: "List teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newLinearClient()
			if err != nil {
				return err
			}

			query := `
				query Teams {
					teams {
						nodes {
							id
							key
							name
							description
							issueCount
							createdAt
						}
					}
				}
			`

			body, err := client.doQuery(query, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Data struct {
					Teams struct {
						Nodes []struct {
							ID          string `json:"id"`
							Key         string `json:"key"`
							Name        string `json:"name"`
							Description string `json:"description"`
							IssueCount  int    `json:"issueCount"`
							CreatedAt   string `json:"createdAt"`
						} `json:"nodes"`
					} `json:"teams"`
				} `json:"data"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			teams := make([]map[string]any, len(result.Data.Teams.Nodes))
			for i, team := range result.Data.Teams.Nodes {
				teams[i] = map[string]any{
					"id":          team.ID,
					"key":         team.Key,
					"name":        team.Name,
					"description": team.Description,
					"issues":      team.IssueCount,
					"created_at":  team.CreatedAt,
				}
			}

			return output.Print(map[string]any{
				"count": len(teams),
				"teams": teams,
			})
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var team string
	var title string
	var priority int

	cmd := &cobra.Command{
		Use:   "create [description]",
		Short: "Create an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newLinearClient()
			if err != nil {
				return err
			}

			// First get the team ID
			teamQuery := `
				query Team($key: String!) {
					teams(filter: { key: { eq: $key } }) {
						nodes { id }
					}
				}
			`

			teamBody, err := client.doQuery(teamQuery, map[string]any{"key": team})
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var teamResult struct {
				Data struct {
					Teams struct {
						Nodes []struct {
							ID string `json:"id"`
						} `json:"nodes"`
					} `json:"teams"`
				} `json:"data"`
			}

			if err := json.Unmarshal(teamBody, &teamResult); err != nil || len(teamResult.Data.Teams.Nodes) == 0 {
				return output.PrintError("team_not_found", fmt.Sprintf("Team '%s' not found", team), nil)
			}

			teamID := teamResult.Data.Teams.Nodes[0].ID

			// Create the issue
			createQuery := `
				mutation CreateIssue($input: IssueCreateInput!) {
					issueCreate(input: $input) {
						issue {
							id
							identifier
							title
							url
						}
					}
				}
			`

			input := map[string]any{
				"teamId":      teamID,
				"title":       title,
				"description": args[0],
			}
			if priority > 0 && priority <= 4 {
				input["priority"] = priority
			}

			body, err := client.doQuery(createQuery, map[string]any{"input": input})
			if err != nil {
				return output.PrintError("create_failed", err.Error(), nil)
			}

			var result struct {
				Data struct {
					IssueCreate struct {
						Issue struct {
							ID         string `json:"id"`
							Identifier string `json:"identifier"`
							Title      string `json:"title"`
							URL        string `json:"url"`
						} `json:"issue"`
					} `json:"issueCreate"`
				} `json:"data"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			issue := result.Data.IssueCreate.Issue
			return output.Print(map[string]any{
				"id":         issue.ID,
				"identifier": issue.Identifier,
				"title":      issue.Title,
				"url":        issue.URL,
			})
		},
	}

	cmd.Flags().StringVarP(&team, "team", "t", "", "Team key (required)")
	cmd.Flags().StringVar(&title, "title", "", "Issue title (required)")
	cmd.Flags().IntVarP(&priority, "priority", "p", 0, "Priority (0=none, 1=urgent, 2=high, 3=medium, 4=low)")
	cmd.MarkFlagRequired("team")
	cmd.MarkFlagRequired("title")

	return cmd
}

func newMeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Get current user info",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newLinearClient()
			if err != nil {
				return err
			}

			query := `
				query Me {
					viewer {
						id
						name
						email
						displayName
						avatarUrl
						createdAt
						assignedIssues { nodes { id } }
					}
				}
			`

			body, err := client.doQuery(query, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Data struct {
					Viewer struct {
						ID             string `json:"id"`
						Name           string `json:"name"`
						Email          string `json:"email"`
						DisplayName    string `json:"displayName"`
						AvatarURL      string `json:"avatarUrl"`
						CreatedAt      string `json:"createdAt"`
						AssignedIssues struct {
							Nodes []struct {
								ID string `json:"id"`
							} `json:"nodes"`
						} `json:"assignedIssues"`
					} `json:"viewer"`
				} `json:"data"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			user := result.Data.Viewer
			return output.Print(map[string]any{
				"id":              user.ID,
				"name":            user.Name,
				"email":           user.Email,
				"display_name":    user.DisplayName,
				"avatar":          user.AvatarURL,
				"created_at":      user.CreatedAt,
				"assigned_issues": len(user.AssignedIssues.Nodes),
			})
		},
	}

	return cmd
}
