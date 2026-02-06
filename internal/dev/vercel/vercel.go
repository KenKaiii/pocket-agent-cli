package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://api.vercel.com"

var httpClient = &http.Client{}

// Project is LLM-friendly project output
type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Framework string `json:"framework,omitempty"`
	Updated   string `json:"updated"`
	URL       string `json:"url,omitempty"`
}

// Deployment is LLM-friendly deployment output
type Deployment struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	State   string `json:"state"`
	Created string `json:"created"`
	Target  string `json:"target,omitempty"`
}

// Domain is LLM-friendly domain output
type Domain struct {
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
	Verified   bool   `json:"verified"`
	Created    string `json:"created"`
}

// EnvVar is LLM-friendly environment variable output
type EnvVar struct {
	ID      string   `json:"id"`
	Key     string   `json:"key"`
	Type    string   `json:"type"`
	Targets []string `json:"targets,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vercel",
		Aliases: []string{"vc"},
		Short:   "Vercel commands",
	}

	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newProjectCmd())
	cmd.AddCommand(newDeploymentsCmd())
	cmd.AddCommand(newDeploymentCmd())
	cmd.AddCommand(newDomainsCmd())
	cmd.AddCommand(newEnvCmd())

	return cmd
}

func newProjectsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v9/projects?limit=%d", baseURL, limit)

			var resp map[string]any
			if err := vcGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			projects, _ := resp["projects"].([]any)
			result := make([]Project, 0, len(projects))
			for _, p := range projects {
				if proj, ok := p.(map[string]any); ok {
					result = append(result, toProject(proj))
				}
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of projects")

	return cmd
}

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project [name]",
		Short: "Get project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v9/projects/%s", baseURL, args[0])

			var proj map[string]any
			if err := vcGet(token, url, &proj); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			return output.Print(toProject(proj))
		},
	}

	return cmd
}

func newDeploymentsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "deployments [project]",
		Short: "List deployments for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v6/deployments?projectId=%s&limit=%d", baseURL, args[0], limit)

			var resp map[string]any
			if err := vcGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			deployments, _ := resp["deployments"].([]any)
			result := make([]Deployment, 0, len(deployments))
			for _, d := range deployments {
				if dep, ok := d.(map[string]any); ok {
					result = append(result, toDeployment(dep))
				}
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of deployments")

	return cmd
}

func newDeploymentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment [id]",
		Short: "Get deployment details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v13/deployments/%s", baseURL, args[0])

			var dep map[string]any
			if err := vcGet(token, url, &dep); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			return output.Print(toDeployment(dep))
		},
	}

	return cmd
}

func newDomainsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "domains",
		Short: "List all domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v5/domains?limit=%d", baseURL, limit)

			var resp map[string]any
			if err := vcGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			domains, _ := resp["domains"].([]any)
			result := make([]Domain, 0, len(domains))
			for _, d := range domains {
				if dom, ok := d.(map[string]any); ok {
					result = append(result, toDomain(dom))
				}
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of domains")

	return cmd
}

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env [project]",
		Short: "List environment variables for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v10/projects/%s/env", baseURL, args[0])

			var resp map[string]any
			if err := vcGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			envs, _ := resp["envs"].([]any)
			result := make([]EnvVar, 0, len(envs))
			for _, e := range envs {
				if env, ok := e.(map[string]any); ok {
					result = append(result, toEnvVar(env))
				}
			}

			return output.Print(result)
		},
	}

	return cmd
}

func getToken() (string, error) {
	token, err := config.Get("vercel_token")
	if err != nil {
		return "", output.PrintError("config_error", err.Error(), nil)
	}
	if token == "" {
		return "", output.PrintError("missing_config", "vercel_token not configured", map[string]string{
			"setup": "Get your token from https://vercel.com/account/tokens then run: pocket config set vercel_token <token>",
		})
	}
	return token, nil
}

func vcGet(token, url string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errObj, ok := errResp["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok {
				return fmt.Errorf("%s", msg)
			}
		}
		return fmt.Errorf("%s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func toProject(p map[string]any) Project {
	proj := Project{
		ID:        getString(p, "id"),
		Name:      getString(p, "name"),
		Framework: getString(p, "framework"),
	}

	if updated := getInt64(p, "updatedAt"); updated > 0 {
		proj.Updated = timeAgo(time.UnixMilli(updated))
	}

	// Build project URL
	if proj.Name != "" {
		proj.URL = fmt.Sprintf("https://vercel.com/%s", proj.Name)
	}

	return proj
}

func toDeployment(d map[string]any) Deployment {
	dep := Deployment{
		ID:    getString(d, "uid"),
		Name:  getString(d, "name"),
		URL:   getString(d, "url"),
		State: getString(d, "state"),
	}

	// Fallback for id field
	if dep.ID == "" {
		dep.ID = getString(d, "id")
	}

	// Add https:// prefix if URL doesn't have it
	if dep.URL != "" && dep.URL[:4] != "http" {
		dep.URL = "https://" + dep.URL
	}

	if created := getInt64(d, "created"); created > 0 {
		dep.Created = timeAgo(time.UnixMilli(created))
	} else if created := getInt64(d, "createdAt"); created > 0 {
		dep.Created = timeAgo(time.UnixMilli(created))
	}

	if target := getString(d, "target"); target != "" {
		dep.Target = target
	}

	// Map readyState to state if state is empty
	if dep.State == "" {
		dep.State = getString(d, "readyState")
	}

	return dep
}

func toDomain(d map[string]any) Domain {
	dom := Domain{
		Name:       getString(d, "name"),
		Configured: getBool(d, "configured"),
		Verified:   getBool(d, "verified"),
	}

	if created := getInt64(d, "createdAt"); created > 0 {
		dom.Created = timeAgo(time.UnixMilli(created))
	}

	return dom
}

func toEnvVar(e map[string]any) EnvVar {
	env := EnvVar{
		ID:   getString(e, "id"),
		Key:  getString(e, "key"),
		Type: getString(e, "type"),
	}

	if targets, ok := e["target"].([]any); ok {
		for _, t := range targets {
			if target, ok := t.(string); ok {
				env.Targets = append(env.Targets, target)
			}
		}
	}

	return env
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
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
