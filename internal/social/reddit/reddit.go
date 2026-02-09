package reddit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const (
	authURL      = "https://www.reddit.com/api/v1/authorize"
	tokenURL     = "https://www.reddit.com/api/v1/access_token"
	apiBaseURL   = "https://oauth.reddit.com"
	userAgent    = "pocket-cli/1.0"
	callbackPort = "8766"
	redirectURI  = "http://localhost:" + callbackPort + "/callback"
	scopes       = "identity read mysubreddits history"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reddit",
		Aliases: []string{"rd"},
		Short:   "Reddit commands",
		Long:    "Reddit API integration using OAuth 2.0. Free tier for non-commercial use (100 req/min).",
	}

	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newFeedCmd())
	cmd.AddCommand(newSubredditCmd())
	cmd.AddCommand(newPostCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newUserCmd())
	cmd.AddCommand(newCommentsCmd())

	return cmd
}

type redditClient struct {
	clientID   string
	token      string
	httpClient *http.Client
}

func newRedditClient() (*redditClient, error) {
	clientID, err := config.MustGet("reddit_client_id")
	if err != nil {
		return nil, err
	}

	accessToken, _ := config.Get("reddit_access_token")
	refreshToken, _ := config.Get("reddit_refresh_token")
	expiryStr, _ := config.Get("reddit_token_expiry")

	// Check if token exists and is valid
	if accessToken != "" && expiryStr != "" {
		expiry, _ := time.Parse(time.RFC3339, expiryStr)
		if time.Now().Before(expiry) {
			return &redditClient{
				clientID:   clientID,
				token:      accessToken,
				httpClient: &http.Client{Timeout: 30 * time.Second},
			}, nil
		}
	}

	// Token expired or missing — try refresh
	if refreshToken != "" {
		newToken, err := refreshAccessToken(clientID, refreshToken)
		if err == nil {
			return &redditClient{
				clientID:   clientID,
				token:      newToken,
				httpClient: &http.Client{Timeout: 30 * time.Second},
			}, nil
		}
	}

	return nil, output.PrintError("auth_required",
		"Reddit OAuth not configured or expired. Run: pocket social reddit auth",
		nil)
}

func refreshAccessToken(clientID, refreshToken string) (string, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	// Installed apps use Basic auth with empty password
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("auth error: %s", tokenResp.Error)
	}

	// Store new access token (Reddit refresh tokens don't rotate)
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	config.Set("reddit_access_token", tokenResp.AccessToken)
	config.Set("reddit_token_expiry", expiry.Format(time.RFC3339))

	return tokenResp.AccessToken, nil
}

