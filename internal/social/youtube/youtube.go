package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://www.googleapis.com/youtube/v3"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Video is LLM-friendly video output
type Video struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Channel     string `json:"channel"`
	ChannelID   string `json:"channel_id"`
	Published   string `json:"published"`
	Duration    string `json:"duration,omitempty"`
	Views       int64  `json:"views,omitempty"`
	Likes       int64  `json:"likes,omitempty"`
	Comments    int64  `json:"comments,omitempty"`
	Description string `json:"desc,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	URL         string `json:"url"`
}

// Channel is LLM-friendly channel output
type Channel struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"desc,omitempty"`
	CustomURL   string `json:"custom_url,omitempty"`
	Published   string `json:"published"`
	Subscribers int64  `json:"subscribers"`
	Videos      int64  `json:"videos"`
	Views       int64  `json:"views"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	URL         string `json:"url"`
}

// Comment is LLM-friendly comment output
type Comment struct {
	Author    string `json:"author"`
	Text      string `json:"text"`
	Likes     int64  `json:"likes"`
	Published string `json:"published"`
	Replies   int64  `json:"replies,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "youtube",
		Aliases: []string{"yt"},
		Short:   "YouTube commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newVideoCmd())
	cmd.AddCommand(newChannelCmd())
	cmd.AddCommand(newVideosCmd())
	cmd.AddCommand(newCommentsCmd())
	cmd.AddCommand(newTrendingCmd())

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int
	var order string
	var publishedAfter string
	var duration string
	var channelID string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for videos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			query := args[0]
			params := url.Values{
				"part":       {"snippet"},
				"q":          {query},
				"type":       {"video"},
				"maxResults": {fmt.Sprintf("%d", limit)},
				"order":      {order},
				"key":        {apiKey},
			}

			if publishedAfter != "" {
				// Parse relative time like "7d", "1m", "1y"
				t, err := parseRelativeTime(publishedAfter)
				if err == nil {
					params.Set("publishedAfter", t.Format(time.RFC3339))
				}
			}

			if duration != "" {
				// short (<4min), medium (4-20min), long (>20min)
				params.Set("videoDuration", duration)
			}

			if channelID != "" {
				params.Set("channelId", channelID)
			}

			reqURL := fmt.Sprintf("%s/search?%s", baseURL, params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Items []struct {
					ID struct {
						VideoID string `json:"videoId"`
					} `json:"id"`
					Snippet struct {
						Title        string `json:"title"`
						ChannelTitle string `json:"channelTitle"`
						ChannelID    string `json:"channelId"`
						PublishedAt  string `json:"publishedAt"`
						Description  string `json:"description"`
						Thumbnails   struct {
							Medium struct {
								URL string `json:"url"`
							} `json:"medium"`
						} `json:"thumbnails"`
					} `json:"snippet"`
				} `json:"items"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var videos []Video
			for _, item := range resp.Items {
				videos = append(videos, Video{
					ID:          item.ID.VideoID,
					Title:       item.Snippet.Title,
					Channel:     item.Snippet.ChannelTitle,
					ChannelID:   item.Snippet.ChannelID,
					Published:   formatTime(item.Snippet.PublishedAt),
					Description: truncate(item.Snippet.Description, 150),
					Thumbnail:   item.Snippet.Thumbnails.Medium.URL,
					URL:         "https://youtube.com/watch?v=" + item.ID.VideoID,
				})
			}

			return output.Print(videos)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Max results (1-50)")
	cmd.Flags().StringVarP(&order, "order", "s", "relevance", "Sort: relevance, date, viewCount, rating")
	cmd.Flags().StringVarP(&publishedAfter, "after", "a", "", "Published after: 1d, 7d, 1m, 1y")
	cmd.Flags().StringVarP(&duration, "duration", "d", "", "Duration: short (<4m), medium (4-20m), long (>20m)")
	cmd.Flags().StringVarP(&channelID, "channel", "c", "", "Filter by channel ID")

	return cmd
}

func newVideoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "video [id]",
		Short: "Get video details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			videoID := extractVideoID(args[0])

			params := url.Values{
				"part": {"snippet,statistics,contentDetails"},
				"id":   {videoID},
				"key":  {apiKey},
			}

			reqURL := fmt.Sprintf("%s/videos?%s", baseURL, params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Items []struct {
					ID      string `json:"id"`
					Snippet struct {
						Title        string `json:"title"`
						ChannelTitle string `json:"channelTitle"`
						ChannelID    string `json:"channelId"`
						PublishedAt  string `json:"publishedAt"`
						Description  string `json:"description"`
						Thumbnails   struct {
							Medium struct {
								URL string `json:"url"`
							} `json:"medium"`
						} `json:"thumbnails"`
					} `json:"snippet"`
					Statistics struct {
						ViewCount    string `json:"viewCount"`
						LikeCount    string `json:"likeCount"`
						CommentCount string `json:"commentCount"`
					} `json:"statistics"`
					ContentDetails struct {
						Duration string `json:"duration"`
					} `json:"contentDetails"`
				} `json:"items"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if len(resp.Items) == 0 {
				return output.PrintError("not_found", "Video not found: "+videoID, nil)
			}

			item := resp.Items[0]
			video := Video{
				ID:          item.ID,
				Title:       item.Snippet.Title,
				Channel:     item.Snippet.ChannelTitle,
				ChannelID:   item.Snippet.ChannelID,
				Published:   formatTime(item.Snippet.PublishedAt),
				Duration:    parseDuration(item.ContentDetails.Duration),
				Views:       parseInt(item.Statistics.ViewCount),
				Likes:       parseInt(item.Statistics.LikeCount),
				Comments:    parseInt(item.Statistics.CommentCount),
				Description: truncate(item.Snippet.Description, 500),
				Thumbnail:   item.Snippet.Thumbnails.Medium.URL,
				URL:         "https://youtube.com/watch?v=" + item.ID,
			}

			return output.Print(video)
		},
	}

	return cmd
}

func newChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel [id-or-username]",
		Short: "Get channel info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			channelArg := args[0]
			params := url.Values{
				"part": {"snippet,statistics"},
				"key":  {apiKey},
			}

			// Determine if it's a channel ID or username/handle
			if strings.HasPrefix(channelArg, "UC") && len(channelArg) == 24 {
				params.Set("id", channelArg)
			} else if strings.HasPrefix(channelArg, "@") {
				params.Set("forHandle", channelArg)
			} else {
				params.Set("forHandle", "@"+channelArg)
			}

			reqURL := fmt.Sprintf("%s/channels?%s", baseURL, params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Items []struct {
					ID      string `json:"id"`
					Snippet struct {
						Title       string `json:"title"`
						Description string `json:"description"`
						CustomURL   string `json:"customUrl"`
						PublishedAt string `json:"publishedAt"`
						Thumbnails  struct {
							Medium struct {
								URL string `json:"url"`
							} `json:"medium"`
						} `json:"thumbnails"`
					} `json:"snippet"`
					Statistics struct {
						SubscriberCount string `json:"subscriberCount"`
						VideoCount      string `json:"videoCount"`
						ViewCount       string `json:"viewCount"`
					} `json:"statistics"`
				} `json:"items"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if len(resp.Items) == 0 {
				return output.PrintError("not_found", "Channel not found: "+channelArg, nil)
			}

			item := resp.Items[0]
			channel := Channel{
				ID:          item.ID,
				Title:       item.Snippet.Title,
				Description: truncate(item.Snippet.Description, 300),
				CustomURL:   item.Snippet.CustomURL,
				Published:   formatTime(item.Snippet.PublishedAt),
				Subscribers: parseInt(item.Statistics.SubscriberCount),
				Videos:      parseInt(item.Statistics.VideoCount),
				Views:       parseInt(item.Statistics.ViewCount),
				Thumbnail:   item.Snippet.Thumbnails.Medium.URL,
				URL:         "https://youtube.com/channel/" + item.ID,
			}

			return output.Print(channel)
		},
	}

	return cmd
}

func newVideosCmd() *cobra.Command {
	var limit int
	var order string

	cmd := &cobra.Command{
		Use:   "videos [channel-id]",
		Short: "List videos from a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			channelID := args[0]

			// First get the uploads playlist ID
			params := url.Values{
				"part": {"contentDetails"},
				"id":   {channelID},
				"key":  {apiKey},
			}

			reqURL := fmt.Sprintf("%s/channels?%s", baseURL, params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var channelResp struct {
				Items []struct {
					ContentDetails struct {
						RelatedPlaylists struct {
							Uploads string `json:"uploads"`
						} `json:"relatedPlaylists"`
					} `json:"contentDetails"`
				} `json:"items"`
			}

			if err := json.Unmarshal(data, &channelResp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			if len(channelResp.Items) == 0 {
				return output.PrintError("not_found", "Channel not found: "+channelID, nil)
			}

			uploadsPlaylist := channelResp.Items[0].ContentDetails.RelatedPlaylists.Uploads

			// Now get videos from the uploads playlist
			params = url.Values{
				"part":       {"snippet"},
				"playlistId": {uploadsPlaylist},
				"maxResults": {fmt.Sprintf("%d", limit)},
				"key":        {apiKey},
			}

			reqURL = fmt.Sprintf("%s/playlistItems?%s", baseURL, params.Encode())
			data, err = doRequest(reqURL)
			if err != nil {
				return err
			}

			var playlistResp struct {
				Items []struct {
					Snippet struct {
						Title        string `json:"title"`
						ChannelTitle string `json:"channelTitle"`
						ChannelID    string `json:"channelId"`
						PublishedAt  string `json:"publishedAt"`
						Description  string `json:"description"`
						ResourceID   struct {
							VideoID string `json:"videoId"`
						} `json:"resourceId"`
						Thumbnails struct {
							Medium struct {
								URL string `json:"url"`
							} `json:"medium"`
						} `json:"thumbnails"`
					} `json:"snippet"`
				} `json:"items"`
			}

			if err := json.Unmarshal(data, &playlistResp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			// Get video IDs to fetch statistics
			var videoIDs []string
			for _, item := range playlistResp.Items {
				videoIDs = append(videoIDs, item.Snippet.ResourceID.VideoID)
			}

			// Fetch video statistics
			statsMap := make(map[string]struct {
				Views    int64
				Likes    int64
				Comments int64
				Duration string
			})

			if len(videoIDs) > 0 {
				params = url.Values{
					"part": {"statistics,contentDetails"},
					"id":   {strings.Join(videoIDs, ",")},
					"key":  {apiKey},
				}

				reqURL = fmt.Sprintf("%s/videos?%s", baseURL, params.Encode())
				data, err = doRequest(reqURL)
				if err == nil {
					var statsResp struct {
						Items []struct {
							ID         string `json:"id"`
							Statistics struct {
								ViewCount    string `json:"viewCount"`
								LikeCount    string `json:"likeCount"`
								CommentCount string `json:"commentCount"`
							} `json:"statistics"`
							ContentDetails struct {
								Duration string `json:"duration"`
							} `json:"contentDetails"`
						} `json:"items"`
					}
					if json.Unmarshal(data, &statsResp) == nil {
						for _, item := range statsResp.Items {
							statsMap[item.ID] = struct {
								Views    int64
								Likes    int64
								Comments int64
								Duration string
							}{
								Views:    parseInt(item.Statistics.ViewCount),
								Likes:    parseInt(item.Statistics.LikeCount),
								Comments: parseInt(item.Statistics.CommentCount),
								Duration: parseDuration(item.ContentDetails.Duration),
							}
						}
					}
				}
			}

			var videos []Video
			for _, item := range playlistResp.Items {
				vid := item.Snippet.ResourceID.VideoID
				stats := statsMap[vid]
				videos = append(videos, Video{
					ID:          vid,
					Title:       item.Snippet.Title,
					Channel:     item.Snippet.ChannelTitle,
					ChannelID:   item.Snippet.ChannelID,
					Published:   formatTime(item.Snippet.PublishedAt),
					Duration:    stats.Duration,
					Views:       stats.Views,
					Likes:       stats.Likes,
					Comments:    stats.Comments,
					Description: truncate(item.Snippet.Description, 150),
					Thumbnail:   item.Snippet.Thumbnails.Medium.URL,
					URL:         "https://youtube.com/watch?v=" + vid,
				})
			}

			return output.Print(videos)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Max results (1-50)")
	cmd.Flags().StringVarP(&order, "order", "s", "date", "Sort: date, viewCount")

	return cmd
}

func newCommentsCmd() *cobra.Command {
	var limit int
	var order string

	cmd := &cobra.Command{
		Use:   "comments [video-id]",
		Short: "Get video comments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			videoID := extractVideoID(args[0])

			params := url.Values{
				"part":       {"snippet"},
				"videoId":    {videoID},
				"maxResults": {fmt.Sprintf("%d", limit)},
				"order":      {order},
				"key":        {apiKey},
			}

			reqURL := fmt.Sprintf("%s/commentThreads?%s", baseURL, params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Items []struct {
					Snippet struct {
						TopLevelComment struct {
							Snippet struct {
								AuthorDisplayName string `json:"authorDisplayName"`
								TextDisplay       string `json:"textDisplay"`
								LikeCount         int64  `json:"likeCount"`
								PublishedAt       string `json:"publishedAt"`
							} `json:"snippet"`
						} `json:"topLevelComment"`
						TotalReplyCount int64 `json:"totalReplyCount"`
					} `json:"snippet"`
				} `json:"items"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var comments []Comment
			for _, item := range resp.Items {
				c := item.Snippet.TopLevelComment.Snippet
				comments = append(comments, Comment{
					Author:    c.AuthorDisplayName,
					Text:      cleanHTML(c.TextDisplay),
					Likes:     c.LikeCount,
					Published: formatTime(c.PublishedAt),
					Replies:   item.Snippet.TotalReplyCount,
				})
			}

			return output.Print(comments)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Max results (1-100)")
	cmd.Flags().StringVarP(&order, "order", "s", "relevance", "Sort: relevance, time")

	return cmd
}

