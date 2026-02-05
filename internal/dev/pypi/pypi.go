package pypi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://pypi.org"

var client = &http.Client{Timeout: 10 * time.Second}

// Package is LLM-friendly package info
type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Summary     string   `json:"summary,omitempty"`
	Author      string   `json:"author,omitempty"`
	License     string   `json:"license,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repo,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Requires    []string `json:"requires,omitempty"`
	PythonReq   string   `json:"python,omitempty"`
	Updated     string   `json:"updated,omitempty"`
}

// SearchResult is LLM-friendly search result
type SearchResult struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Summary string `json:"summary,omitempty"`
}

// Version info
type Version struct {
	Version  string `json:"version"`
	Released string `json:"released"`
	Python   string `json:"python,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pypi",
		Aliases: []string{"pip", "python"},
		Short:   "PyPI registry commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newInfoCmd())
	cmd.AddCommand(newVersionsCmd())
	cmd.AddCommand(newDepsCmd())

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search PyPI packages",
		Long:  "Search PyPI packages. Note: Uses PyPI simple search which may be limited.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// PyPI doesn't have a great search API, so we'll use a workaround
			// Search via the warehouse API (unofficial but works)
			query := url.QueryEscape(args[0])
			reqURL := fmt.Sprintf("https://pypi.org/search/?q=%s", query)

			// Since PyPI search returns HTML, we'll suggest using pip search locally
			// or fetch top packages matching the name directly
			// Let's try fetching the package directly first

			// Try direct package lookup
			var pkg pypiResponse
			if err := pypiGet(fmt.Sprintf("%s/pypi/%s/json", baseURL, args[0]), &pkg); err == nil {
				return output.Print([]SearchResult{{
					Name:    pkg.Info.Name,
					Version: pkg.Info.Version,
					Summary: truncate(pkg.Info.Summary, 100),
				}})
			}

			// If direct lookup fails, return a helpful message
			return output.Print(map[string]any{
				"note":    "PyPI search API is limited. Try exact package name or use: pip search",
				"suggest": fmt.Sprintf("pocket dev pypi info %s", args[0]),
				"web":     reqURL,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of results")

	return cmd
}

func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [package]",
		Short: "Get package info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgName := strings.ToLower(args[0])
			reqURL := fmt.Sprintf("%s/pypi/%s/json", baseURL, url.PathEscape(pkgName))

			var data pypiResponse
			if err := pypiGet(reqURL, &data); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			pkg := Package{
				Name:      data.Info.Name,
				Version:   data.Info.Version,
				Summary:   truncate(data.Info.Summary, 200),
				Author:    data.Info.Author,
				License:   data.Info.License,
				Homepage:  data.Info.HomePage,
				PythonReq: data.Info.RequiresPython,
			}

			// Get repo URL from project URLs
			if data.Info.ProjectURLs != nil {
				if repo, ok := data.Info.ProjectURLs["Repository"]; ok {
					pkg.Repository = repo
				} else if repo, ok := data.Info.ProjectURLs["Source"]; ok {
					pkg.Repository = repo
				} else if repo, ok := data.Info.ProjectURLs["GitHub"]; ok {
					pkg.Repository = repo
				}
			}

			// Keywords
			if data.Info.Keywords != "" {
				pkg.Keywords = strings.Split(data.Info.Keywords, ",")
				for i := range pkg.Keywords {
					pkg.Keywords[i] = strings.TrimSpace(pkg.Keywords[i])
				}
			}

			// Get upload time from releases
			if releases, ok := data.Releases[data.Info.Version]; ok && len(releases) > 0 {
				pkg.Updated = parseTimeAgo(releases[0].UploadTime)
			}

			// Dependencies (requires_dist)
			if len(data.Info.RequiresDist) > 0 {
				for _, req := range data.Info.RequiresDist {
					// Extract just the package name (before any version specifier or extras)
					name := strings.Split(req, " ")[0]
					name = strings.Split(name, ";")[0]
					name = strings.Split(name, "[")[0]
					name = strings.Split(name, "<")[0]
					name = strings.Split(name, ">")[0]
					name = strings.Split(name, "=")[0]
					name = strings.Split(name, "!")[0]
					if name != "" {
						pkg.Requires = append(pkg.Requires, name)
					}
				}
			}

			return output.Print(pkg)
		},
	}

	return cmd
}

func newVersionsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "versions [package]",
		Short: "List package versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgName := strings.ToLower(args[0])
			reqURL := fmt.Sprintf("%s/pypi/%s/json", baseURL, url.PathEscape(pkgName))

			var data pypiResponse
			if err := pypiGet(reqURL, &data); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			// Collect versions with release info
			type versionInfo struct {
				ver  string
				time string
			}

			versions := make([]versionInfo, 0)
			for ver, releases := range data.Releases {
				if len(releases) > 0 {
					versions = append(versions, versionInfo{
						ver:  ver,
						time: releases[0].UploadTime,
					})
				}
			}

			// Sort by upload time (newest first) - simple bubble sort
			for i := 0; i < len(versions); i++ {
				for j := i + 1; j < len(versions); j++ {
					if versions[j].time > versions[i].time {
						versions[i], versions[j] = versions[j], versions[i]
					}
				}
			}

			// Limit results
			if len(versions) > limit {
				versions = versions[:limit]
			}

			result := make([]Version, 0, len(versions))
			for _, v := range versions {
				result = append(result, Version{
					Version:  v.ver,
					Released: parseTimeAgo(v.time),
				})
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of versions")

	return cmd
}

func newDepsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps [package]",
		Short: "List package dependencies",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgName := strings.ToLower(args[0])
			reqURL := fmt.Sprintf("%s/pypi/%s/json", baseURL, url.PathEscape(pkgName))

			var data pypiResponse
			if err := pypiGet(reqURL, &data); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			type Dep struct {
				Name      string `json:"name"`
				Specifier string `json:"spec,omitempty"`
				Extra     string `json:"extra,omitempty"`
			}

			deps := make([]Dep, 0)
			for _, req := range data.Info.RequiresDist {
				dep := Dep{}

				// Parse the requirement string
				// Format: "name[extra] (>=1.0) ; condition"
				parts := strings.SplitN(req, ";", 2)
				main := strings.TrimSpace(parts[0])

				// Check for extras condition
				if len(parts) > 1 {
					cond := strings.TrimSpace(parts[1])
					if strings.Contains(cond, "extra ==") {
						// Extract extra name
						start := strings.Index(cond, "'")
						end := strings.LastIndex(cond, "'")
						if start != -1 && end > start {
							dep.Extra = cond[start+1 : end]
						}
					}
				}

				// Parse name and version specifier
				for _, sep := range []string{">=", "<=", "==", "!=", ">", "<", "~="} {
					if idx := strings.Index(main, sep); idx != -1 {
						dep.Name = strings.TrimSpace(main[:idx])
						dep.Specifier = strings.TrimSpace(main[idx:])
						break
					}
				}

				if dep.Name == "" {
					// No version specifier
					dep.Name = strings.Split(main, "[")[0]
					dep.Name = strings.TrimSpace(dep.Name)
				}

				if dep.Name != "" {
					deps = append(deps, dep)
				}
			}

			return output.Print(deps)
		},
	}

	return cmd
}

type pypiResponse struct {
	Info struct {
		Name           string            `json:"name"`
		Version        string            `json:"version"`
		Summary        string            `json:"summary"`
		Author         string            `json:"author"`
		License        string            `json:"license"`
		HomePage       string            `json:"home_page"`
		Keywords       string            `json:"keywords"`
		RequiresPython string            `json:"requires_python"`
		RequiresDist   []string          `json:"requires_dist"`
		ProjectURLs    map[string]string `json:"project_urls"`
	} `json:"info"`
	Releases map[string][]struct {
		UploadTime string `json:"upload_time"`
		PythonReq  string `json:"requires_python"`
	} `json:"releases"`
}

func pypiGet(url string, result any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("package not found")
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func parseTimeAgo(ts string) string {
	// PyPI uses format: "2024-01-15T10:30:00"
	t, err := time.Parse("2006-01-02T15:04:05", ts)
	if err != nil {
		// Try with timezone
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return ""
		}
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
