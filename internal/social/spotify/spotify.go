package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const (
	apiBaseURL = "https://api.spotify.com/v1"
	tokenURL   = "https://accounts.spotify.com/api/token"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Token cache
var (
	cachedToken string
	tokenExpiry time.Time
	tokenMu     sync.Mutex
)

// TrackResult holds search result for a track
type TrackResult struct {
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	Duration   string `json:"duration"`
	Popularity int    `json:"popularity"`
	PreviewURL string `json:"preview_url,omitempty"`
	SpotifyURL string `json:"spotify_url"`
}

// ArtistResult holds search result for an artist
type ArtistResult struct {
	Name       string   `json:"name"`
	Genres     []string `json:"genres,omitempty"`
	Followers  int      `json:"followers"`
	Popularity int      `json:"popularity"`
	SpotifyURL string   `json:"spotify_url"`
}

// AlbumResult holds search result for an album
type AlbumResult struct {
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	ReleaseDate string `json:"release_date"`
	TotalTracks int    `json:"total_tracks"`
	SpotifyURL  string `json:"spotify_url"`
}

// Track holds detailed track info
type Track struct {
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	Duration   string `json:"duration"`
	Popularity int    `json:"popularity"`
	PreviewURL string `json:"preview_url,omitempty"`
	SpotifyURL string `json:"spotify_url"`
	ISRC       string `json:"isrc,omitempty"`
	Explicit   bool   `json:"explicit"`
}

// Artist holds detailed artist info
type Artist struct {
	Name       string   `json:"name"`
	Genres     []string `json:"genres,omitempty"`
	Followers  int      `json:"followers"`
	Popularity int      `json:"popularity"`
	SpotifyURL string   `json:"spotify_url"`
	TopTracks  []string `json:"top_tracks,omitempty"`
}

// Album holds detailed album info
type Album struct {
	Name        string       `json:"name"`
	Artist      string       `json:"artist"`
	ReleaseDate string       `json:"release_date"`
	TotalTracks int          `json:"total_tracks"`
	Tracks      []AlbumTrack `json:"tracks,omitempty"`
	SpotifyURL  string       `json:"spotify_url"`
	Label       string       `json:"label,omitempty"`
	Genres      []string     `json:"genres,omitempty"`
}

// AlbumTrack holds a track within an album
type AlbumTrack struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Duration string `json:"duration"`
}

// NewCmd returns the Spotify command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "spotify",
		Aliases: []string{"sp", "music"},
		Short:   "Spotify commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newTrackCmd())
	cmd.AddCommand(newArtistCmd())
	cmd.AddCommand(newAlbumCmd())

	return cmd
}

func getToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	// Return cached token if still valid (with 60s buffer)
	if cachedToken != "" && time.Now().Add(60*time.Second).Before(tokenExpiry) {
		return cachedToken, nil
	}

	clientID, err := config.Get("spotify_client_id")
	if err != nil || clientID == "" {
		return "", output.PrintError("setup_required", "Spotify client ID not configured", map[string]any{
			"missing":   []string{"spotify_client_id", "spotify_client_secret"},
			"setup_cmd": "pocket config set spotify_client_id <your-id>",
			"hint":      "Create an app at https://developer.spotify.com/dashboard",
		})
	}

	clientSecret, err := config.Get("spotify_client_secret")
	if err != nil || clientSecret == "" {
		return "", output.PrintError("setup_required", "Spotify client secret not configured", map[string]any{
			"missing":   []string{"spotify_client_secret"},
			"setup_cmd": "pocket config set spotify_client_secret <your-secret>",
			"hint":      "Create an app at https://developer.spotify.com/dashboard",
		})
	}

	// Request token using Client Credentials flow
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", output.PrintError("auth_failed", fmt.Sprintf("Failed to create auth request: %s", err.Error()), nil)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", output.PrintError("auth_failed", fmt.Sprintf("Auth request failed: %s", err.Error()), nil)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", output.PrintError("auth_failed", fmt.Sprintf("Failed to read auth response: %s", err.Error()), nil)
	}

	if resp.StatusCode != 200 {
		return "", output.PrintError("auth_failed", fmt.Sprintf("Auth failed with HTTP %d", resp.StatusCode), map[string]any{
			"hint": "Check your spotify_client_id and spotify_client_secret",
		})
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", output.PrintError("auth_failed", fmt.Sprintf("Failed to parse auth response: %s", err.Error()), nil)
	}

	cachedToken = tokenResp.AccessToken
	tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return cachedToken, nil
}