func newTrendingCmd() *cobra.Command {
	var limit int
	var region string
	var category string

	cmd := &cobra.Command{
		Use:   "trending",
		Short: "Get trending videos",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			params := url.Values{
				"part":       {"snippet,statistics,contentDetails"},
				"chart":      {"mostPopular"},
				"regionCode": {region},
				"maxResults": {fmt.Sprintf("%d", limit)},
				"key":        {apiKey},
			}

			if category != "" {
				params.Set("videoCategoryId", category)
			}

			reqURL := fmt.Sprintf("%s/videos?%s", baseURL, params.Encode())
			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Items []struct {
					ID      string `json:"id"`
					Snippet struct {
						Title        string `json:"title"`
						ChannelTitle string `json:"channelTitle"`
						ChannelID    string `json:"channelId"`
						PublishedAt  string `json:"publishedAt"`
						Description  string `json:"description"`
						Thumbnails   struct {
							Medium struct {
								URL string `json:"url"`
							} `json:"medium"`
						} `json:"thumbnails"`
					} `json:"snippet"`
					Statistics struct {
						ViewCount    string `json:"viewCount"`
						LikeCount    string `json:"likeCount"`
						CommentCount string `json:"commentCount"`
					} `json:"statistics"`
					ContentDetails struct {
						Duration string `json:"duration"`
					} `json:"contentDetails"`
				} `json:"items"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			var videos []Video
			for _, item := range resp.Items {
				videos = append(videos, Video{
					ID:          item.ID,
					Title:       item.Snippet.Title,
					Channel:     item.Snippet.ChannelTitle,
					ChannelID:   item.Snippet.ChannelID,
					Published:   formatTime(item.Snippet.PublishedAt),
					Duration:    parseDuration(item.ContentDetails.Duration),
					Views:       parseInt(item.Statistics.ViewCount),
					Likes:       parseInt(item.Statistics.LikeCount),
					Comments:    parseInt(item.Statistics.CommentCount),
					Description: truncate(item.Snippet.Description, 150),
					Thumbnail:   item.Snippet.Thumbnails.Medium.URL,
					URL:         "https://youtube.com/watch?v=" + item.ID,
				})
			}

			return output.Print(videos)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Max results (1-50)")
	cmd.Flags().StringVarP(&region, "region", "r", "US", "Region code (US, GB, JP, etc.)")
	cmd.Flags().StringVarP(&category, "category", "c", "", "Category: 10=Music, 20=Gaming, 24=Entertainment, 25=News, 28=Science")

	return cmd
}

func getAPIKey() (string, error) {
	key, err := config.Get("youtube_api_key")
	if err != nil || key == "" {
		return "", output.PrintError("setup_required", "YouTube API key not configured", map[string]any{
			"missing":   []string{"youtube_api_key"},
			"setup_cmd": "pocket setup show youtube",
			"hint":      "Run 'pocket setup show youtube' for setup instructions",
		})
	}
	return key, nil
}

func doRequest(reqURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("request_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, output.PrintError("request_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, output.PrintError("quota_exceeded", "YouTube API quota exceeded or invalid API key", map[string]any{
			"hint": "Check your API key or wait for quota reset",
		})
	}

	if resp.StatusCode >= 400 {
		return nil, output.PrintError("request_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, output.PrintError("read_failed", err.Error(), nil)
	}

	return data, nil
}

func extractVideoID(input string) string {
	// Handle full URLs
	if strings.Contains(input, "youtube.com/watch") {
		if u, err := url.Parse(input); err == nil {
			return u.Query().Get("v")
		}
	}
	if strings.Contains(input, "youtu.be/") {
		parts := strings.Split(input, "youtu.be/")
		if len(parts) > 1 {
			return strings.Split(parts[1], "?")[0]
		}
	}
	return input
}

func parseInt(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func formatTime(isoTime string) string {
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours < 1 {
			return fmt.Sprintf("%dm ago", int(diff.Minutes()))
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	if diff < 30*24*time.Hour {
		return fmt.Sprintf("%dw ago", int(diff.Hours()/(24*7)))
	}
	if diff < 365*24*time.Hour {
		return fmt.Sprintf("%dmo ago", int(diff.Hours()/(24*30)))
	}
	return fmt.Sprintf("%dy ago", int(diff.Hours()/(24*365)))
}

func parseDuration(isoDuration string) string {
	// Parse ISO 8601 duration like PT4M13S
	isoDuration = strings.TrimPrefix(isoDuration, "PT")
	isoDuration = strings.ToLower(isoDuration)

	var hours, minutes, seconds int
	fmt.Sscanf(isoDuration, "%dh%dm%ds", &hours, &minutes, &seconds)

	if hours == 0 && minutes == 0 && seconds == 0 {
		// Try without hours
		fmt.Sscanf(isoDuration, "%dm%ds", &minutes, &seconds)
	}
	if minutes == 0 && seconds == 0 {
		// Try seconds only
		fmt.Sscanf(isoDuration, "%ds", &seconds)
	}

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func parseRelativeTime(rel string) (time.Time, error) {
	rel = strings.ToLower(rel)
	now := time.Now()

	var value int
	var unit string
	fmt.Sscanf(rel, "%d%s", &value, &unit)

	switch unit {
	case "d", "day", "days":
		return now.AddDate(0, 0, -value), nil
	case "w", "week", "weeks":
		return now.AddDate(0, 0, -value*7), nil
	case "m", "mo", "month", "months":
		return now.AddDate(0, -value, 0), nil
	case "y", "year", "years":
		return now.AddDate(-value, 0, 0), nil
	}

	return time.Time{}, fmt.Errorf("invalid relative time: %s", rel)
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func cleanHTML(s string) string {
	// Remove HTML tags
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")

	// Simple tag removal
	result := strings.Builder{}
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}
