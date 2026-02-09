package twitter

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	tweetEndpoint = "https://api.x.com/2/tweets"
	userEndpoint  = "https://api.x.com/2/users"
	authEndpoint  = "https://x.com/i/oauth2/authorize"
	tokenEndpoint = "https://api.x.com/2/oauth2/token"
	callbackPort  = "8765"
	redirectURI   = "http://127.0.0.1:" + callbackPort + "/callback"
	defaultScopes = "tweet.read tweet.write users.read offline.access"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "twitter",
		Aliases: []string{"tw", "x"},
		Short:   "Twitter/X commands",
		Long:    "Twitter/X API v2 integration using OAuth 2.0 with PKCE.",
	}

	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newPostCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newMeCmd())
	cmd.AddCommand(newTimelineCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

// OAuth 2.0 client for X API
type xClient struct {
	clientID   string
	token      string
	httpClient *http.Client
}

func newXClient() (*xClient, error) {
	clientID, err := config.MustGet("x_client_id")
	if err != nil {
		return nil, err
	}

	accessToken, _ := config.Get("x_access_token")
	refreshToken, _ := config.Get("x_refresh_token")
	expiryStr, _ := config.Get("x_token_expiry")

	// Check if token exists and is valid
	if accessToken != "" && expiryStr != "" {
		expiry, _ := time.Parse(time.RFC3339, expiryStr)
		if time.Now().Before(expiry) {
			return &xClient{
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
			return &xClient{
				clientID:   clientID,
				token:      newToken,
				httpClient: &http.Client{Timeout: 30 * time.Second},
			}, nil
		}
	}

	return nil, output.PrintError("auth_required",
		"X/Twitter OAuth not configured or expired. Run: pocket social twitter auth",
		nil)
}

func refreshAccessToken(clientID, refreshToken string) (string, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	// Store new tokens (X uses rotating refresh tokens)
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	config.Set("x_access_token", tokenResp.AccessToken)
	config.Set("x_refresh_token", tokenResp.RefreshToken)
	config.Set("x_token_expiry", expiry.Format(time.RFC3339))

	return tokenResp.AccessToken, nil
}

func (c *xClient) doRequest(method, reqURL string, body any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = strings.NewReader(string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

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
			Detail string `json:"detail"`
			Title  string `json:"title"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if json.Unmarshal(respBody, &errResp) == nil {
			if errResp.Detail != "" {
				return nil, fmt.Errorf("X API error: %s", errResp.Detail)
			}
			if len(errResp.Errors) > 0 {
				return nil, fmt.Errorf("X API error: %s", errResp.Errors[0].Message)
			}
		}
		return nil, fmt.Errorf("X API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// PKCE helpers

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with X/Twitter using OAuth 2.0",
		Long:  "Opens your browser to authorize Pocket CLI with your X account.\nRequires x_client_id to be configured first.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID, err := config.MustGet("x_client_id")
			if err != nil {
				return output.PrintError("setup_required",
					"x_client_id not configured. Set it first: pocket config set x_client_id <your-client-id>",
					map[string]string{
						"setup": "1. Go to https://developer.x.com/en/portal/dashboard\n2. Create or select your app\n3. Enable OAuth 2.0 and select 'Native App'\n4. Set callback URL to " + redirectURI + "\n5. Copy the Client ID",
					})
			}

			verifier, err := generateCodeVerifier()
			if err != nil {
				return output.PrintError("auth_error", "Failed to generate PKCE verifier: "+err.Error(), nil)
			}

			state, err := generateState()
			if err != nil {
				return output.PrintError("auth_error", "Failed to generate state: "+err.Error(), nil)
			}

			challenge := generateCodeChallenge(verifier)

			// Build authorization URL
			params := url.Values{
				"response_type":         {"code"},
				"client_id":             {clientID},
				"redirect_uri":          {redirectURI},
				"scope":                 {defaultScopes},
				"state":                 {state},
				"code_challenge":        {challenge},
				"code_challenge_method": {"S256"},
			}
			authorizationURL := authEndpoint + "?" + params.Encode()

			// Start local callback server
			listener, err := net.Listen("tcp", "127.0.0.1:"+callbackPort)
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
			fmt.Println("Opening browser for X/Twitter authorization...")
			fmt.Println("If the browser doesn't open, visit this URL:")
			fmt.Println(authorizationURL)
			exec.Command("open", authorizationURL).Start()

			// Wait for callback (with timeout)
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
				"grant_type":    {"authorization_code"},
				"code":          {code},
				"redirect_uri":  {redirectURI},
				"client_id":     {clientID},
				"code_verifier": {verifier},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(tokenData.Encode()))
			if err != nil {
				return output.PrintError("auth_error", "Failed to create token request: "+err.Error(), nil)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
			}
			if err := json.Unmarshal(respBody, &tokenResp); err != nil {
				return output.PrintError("auth_error", "Failed to parse token response: "+err.Error(), nil)
			}

			// Store tokens
			expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
			config.Set("x_access_token", tokenResp.AccessToken)
			config.Set("x_refresh_token", tokenResp.RefreshToken)
			config.Set("x_token_expiry", expiry.Format(time.RFC3339))

			return output.Print(map[string]any{
				"status": "authenticated",
				"scopes": tokenResp.Scope,
				"expiry": expiry.Format(time.RFC3339),
			})
		},
	}

	return cmd
}

func newPostCmd() *cobra.Command {
	var replyTo string

	cmd := &cobra.Command{
		Use:   "post [message]",
		Short: "Post a tweet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newXClient()
			if err != nil {
				return err
			}

			payload := map[string]any{
				"text": args[0],
			}

			if replyTo != "" {
				payload["reply"] = map[string]string{
					"in_reply_to_tweet_id": replyTo,
				}
			}

			respBody, err := client.doRequest("POST", tweetEndpoint, payload)
			if err != nil {
				return output.PrintError("post_failed", err.Error(), nil)
			}

			var result struct {
				Data struct {
					ID   string `json:"id"`
					Text string `json:"text"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"id":   result.Data.ID,
				"text": result.Data.Text,
				"url":  fmt.Sprintf("https://x.com/i/status/%s", result.Data.ID),
			})
		},
	}

	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Tweet ID to reply to")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [tweet-id]",
		Short: "Delete a tweet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newXClient()
			if err != nil {
				return err
			}

			deleteURL := fmt.Sprintf("%s/%s", tweetEndpoint, args[0])
			_, err = client.doRequest("DELETE", deleteURL, nil)
			if err != nil {
				return output.PrintError("delete_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"deleted": true,
				"id":      args[0],
			})
		},
	}

	return cmd
}

func newMeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Get your account info",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newXClient()
			if err != nil {
				return err
			}

			meURL := userEndpoint + "/me?user.fields=id,name,username,description,public_metrics,profile_image_url,created_at"
			respBody, err := client.doRequest("GET", meURL, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Data struct {
					ID              string `json:"id"`
					Name            string `json:"name"`
					Username        string `json:"username"`
					Description     string `json:"description"`
					ProfileImageURL string `json:"profile_image_url"`
					CreatedAt       string `json:"created_at"`
					PublicMetrics   struct {
						Followers  int `json:"followers_count"`
						Following  int `json:"following_count"`
						TweetCount int `json:"tweet_count"`
					} `json:"public_metrics"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"id":          result.Data.ID,
				"name":        result.Data.Name,
				"username":    result.Data.Username,
				"description": result.Data.Description,
				"avatar":      result.Data.ProfileImageURL,
				"created_at":  result.Data.CreatedAt,
				"followers":   result.Data.PublicMetrics.Followers,
				"following":   result.Data.PublicMetrics.Following,
				"tweets":      result.Data.PublicMetrics.TweetCount,
				"url":         fmt.Sprintf("https://x.com/%s", result.Data.Username),
			})
		},
	}

	return cmd
}

func newTimelineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Get home timeline (requires paid tier)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.PrintError("paid_tier_required",
				"Timeline access requires X API Basic tier ($200/month) or higher. Free tier only supports posting tweets.",
				map[string]string{
					"pricing": "https://developer.x.com/en/products/twitter-api",
				})
		},
	}

	return cmd
}

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search tweets (requires paid tier)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.PrintError("paid_tier_required",
				"Search access requires X API Basic tier ($200/month) or higher. Free tier only supports posting tweets.",
				map[string]string{
					"pricing": "https://developer.x.com/en/products/twitter-api",
				})
		},
	}

	return cmd
}