func newSearchCmd() *cobra.Command {
	var searchType string
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for tracks, artists, albums, or playlists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			query := args[0]
			params := url.Values{
				"q":     {query},
				"type":  {searchType},
				"limit": {fmt.Sprintf("%d", limit)},
			}

			reqURL := fmt.Sprintf("%s/search?%s", apiBaseURL, params.Encode())
			data, err := doRequest(token, reqURL)
			if err != nil {
				return err
			}

			switch searchType {
			case "track":
				return parseTrackSearch(data)
			case "artist":
				return parseArtistSearch(data)
			case "album":
				return parseAlbumSearch(data)
			case "playlist":
				return parsePlaylistSearch(data)
			default:
				return parseTrackSearch(data)
			}
		},
	}

	cmd.Flags().StringVarP(&searchType, "type", "t", "track", "Type: track, artist, album, playlist")
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of results")

	return cmd
}

func newTrackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track [id]",
		Short: "Get track details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			trackID := extractID(args[0])
			reqURL := fmt.Sprintf("%s/tracks/%s", apiBaseURL, url.PathEscape(trackID))
			data, err := doRequest(token, reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name string `json:"name"`
				} `json:"album"`
				DurationMs   int    `json:"duration_ms"`
				Popularity   int    `json:"popularity"`
				PreviewURL   string `json:"preview_url"`
				ExternalURLs struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
				ExternalIDs struct {
					ISRC string `json:"isrc"`
				} `json:"external_ids"`
				Explicit bool `json:"explicit"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
			}

			artistNames := make([]string, 0, len(resp.Artists))
			for _, a := range resp.Artists {
				artistNames = append(artistNames, a.Name)
			}

			result := Track{
				Name:       resp.Name,
				Artist:     strings.Join(artistNames, ", "),
				Album:      resp.Album.Name,
				Duration:   formatDuration(resp.DurationMs),
				Popularity: resp.Popularity,
				PreviewURL: resp.PreviewURL,
				SpotifyURL: resp.ExternalURLs.Spotify,
				ISRC:       resp.ExternalIDs.ISRC,
				Explicit:   resp.Explicit,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newArtistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artist [id]",
		Short: "Get artist details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			artistID := extractID(args[0])

			// Get artist info
			reqURL := fmt.Sprintf("%s/artists/%s", apiBaseURL, url.PathEscape(artistID))
			data, err := doRequest(token, reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Name      string   `json:"name"`
				Genres    []string `json:"genres"`
				Followers struct {
					Total int `json:"total"`
				} `json:"followers"`
				Popularity   int `json:"popularity"`
				ExternalURLs struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
			}

			// Get top tracks
			topTracksURL := fmt.Sprintf("%s/artists/%s/top-tracks?market=US", apiBaseURL, url.PathEscape(artistID))
			topData, topErr := doRequest(token, topTracksURL)

			var topTrackNames []string
			if topErr == nil {
				var topResp struct {
					Tracks []struct {
						Name string `json:"name"`
					} `json:"tracks"`
				}
				if json.Unmarshal(topData, &topResp) == nil {
					for i, t := range topResp.Tracks {
						if i >= 5 {
							break
						}
						topTrackNames = append(topTrackNames, t.Name)
					}
				}
			}

			result := Artist{
				Name:       resp.Name,
				Genres:     resp.Genres,
				Followers:  resp.Followers.Total,
				Popularity: resp.Popularity,
				SpotifyURL: resp.ExternalURLs.Spotify,
				TopTracks:  topTrackNames,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newAlbumCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "album [id]",
		Short: "Get album details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			albumID := extractID(args[0])
			reqURL := fmt.Sprintf("%s/albums/%s", apiBaseURL, url.PathEscape(albumID))
			data, err := doRequest(token, reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				ReleaseDate string `json:"release_date"`
				TotalTracks int    `json:"total_tracks"`
				Tracks      struct {
					Items []struct {
						TrackNumber int    `json:"track_number"`
						Name        string `json:"name"`
						DurationMs  int    `json:"duration_ms"`
					} `json:"items"`
				} `json:"tracks"`
				ExternalURLs struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
				Label  string   `json:"label"`
				Genres []string `json:"genres"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
			}

			artistNames := make([]string, 0, len(resp.Artists))
			for _, a := range resp.Artists {
				artistNames = append(artistNames, a.Name)
			}

			tracks := make([]AlbumTrack, 0, len(resp.Tracks.Items))
			for _, t := range resp.Tracks.Items {
				tracks = append(tracks, AlbumTrack{
					Number:   t.TrackNumber,
					Name:     t.Name,
					Duration: formatDuration(t.DurationMs),
				})
			}

			result := Album{
				Name:        resp.Name,
				Artist:      strings.Join(artistNames, ", "),
				ReleaseDate: resp.ReleaseDate,
				TotalTracks: resp.TotalTracks,
				Tracks:      tracks,
				SpotifyURL:  resp.ExternalURLs.Spotify,
				Label:       resp.Label,
				Genres:      resp.Genres,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func doRequest(token, reqURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("request_failed", fmt.Sprintf("Failed to create request: %s", err.Error()), nil)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, output.PrintError("request_failed", fmt.Sprintf("Request failed: %s", err.Error()), nil)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, output.PrintError("read_failed", fmt.Sprintf("Failed to read response: %s", err.Error()), nil)
	}

	if resp.StatusCode == 401 {
		// Token expired, clear cache
		tokenMu.Lock()
		cachedToken = ""
		tokenMu.Unlock()
		return nil, output.PrintError("auth_expired", "Access token expired. Please retry the command.", nil)
	}

	if resp.StatusCode == 404 {
		return nil, output.PrintError("not_found", "Resource not found", nil)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, output.PrintError("api_error", errResp.Error.Message, nil)
		}
		return nil, output.PrintError("api_error", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return body, nil
}

func parseTrackSearch(data []byte) error {
	var resp struct {
		Tracks struct {
			Items []struct {
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name string `json:"name"`
				} `json:"album"`
				DurationMs   int    `json:"duration_ms"`
				Popularity   int    `json:"popularity"`
				PreviewURL   string `json:"preview_url"`
				ExternalURLs struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
	}

	results := make([]TrackResult, 0, len(resp.Tracks.Items))
	for _, item := range resp.Tracks.Items {
		artistNames := make([]string, 0, len(item.Artists))
		for _, a := range item.Artists {
			artistNames = append(artistNames, a.Name)
		}
		results = append(results, TrackResult{
			Name:       item.Name,
			Artist:     strings.Join(artistNames, ", "),
			Album:      item.Album.Name,
			Duration:   formatDuration(item.DurationMs),
			Popularity: item.Popularity,
			PreviewURL: item.PreviewURL,
			SpotifyURL: item.ExternalURLs.Spotify,
		})
	}

	return output.Print(results)
}

func parseArtistSearch(data []byte) error {
	var resp struct {
		Artists struct {
			Items []struct {
				Name      string   `json:"name"`
				Genres    []string `json:"genres"`
				Followers struct {
					Total int `json:"total"`
				} `json:"followers"`
				Popularity   int `json:"popularity"`
				ExternalURLs struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
			} `json:"items"`
		} `json:"artists"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
	}

	results := make([]ArtistResult, 0, len(resp.Artists.Items))
	for _, item := range resp.Artists.Items {
		results = append(results, ArtistResult{
			Name:       item.Name,
			Genres:     item.Genres,
			Followers:  item.Followers.Total,
			Popularity: item.Popularity,
			SpotifyURL: item.ExternalURLs.Spotify,
		})
	}

	return output.Print(results)
}

func parseAlbumSearch(data []byte) error {
	var resp struct {
		Albums struct {
			Items []struct {
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				ReleaseDate  string `json:"release_date"`
				TotalTracks  int    `json:"total_tracks"`
				ExternalURLs struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
			} `json:"items"`
		} `json:"albums"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
	}

	results := make([]AlbumResult, 0, len(resp.Albums.Items))
	for _, item := range resp.Albums.Items {
		artistNames := make([]string, 0, len(item.Artists))
		for _, a := range item.Artists {
			artistNames = append(artistNames, a.Name)
		}
		results = append(results, AlbumResult{
			Name:        item.Name,
			Artist:      strings.Join(artistNames, ", "),
			ReleaseDate: item.ReleaseDate,
			TotalTracks: item.TotalTracks,
			SpotifyURL:  item.ExternalURLs.Spotify,
		})
	}

	return output.Print(results)
}

func parsePlaylistSearch(data []byte) error {
	var resp struct {
		Playlists struct {
			Items []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Owner       struct {
					DisplayName string `json:"display_name"`
				} `json:"owner"`
				Tracks struct {
					Total int `json:"total"`
				} `json:"tracks"`
				ExternalURLs struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
			} `json:"items"`
		} `json:"playlists"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
	}

	type PlaylistResult struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Owner       string `json:"owner"`
		TrackCount  int    `json:"track_count"`
		SpotifyURL  string `json:"spotify_url"`
	}

	results := make([]PlaylistResult, 0, len(resp.Playlists.Items))
	for _, item := range resp.Playlists.Items {
		results = append(results, PlaylistResult{
			Name:        item.Name,
			Description: item.Description,
			Owner:       item.Owner.DisplayName,
			TrackCount:  item.Tracks.Total,
			SpotifyURL:  item.ExternalURLs.Spotify,
		})
	}

	return output.Print(results)
}

func formatDuration(ms int) string {
	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func extractID(input string) string {
	// Handle Spotify URLs like https://open.spotify.com/track/6rqhFgbbKwnb9MLmUQDhG6
	if strings.Contains(input, "open.spotify.com") {
		parts := strings.Split(input, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			// Remove query params
			if idx := strings.Index(lastPart, "?"); idx >= 0 {
				lastPart = lastPart[:idx]
			}
			return lastPart
		}
	}
	// Handle spotify:track:id URIs
	if strings.HasPrefix(input, "spotify:") {
		parts := strings.Split(input, ":")
		if len(parts) >= 3 {
			return parts[2]
		}
	}
	return input
}