func (c *redditClient) doRequest(method, endpoint string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := apiBaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", userAgent)

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
		return nil, fmt.Errorf("Reddit API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// generateState is unused but retained for potential future use outside auth command.
// The auth command generates state inline.

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Reddit using OAuth 2.0",
		Long:  "Opens your browser to authorize Pocket CLI with your Reddit account.\nRequires reddit_client_id to be configured first.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID, err := config.MustGet("reddit_client_id")
			if err != nil {
				return output.PrintError("setup_required",
					"reddit_client_id not configured. Set it first: pocket config set reddit_client_id <your-client-id>",
					map[string]string{
						"setup": "1. Go to https://www.reddit.com/prefs/apps\n2. Click 'create another app'\n3. Select 'installed app' type\n4. Set redirect URI to " + redirectURI + "\n5. Copy the client ID (shown under app name)",
					})
			}

			state := fmt.Sprintf("%d", time.Now().UnixNano())

			// Build authorization URL
			params := url.Values{
				"client_id":     {clientID},
				"response_type": {"code"},
				"state":         {state},
				"redirect_uri":  {redirectURI},
				"duration":      {"permanent"},
				"scope":         {scopes},
			}
			authorizationURL := authURL + "?" + params.Encode()

			// Start local callback server
			listener, err := net.Listen("tcp", "localhost:"+callbackPort)
			if err != nil {
				return output.PrintError("auth_error",
					fmt.Sprintf("Failed to start callback server on port %s: %s", callbackPort, err.Error()), nil)
			}

			codeCh := make(chan string, 1)
			errCh := make(chan string, 1)

			mux := http.NewServeMux()
			mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("state") != state {
					errCh <- "state mismatch"
					fmt.Fprintf(w, "Error: state mismatch. Please try again.")
					return
				}
				if errMsg := r.URL.Query().Get("error"); errMsg != "" {
					errCh <- errMsg
					fmt.Fprintf(w, "Authorization denied: %s", errMsg)
					return
				}
				code := r.URL.Query().Get("code")
				if code == "" {
					errCh <- "no code in callback"
					fmt.Fprintf(w, "Error: no authorization code received.")
					return
				}
				codeCh <- code
				fmt.Fprintf(w, "Authorization successful! You can close this tab and return to the terminal.")
			})

			server := &http.Server{Handler: mux}
			go server.Serve(listener)

			// Open browser
			fmt.Println("Opening browser for Reddit authorization...")
			fmt.Println("If the browser doesn't open, visit this URL:")
			fmt.Println(authorizationURL)
			exec.Command("open", authorizationURL).Start()

			// Wait for callback
			fmt.Println("\nWaiting for authorization...")
			var code string
			select {
			case code = <-codeCh:
				// Success
			case errMsg := <-errCh:
				server.Close()
				return output.PrintError("auth_denied", "Authorization failed: "+errMsg, nil)
			case <-time.After(5 * time.Minute):
				server.Close()
				return output.PrintError("auth_timeout", "Authorization timed out after 5 minutes", nil)
			}
			server.Close()

			// Exchange code for tokens
			tokenData := url.Values{
				"grant_type":   {"authorization_code"},
				"code":         {code},
				"redirect_uri": {redirectURI},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(tokenData.Encode()))
			if err != nil {
				return output.PrintError("auth_error", "Failed to create token request: "+err.Error(), nil)
			}

			// Installed apps: Basic auth with empty password
			authHeader := base64.StdEncoding.EncodeToString([]byte(clientID + ":"))
			req.Header.Set("Authorization", "Basic "+authHeader)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("User-Agent", userAgent)

			resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
			if err != nil {
				return output.PrintError("auth_error", "Token exchange failed: "+err.Error(), nil)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return output.PrintError("auth_error", "Failed to read token response: "+err.Error(), nil)
			}

			if resp.StatusCode != 200 {
				return output.PrintError("auth_error",
					fmt.Sprintf("Token exchange failed (HTTP %d): %s", resp.StatusCode, string(respBody)), nil)
			}

			var tokenResp struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				ExpiresIn    int    `json:"expires_in"`
				Scope        string `json:"scope"`
				Error        string `json:"error"`
			}
			if err := json.Unmarshal(respBody, &tokenResp); err != nil {
				return output.PrintError("auth_error", "Failed to parse token response: "+err.Error(), nil)
			}

			if tokenResp.Error != "" {
				return output.PrintError("auth_error", "Reddit auth error: "+tokenResp.Error, nil)
			}

			// Store tokens
			expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
			config.Set("reddit_access_token", tokenResp.AccessToken)
			config.Set("reddit_refresh_token", tokenResp.RefreshToken)
			config.Set("reddit_token_expiry", expiry.Format(time.RFC3339))

			return output.Print(map[string]any{
				"status": "authenticated",
				"scopes": tokenResp.Scope,
				"expiry": expiry.Format(time.RFC3339),
			})
		},
	}

	return cmd
}

type redditPost struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Subreddit string `json:"subreddit"`
	Score     int    `json:"score"`
	Comments  int    `json:"num_comments"`
	URL       string `json:"url"`
	Permalink string `json:"permalink"`
	Created   int64  `json:"created_utc"`
	SelfText  string `json:"selftext,omitempty"`
	IsNSFW    bool   `json:"over_18"`
}

