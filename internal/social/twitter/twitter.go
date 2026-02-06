package twitter

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const (
	tweetEndpoint = "https://api.x.com/2/tweets"
	userEndpoint  = "https://api.x.com/2/users"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "twitter",
		Aliases: []string{"tw", "x"},
		Short:   "Twitter/X commands",
		Long:    "Twitter/X API v2 integration. Free tier supports posting only (1,500 tweets/month).",
	}

	cmd.AddCommand(newPostCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newMeCmd())
	cmd.AddCommand(newTimelineCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

// OAuth 1.0a client for X API
type oauthClient struct {
	consumerKey    string
	consumerSecret string
	accessToken    string
	accessSecret   string
}

func newOAuthClient() (*oauthClient, error) {
	consumerKey, err := config.MustGet("x_api_key")
	if err != nil {
		return nil, err
	}
	consumerSecret, err := config.MustGet("x_api_secret")
	if err != nil {
		return nil, err
	}
	accessToken, err := config.MustGet("x_access_token")
	if err != nil {
		return nil, err
	}
	accessSecret, err := config.MustGet("x_access_secret")
	if err != nil {
		return nil, err
	}

	return &oauthClient{
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		accessToken:    accessToken,
		accessSecret:   accessSecret,
	}, nil
}

func (c *oauthClient) generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func (c *oauthClient) generateSignature(method, urlStr string, params map[string]string) string {
	// Collect all parameters
	allParams := make(map[string]string)
	for k, v := range params {
		allParams[k] = v
	}

	// Sort parameter keys
	var keys []string
	for k := range allParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build parameter string
	var paramPairs []string
	for _, k := range keys {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", percentEncode(k), percentEncode(allParams[k])))
	}
	paramString := strings.Join(paramPairs, "&")

	// Build signature base string
	baseString := fmt.Sprintf("%s&%s&%s",
		strings.ToUpper(method),
		percentEncode(urlStr),
		percentEncode(paramString),
	)

	// Build signing key
	signingKey := fmt.Sprintf("%s&%s",
		percentEncode(c.consumerSecret),
		percentEncode(c.accessSecret),
	)

	// Generate HMAC-SHA1 signature
	h := hmac.New(sha1.New, []byte(signingKey))
	h.Write([]byte(baseString))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

func (c *oauthClient) buildAuthHeader(method, urlStr string) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := c.generateNonce()

	oauthParams := map[string]string{
		"oauth_consumer_key":     c.consumerKey,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_token":            c.accessToken,
		"oauth_version":          "1.0",
	}

	signature := c.generateSignature(method, urlStr, oauthParams)
	oauthParams["oauth_signature"] = signature

	// Build Authorization header
	var headerParts []string
	for k, v := range oauthParams {
		headerParts = append(headerParts, fmt.Sprintf(`%s="%s"`, k, percentEncode(v)))
	}
	sort.Strings(headerParts)

	return "OAuth " + strings.Join(headerParts, ", ")
}

func (c *oauthClient) doRequest(method, urlStr string, body any) ([]byte, error) {
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

	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.buildAuthHeader(method, urlStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
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

func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	// OAuth requires some characters to not be encoded
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	return encoded
}

func newPostCmd() *cobra.Command {
	var replyTo string

	cmd := &cobra.Command{
		Use:   "post [message]",
		Short: "Post a tweet (free tier: 1,500/month)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newOAuthClient()
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
			client, err := newOAuthClient()
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
			client, err := newOAuthClient()
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
