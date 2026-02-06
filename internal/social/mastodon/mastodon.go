package mastodon

import (
	"bytes"
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
		Use:     "mastodon",
		Aliases: []string{"masto", "fedi"},
		Short:   "Mastodon/Fediverse commands",
		Long:    "Mastodon API integration. Works with any Mastodon-compatible instance.",
	}

	cmd.AddCommand(newTimelineCmd())
	cmd.AddCommand(newPostCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newNotificationsCmd())
	cmd.AddCommand(newMeCmd())

	return cmd
}

type mastoClient struct {
	server     string
	token      string
	httpClient *http.Client
}

func newMastoClient() (*mastoClient, error) {
	server, err := config.MustGet("mastodon_server")
	if err != nil {
		return nil, err
	}
	token, err := config.MustGet("mastodon_token")
	if err != nil {
		return nil, err
	}

	// Ensure server has https://
	if server[0:4] != "http" {
		server = "https://" + server
	}

	return &mastoClient{
		server:     server,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *mastoClient) doRequest(method, endpoint string, body any) ([]byte, error) {
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

	reqURL := c.server + "/api/v1" + endpoint
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
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
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("Mastodon API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("Mastodon API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type status struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Content   string `json:"content"`
	URL       string `json:"url"`
	Account   struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Acct        string `json:"acct"`
	} `json:"account"`
	ReblogsCount    int     `json:"reblogs_count"`
	FavouritesCount int     `json:"favourites_count"`
	RepliesCount    int     `json:"replies_count"`
	Reblog          *status `json:"reblog,omitempty"`
}

func formatStatuses(statuses []status) []map[string]any {
	result := make([]map[string]any, len(statuses))
	for i, s := range statuses {
		// Handle boosts
		content := s
		if s.Reblog != nil {
			content = *s.Reblog
		}
		result[i] = map[string]any{
			"id":         s.ID,
			"author":     content.Account.Acct,
			"display":    content.Account.DisplayName,
			"content":    content.Content,
			"url":        content.URL,
			"created_at": content.CreatedAt,
			"reblogs":    content.ReblogsCount,
			"favourites": content.FavouritesCount,
			"replies":    content.RepliesCount,
			"is_boost":   s.Reblog != nil,
		}
	}
	return result
}

func newTimelineCmd() *cobra.Command {
	var limit int
	var timeline string

	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Get timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newMastoClient()
			if err != nil {
				return err
			}

			var endpoint string
			switch timeline {
			case "home":
				endpoint = "/timelines/home"
			case "local":
				endpoint = "/timelines/public?local=true"
			case "public", "federated":
				endpoint = "/timelines/public"
			default:
				endpoint = "/timelines/home"
			}

			if limit > 0 {
				if timeline == "local" {
					endpoint += fmt.Sprintf("&limit=%d", limit)
				} else {
					endpoint += fmt.Sprintf("?limit=%d", limit)
				}
			}

			body, err := client.doRequest("GET", endpoint, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var statuses []status
			if err := json.Unmarshal(body, &statuses); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"timeline": timeline,
				"count":    len(statuses),
				"statuses": formatStatuses(statuses),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of posts")
	cmd.Flags().StringVarP(&timeline, "type", "t", "home", "Timeline: home, local, public/federated")

	return cmd
}

func newPostCmd() *cobra.Command {
	var visibility string
	var replyTo string
	var spoiler string

	cmd := &cobra.Command{
		Use:   "post [content]",
		Short: "Post a toot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newMastoClient()
			if err != nil {
				return err
			}

			payload := map[string]any{
				"status":     args[0],
				"visibility": visibility,
			}

			if replyTo != "" {
				payload["in_reply_to_id"] = replyTo
			}
			if spoiler != "" {
				payload["spoiler_text"] = spoiler
			}

			body, err := client.doRequest("POST", "/statuses", payload)
			if err != nil {
				return output.PrintError("post_failed", err.Error(), nil)
			}

			var result status
			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"id":         result.ID,
				"url":        result.URL,
				"content":    result.Content,
				"visibility": visibility,
				"created_at": result.CreatedAt,
			})
		},
	}

	cmd.Flags().StringVarP(&visibility, "visibility", "V", "public", "Visibility: public, unlisted, private, direct")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Status ID to reply to")
	cmd.Flags().StringVar(&spoiler, "spoiler", "", "Content warning / spoiler text")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int
	var searchType string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Mastodon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newMastoClient()
			if err != nil {
				return err
			}

			query := url.QueryEscape(args[0])
			endpoint := fmt.Sprintf("/search?q=%s&limit=%d", query, limit)
			if searchType != "all" && searchType != "" {
				endpoint += "&type=" + searchType
			}

			body, err := client.doRequest("GET", "/api/v2/search?q="+query+fmt.Sprintf("&limit=%d", limit), nil)
			if err != nil {
				// Fall back to v1 search
				body, err = client.doRequest("GET", endpoint, nil)
				if err != nil {
					return output.PrintError("request_failed", err.Error(), nil)
				}
			}

			var result struct {
				Accounts []struct {
					ID          string `json:"id"`
					Username    string `json:"username"`
					Acct        string `json:"acct"`
					DisplayName string `json:"display_name"`
					Note        string `json:"note"`
					URL         string `json:"url"`
				} `json:"accounts"`
				Statuses []status `json:"statuses"`
				Hashtags []struct {
					Name string `json:"name"`
					URL  string `json:"url"`
				} `json:"hashtags"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			accounts := make([]map[string]any, len(result.Accounts))
			for i, a := range result.Accounts {
				accounts[i] = map[string]any{
					"id":       a.ID,
					"username": a.Username,
					"acct":     a.Acct,
					"display":  a.DisplayName,
					"url":      a.URL,
				}
			}

			hashtags := make([]map[string]any, len(result.Hashtags))
			for i, h := range result.Hashtags {
				hashtags[i] = map[string]any{
					"name": h.Name,
					"url":  h.URL,
				}
			}

			return output.Print(map[string]any{
				"query":    args[0],
				"accounts": accounts,
				"statuses": formatStatuses(result.Statuses),
				"hashtags": hashtags,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of results")
	cmd.Flags().StringVarP(&searchType, "type", "t", "all", "Type: accounts, hashtags, statuses, all")

	return cmd
}

func newNotificationsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "notifications",
		Short: "Get notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newMastoClient()
			if err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/notifications?limit=%d", limit)
			body, err := client.doRequest("GET", endpoint, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var notifications []struct {
				ID        string `json:"id"`
				Type      string `json:"type"`
				CreatedAt string `json:"created_at"`
				Account   struct {
					Acct        string `json:"acct"`
					DisplayName string `json:"display_name"`
				} `json:"account"`
				Status *status `json:"status,omitempty"`
			}

			if err := json.Unmarshal(body, &notifications); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			result := make([]map[string]any, len(notifications))
			for i, n := range notifications {
				item := map[string]any{
					"id":         n.ID,
					"type":       n.Type,
					"from":       n.Account.Acct,
					"display":    n.Account.DisplayName,
					"created_at": n.CreatedAt,
				}
				if n.Status != nil {
					item["status_id"] = n.Status.ID
					item["status_url"] = n.Status.URL
				}
				result[i] = item
			}

			return output.Print(map[string]any{
				"count":         len(notifications),
				"notifications": result,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of notifications")

	return cmd
}

func newMeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Get your account info",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newMastoClient()
			if err != nil {
				return err
			}

			body, err := client.doRequest("GET", "/accounts/verify_credentials", nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var account struct {
				ID             string `json:"id"`
				Username       string `json:"username"`
				Acct           string `json:"acct"`
				DisplayName    string `json:"display_name"`
				Note           string `json:"note"`
				URL            string `json:"url"`
				Avatar         string `json:"avatar"`
				FollowersCount int    `json:"followers_count"`
				FollowingCount int    `json:"following_count"`
				StatusesCount  int    `json:"statuses_count"`
				CreatedAt      string `json:"created_at"`
			}

			if err := json.Unmarshal(body, &account); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"id":         account.ID,
				"username":   account.Username,
				"acct":       account.Acct,
				"display":    account.DisplayName,
				"bio":        account.Note,
				"url":        account.URL,
				"avatar":     account.Avatar,
				"followers":  account.FollowersCount,
				"following":  account.FollowingCount,
				"statuses":   account.StatusesCount,
				"created_at": account.CreatedAt,
			})
		},
	}

	return cmd
}