func parseListingResponse(body []byte) ([]redditPost, error) {
	var listing struct {
		Data struct {
			Children []struct {
				Data redditPost `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, err
	}

	posts := make([]redditPost, len(listing.Data.Children))
	for i, child := range listing.Data.Children {
		posts[i] = child.Data
	}

	return posts, nil
}

func formatPosts(posts []redditPost) []map[string]any {
	result := make([]map[string]any, len(posts))
	for i, p := range posts {
		result[i] = map[string]any{
			"id":        p.ID,
			"title":     p.Title,
			"author":    p.Author,
			"subreddit": p.Subreddit,
			"score":     p.Score,
			"comments":  p.Comments,
			"url":       p.URL,
			"permalink": "https://reddit.com" + p.Permalink,
			"created":   time.Unix(p.Created, 0).Format(time.RFC3339),
			"nsfw":      p.IsNSFW,
		}
	}
	return result
}

func newFeedCmd() *cobra.Command {
	var limit int
	var sort string

	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Get your home feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newRedditClient()
			if err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/%s?limit=%d", sort, limit)
			body, err := client.doRequest("GET", endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			posts, err := parseListingResponse(body)
			if err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"count": len(posts),
				"posts": formatPosts(posts),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of posts (max 100)")
	cmd.Flags().StringVarP(&sort, "sort", "s", "hot", "Sort: hot, new, top, rising, best")

	return cmd
}

func newSubredditCmd() *cobra.Command {
	var limit int
	var sort string
	var timeFrame string

	cmd := &cobra.Command{
		Use:   "subreddit [name]",
		Short: "Get posts from a subreddit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newRedditClient()
			if err != nil {
				return err
			}

			subreddit := strings.TrimPrefix(args[0], "r/")
			endpoint := fmt.Sprintf("/r/%s/%s?limit=%d", subreddit, sort, limit)
			if sort == "top" || sort == "controversial" {
				endpoint += "&t=" + timeFrame
			}

			body, err := client.doRequest("GET", endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			posts, err := parseListingResponse(body)
			if err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"subreddit": subreddit,
				"count":     len(posts),
				"posts":     formatPosts(posts),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of posts (max 100)")
	cmd.Flags().StringVarP(&sort, "sort", "s", "hot", "Sort: hot, new, top, rising, controversial")
	cmd.Flags().StringVarP(&timeFrame, "time", "t", "day", "Time frame for top/controversial: hour, day, week, month, year, all")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int
	var subreddit string
	var sort string
	var timeFrame string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Reddit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newRedditClient()
			if err != nil {
				return err
			}

			query := url.QueryEscape(args[0])
			endpoint := fmt.Sprintf("/search?q=%s&limit=%d&sort=%s&t=%s", query, limit, sort, timeFrame)
			if subreddit != "" {
				endpoint = fmt.Sprintf("/r/%s/search?q=%s&limit=%d&sort=%s&t=%s&restrict_sr=on",
					strings.TrimPrefix(subreddit, "r/"), query, limit, sort, timeFrame)
			}

			body, err := client.doRequest("GET", endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			posts, err := parseListingResponse(body)
			if err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"query": args[0],
				"count": len(posts),
				"posts": formatPosts(posts),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of results (max 100)")
	cmd.Flags().StringVarP(&subreddit, "subreddit", "r", "", "Limit search to subreddit")
	cmd.Flags().StringVarP(&sort, "sort", "s", "relevance", "Sort: relevance, hot, top, new, comments")
	cmd.Flags().StringVarP(&timeFrame, "time", "t", "all", "Time frame: hour, day, week, month, year, all")

	return cmd
}

func newUserCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "user [username]",
		Short: "Get user info and posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newRedditClient()
			if err != nil {
				return err
			}

			username := strings.TrimPrefix(args[0], "u/")

			// Get user info
			aboutBody, err := client.doRequest("GET", "/user/"+username+"/about")
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var userInfo struct {
				Data struct {
					Name         string `json:"name"`
					Created      int64  `json:"created_utc"`
					LinkKarma    int    `json:"link_karma"`
					CommentKarma int    `json:"comment_karma"`
					IsGold       bool   `json:"is_gold"`
					IconImg      string `json:"icon_img"`
				} `json:"data"`
			}
			if err := json.Unmarshal(aboutBody, &userInfo); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			// Get recent posts
			postsBody, err := client.doRequest("GET", fmt.Sprintf("/user/%s/submitted?limit=%d", username, limit))
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			posts, _ := parseListingResponse(postsBody)

			return output.Print(map[string]any{
				"username":      userInfo.Data.Name,
				"created":       time.Unix(userInfo.Data.Created, 0).Format(time.RFC3339),
				"link_karma":    userInfo.Data.LinkKarma,
				"comment_karma": userInfo.Data.CommentKarma,
				"is_premium":    userInfo.Data.IsGold,
				"avatar":        userInfo.Data.IconImg,
				"url":           "https://reddit.com/u/" + username,
				"recent_posts":  formatPosts(posts),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of recent posts to fetch")

	return cmd
}

func newCommentsCmd() *cobra.Command {
	var limit int
	var sort string

	cmd := &cobra.Command{
		Use:   "comments [post-id]",
		Short: "Get comments on a post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newRedditClient()
			if err != nil {
				return err
			}

			postID := args[0]
			endpoint := fmt.Sprintf("/comments/%s?limit=%d&sort=%s", postID, limit, sort)

			body, err := client.doRequest("GET", endpoint)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			// Comments response is an array [post, comments]
			var response []json.RawMessage
			if err := json.Unmarshal(body, &response); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			if len(response) < 2 {
				return output.PrintError("parse_error", "unexpected response format", nil)
			}

			// Parse comments
			var commentsListing struct {
				Data struct {
					Children []struct {
						Data struct {
							ID        string `json:"id"`
							Author    string `json:"author"`
							Body      string `json:"body"`
							Score     int    `json:"score"`
							Created   int64  `json:"created_utc"`
							Permalink string `json:"permalink"`
						} `json:"data"`
					} `json:"children"`
				} `json:"data"`
			}

			if err := json.Unmarshal(response[1], &commentsListing); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			comments := make([]map[string]any, 0)
			for _, c := range commentsListing.Data.Children {
				if c.Data.Author == "" {
					continue // Skip "more" placeholders
				}
				comments = append(comments, map[string]any{
					"id":        c.Data.ID,
					"author":    c.Data.Author,
					"body":      c.Data.Body,
					"score":     c.Data.Score,
					"created":   time.Unix(c.Data.Created, 0).Format(time.RFC3339),
					"permalink": "https://reddit.com" + c.Data.Permalink,
				})
			}

			return output.Print(map[string]any{
				"post_id":  postID,
				"count":    len(comments),
				"comments": comments,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of comments")
	cmd.Flags().StringVarP(&sort, "sort", "s", "best", "Sort: best, top, new, controversial, old, qa")

	return cmd
}

func newPostCmd() *cobra.Command {
	var subreddit string
	var title string
	var flair string

	cmd := &cobra.Command{
		Use:   "post [content]",
		Short: "Create a text post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.PrintError("not_implemented",
				"Posting to Reddit requires the 'submit' OAuth scope. Re-run 'pocket social reddit auth' with submit scope enabled, then use Reddit's API.",
				map[string]string{
					"docs": "https://www.reddit.com/dev/api#POST_api_submit",
				})
		},
	}

	cmd.Flags().StringVarP(&subreddit, "subreddit", "r", "", "Subreddit to post to (required)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Post title (required)")
	cmd.Flags().StringVarP(&flair, "flair", "f", "", "Post flair ID")
	_ = cmd.MarkFlagRequired("subreddit")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}
