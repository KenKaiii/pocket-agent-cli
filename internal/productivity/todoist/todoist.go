package todoist

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

const apiBaseURL = "https://api.todoist.com/rest/v2"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "todoist",
		Aliases: []string{"todo"},
		Short:   "Todoist commands",
		Long:    "Todoist task management integration.",
	}

	cmd.AddCommand(newTasksCmd())
	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newCompleteCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

type todoistClient struct {
	token      string
	httpClient *http.Client
}

func newTodoistClient() (*todoistClient, error) {
	token, err := config.MustGet("todoist_token")
	if err != nil {
		return nil, err
	}

	return &todoistClient{
		token:      token,
		httpClient: &http.Client{},
	}, nil
}

func (c *todoistClient) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiBaseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Todoist API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type task struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Description string `json:"description"`
	ProjectID   string `json:"project_id"`
	Priority    int    `json:"priority"`
	Due         *struct {
		Date      string `json:"date"`
		Datetime  string `json:"datetime,omitempty"`
		String    string `json:"string"`
		Recurring bool   `json:"is_recurring"`
	} `json:"due,omitempty"`
	Labels    []string `json:"labels"`
	CreatedAt string   `json:"created_at"`
	URL       string   `json:"url"`
}

func formatTasks(tasks []task) []map[string]any {
	result := make([]map[string]any, len(tasks))
	for i, t := range tasks {
		item := map[string]any{
			"id":          t.ID,
			"content":     t.Content,
			"description": t.Description,
			"project_id":  t.ProjectID,
			"priority":    t.Priority,
			"labels":      t.Labels,
			"url":         t.URL,
		}
		if t.Due != nil {
			item["due"] = t.Due.String
			item["due_date"] = t.Due.Date
			item["recurring"] = t.Due.Recurring
		}
		result[i] = item
	}
	return result
}

func newTasksCmd() *cobra.Command {
	var projectID string
	var filter string

	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newTodoistClient()
			if err != nil {
				return err
			}

			endpoint := "/tasks"
			if projectID != "" {
				endpoint += "?project_id=" + projectID
			} else if filter != "" {
				endpoint += "?filter=" + filter
			}

			body, err := client.doRequest("GET", endpoint, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var tasks []task
			if err := json.Unmarshal(body, &tasks); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"count": len(tasks),
				"tasks": formatTasks(tasks),
			})
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID to filter by")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "Filter expression (e.g., 'today', 'overdue')")

	return cmd
}

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newTodoistClient()
			if err != nil {
				return err
			}

			body, err := client.doRequest("GET", "/projects", nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var projects []struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				Color          string `json:"color"`
				ParentID       string `json:"parent_id,omitempty"`
				Order          int    `json:"order"`
				CommentCount   int    `json:"comment_count"`
				IsFavorite     bool   `json:"is_favorite"`
				IsInboxProject bool   `json:"is_inbox_project"`
				URL            string `json:"url"`
			}

			if err := json.Unmarshal(body, &projects); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			result := make([]map[string]any, len(projects))
			for i, p := range projects {
				result[i] = map[string]any{
					"id":        p.ID,
					"name":      p.Name,
					"color":     p.Color,
					"parent_id": p.ParentID,
					"order":     p.Order,
					"comments":  p.CommentCount,
					"favorite":  p.IsFavorite,
					"is_inbox":  p.IsInboxProject,
					"url":       p.URL,
				}
			}

			return output.Print(map[string]any{
				"count":    len(projects),
				"projects": result,
			})
		},
	}

	return cmd
}

func newAddCmd() *cobra.Command {
	var projectID string
	var due string
	var priority int
	var labels []string
	var description string

	cmd := &cobra.Command{
		Use:   "add [content]",
		Short: "Add a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newTodoistClient()
			if err != nil {
				return err
			}

			payload := map[string]any{
				"content": args[0],
			}

			if projectID != "" {
				payload["project_id"] = projectID
			}
			if due != "" {
				payload["due_string"] = due
			}
			if priority > 0 && priority <= 4 {
				payload["priority"] = priority
			}
			if len(labels) > 0 {
				payload["labels"] = labels
			}
			if description != "" {
				payload["description"] = description
			}

			body, err := client.doRequest("POST", "/tasks", payload)
			if err != nil {
				return output.PrintError("create_failed", err.Error(), nil)
			}

			var result task
			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			resp := map[string]any{
				"id":      result.ID,
				"content": result.Content,
				"url":     result.URL,
			}
			if result.Due != nil {
				resp["due"] = result.Due.String
			}

			return output.Print(resp)
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID")
	cmd.Flags().StringVarP(&due, "due", "d", "", "Due date (e.g., 'today', 'tomorrow', 'next monday')")
	cmd.Flags().IntVar(&priority, "priority", 1, "Priority (1-4, where 4 is highest)")
	cmd.Flags().StringSliceVarP(&labels, "labels", "l", nil, "Labels to add")
	cmd.Flags().StringVar(&description, "desc", "", "Task description")

	return cmd
}

func newCompleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete [task-id]",
		Short: "Complete a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newTodoistClient()
			if err != nil {
				return err
			}

			endpoint := "/tasks/" + args[0] + "/close"
			_, err = client.doRequest("POST", endpoint, nil)
			if err != nil {
				return output.PrintError("complete_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"completed": true,
				"task_id":   args[0],
			})
		},
	}

	return cmd
}

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [task-id]",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newTodoistClient()
			if err != nil {
				return err
			}

			endpoint := "/tasks/" + args[0]
			_, err = client.doRequest("DELETE", endpoint, nil)
			if err != nil {
				return output.PrintError("delete_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"deleted": true,
				"task_id": args[0],
			})
		},
	}

	return cmd
}
