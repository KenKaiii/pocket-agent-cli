package npm

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

const baseURL = "https://registry.npmjs.org"

var client = &http.Client{Timeout: 10 * time.Second}

// Package is LLM-friendly package info
type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"desc,omitempty"`
	Author      string   `json:"author,omitempty"`
	License     string   `json:"license,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repo,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Dependencies int     `json:"deps"`
	Updated     string   `json:"updated,omitempty"`
}

// SearchResult is LLM-friendly search result
type SearchResult struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"desc,omitempty"`
	Author      string `json:"author,omitempty"`
	Downloads   int    `json:"downloads,omitempty"`
	Score       float64 `json:"score,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npm",
		Short: "npm registry commands",
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
		Short: "Search npm packages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.QueryEscape(args[0])
			reqURL := fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=%d", query, limit)

			resp, err := client.Get(reqURL)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			var data struct {
				Objects []struct {
					Package struct {
						Name        string `json:"name"`
						Version     string `json:"version"`
						Description string `json:"description"`
						Publisher   struct {
							Username string `json:"username"`
						} `json:"publisher"`
					} `json:"package"`
					Score struct {
						Final float64 `json:"final"`
					} `json:"score"`
				} `json:"objects"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			results := make([]SearchResult, 0, len(data.Objects))
			for _, obj := range data.Objects {
				results = append(results, SearchResult{
					Name:        obj.Package.Name,
					Version:     obj.Package.Version,
					Description: truncate(obj.Package.Description, 100),
					Author:      obj.Package.Publisher.Username,
					Score:       obj.Score.Final,
				})
			}

			return output.Print(results)
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
			pkgName := args[0]
			reqURL := fmt.Sprintf("%s/%s", baseURL, url.PathEscape(pkgName))

			resp, err := client.Get(reqURL)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Package not found: "+pkgName, nil)
			}

			var data struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				DistTags    struct {
					Latest string `json:"latest"`
				} `json:"dist-tags"`
				License    string `json:"license"`
				Homepage   string `json:"homepage"`
				Repository struct {
					URL string `json:"url"`
				} `json:"repository"`
				Keywords []string `json:"keywords"`
				Author   struct {
					Name string `json:"name"`
				} `json:"author"`
				Time map[string]string `json:"time"`
				Versions map[string]struct {
					Dependencies map[string]string `json:"dependencies"`
				} `json:"versions"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			pkg := Package{
				Name:        data.Name,
				Version:     data.DistTags.Latest,
				Description: truncate(data.Description, 200),
				Author:      data.Author.Name,
				License:     data.License,
				Homepage:    data.Homepage,
				Keywords:    data.Keywords,
			}

			// Get repo URL
			if data.Repository.URL != "" {
				pkg.Repository = cleanRepoURL(data.Repository.URL)
			}

			// Get updated time
			if modified, ok := data.Time["modified"]; ok {
				pkg.Updated = parseTimeAgo(modified)
			}

			// Count dependencies for latest version
			if latest, ok := data.Versions[data.DistTags.Latest]; ok {
				pkg.Dependencies = len(latest.Dependencies)
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
			pkgName := args[0]
			reqURL := fmt.Sprintf("%s/%s", baseURL, url.PathEscape(pkgName))

			resp, err := client.Get(reqURL)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Package not found: "+pkgName, nil)
			}

			var data struct {
				DistTags map[string]string `json:"dist-tags"`
				Time     map[string]string `json:"time"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			type Version struct {
				Version   string `json:"version"`
				Tag       string `json:"tag,omitempty"`
				Published string `json:"published"`
			}

			// Build tag map
			tagMap := make(map[string]string)
			for tag, ver := range data.DistTags {
				tagMap[ver] = tag
			}

			// Get versions with times
			versions := make([]Version, 0)
			for ver, timeStr := range data.Time {
				if ver == "created" || ver == "modified" {
					continue
				}
				versions = append(versions, Version{
					Version:   ver,
					Tag:       tagMap[ver],
					Published: parseTimeAgo(timeStr),
				})
			}

			// Sort by most recent (simple approach: reverse order)
			// Actually npm returns them in order, but let's limit
			if len(versions) > limit {
				// Get the last N (most recent)
				versions = versions[len(versions)-limit:]
			}

			// Reverse to show newest first
			for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
				versions[i], versions[j] = versions[j], versions[i]
			}

			return output.Print(versions)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of versions")

	return cmd
}

func newDepsCmd() *cobra.Command {
	var dev bool

	cmd := &cobra.Command{
		Use:   "deps [package]",
		Short: "List package dependencies",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgName := args[0]
			reqURL := fmt.Sprintf("%s/%s/latest", baseURL, url.PathEscape(pkgName))

			resp, err := client.Get(reqURL)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Package not found: "+pkgName, nil)
			}

			var data struct {
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			type Dep struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Dev     bool   `json:"dev,omitempty"`
			}

			deps := make([]Dep, 0)

			for name, ver := range data.Dependencies {
				deps = append(deps, Dep{Name: name, Version: ver})
			}

			if dev {
				for name, ver := range data.DevDependencies {
					deps = append(deps, Dep{Name: name, Version: ver, Dev: true})
				}
			}

			return output.Print(deps)
		},
	}

	cmd.Flags().BoolVarP(&dev, "dev", "d", false, "Include dev dependencies")

	return cmd
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func cleanRepoURL(u string) string {
	u = strings.TrimPrefix(u, "git+")
	u = strings.TrimPrefix(u, "git://")
	u = strings.TrimSuffix(u, ".git")
	if strings.HasPrefix(u, "github.com") {
		u = "https://" + u
	}
	return u
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
