package calendar

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

const (
	tokenURL    = "https://oauth2.googleapis.com/token"
	calendarAPI = "https://www.googleapis.com/calendar/v3"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "calendar",
		Aliases: []string{"cal", "gcal"},
		Short:   "Calendar commands (Google Calendar)",
		Long:    "Google Calendar integration. Requires OAuth2 setup with client credentials and refresh token.",
	}

	cmd.AddCommand(newEventsCmd())
	cmd.AddCommand(newTodayCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newCalendarsCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

type calendarClient struct {
	accessToken string
	httpClient  *http.Client
}

func newCalendarClient() (*calendarClient, error) {
	clientID, err := config.MustGet("google_client_id")
	if err != nil {
		return nil, fmt.Errorf("google_client_id not configured. Run 'pocket setup show calendar' for setup instructions")
	}

	clientSecret, err := config.MustGet("google_client_secret")
	if err != nil {
		return nil, fmt.Errorf("google_client_secret not configured. Run 'pocket setup show calendar' for setup instructions")
	}

	refreshToken, err := config.MustGet("google_refresh_token")
	if err != nil {
		return nil, fmt.Errorf("google_refresh_token not configured. Run 'pocket setup show calendar' for setup instructions")
	}

	// Exchange refresh token for access token
	accessToken, err := getAccessToken(clientID, clientSecret, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	return &calendarClient{
		accessToken: accessToken,
		httpClient:  &http.Client{},
	}, nil
}

func getAccessToken(clientID, clientSecret, refreshToken string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.ErrorDescription != "" {
			return "", fmt.Errorf("OAuth error: %s", errResp.ErrorDescription)
		}
		return "", fmt.Errorf("OAuth error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func (c *calendarClient) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
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

	req, err := http.NewRequestWithContext(ctx, method, calendarAPI+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
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
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("Google Calendar API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("Google Calendar API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type calendarEvent struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Start       struct {
		DateTime string `json:"dateTime,omitempty"`
		Date     string `json:"date,omitempty"`
		TimeZone string `json:"timeZone,omitempty"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime,omitempty"`
		Date     string `json:"date,omitempty"`
		TimeZone string `json:"timeZone,omitempty"`
	} `json:"end"`
	Status    string `json:"status"`
	HTMLURL   string `json:"htmlLink"`
	Organizer struct {
		Email string `json:"email"`
		Self  bool   `json:"self"`
	} `json:"organizer,omitempty"`
	Attendees []struct {
		Email          string `json:"email"`
		ResponseStatus string `json:"responseStatus"`
	} `json:"attendees,omitempty"`
}

func formatEvents(events []calendarEvent) []map[string]any {
	result := make([]map[string]any, len(events))
	for i, e := range events {
		startTime := e.Start.DateTime
		if startTime == "" {
			startTime = e.Start.Date
		}
		endTime := e.End.DateTime
		if endTime == "" {
			endTime = e.End.Date
		}

		item := map[string]any{
			"id":     e.ID,
			"title":  e.Summary,
			"start":  startTime,
			"end":    endTime,
			"status": e.Status,
			"url":    e.HTMLURL,
		}

		if e.Description != "" {
			item["description"] = e.Description
		}
		if e.Location != "" {
			item["location"] = e.Location
		}
		if len(e.Attendees) > 0 {
			attendees := make([]string, len(e.Attendees))
			for j, a := range e.Attendees {
				attendees[j] = a.Email
			}
			item["attendees"] = attendees
		}

		result[i] = item
	}
	return result
}

func newEventsCmd() *cobra.Command {
	var days int
	var limit int
	var calendarID string

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List upcoming events",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newCalendarClient()
			if err != nil {
				return err
			}

			now := time.Now()
			timeMin := now.Format(time.RFC3339)
			timeMax := now.AddDate(0, 0, days).Format(time.RFC3339)

			endpoint := fmt.Sprintf("/calendars/%s/events?timeMin=%s&timeMax=%s&maxResults=%d&singleEvents=true&orderBy=startTime",
				url.PathEscape(calendarID), url.QueryEscape(timeMin), url.QueryEscape(timeMax), limit)

			body, err := client.doRequest("GET", endpoint, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Items []calendarEvent `json:"items"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"calendar": calendarID,
				"days":     days,
				"count":    len(result.Items),
				"events":   formatEvents(result.Items),
			})
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 7, "Days to look ahead")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum events")
	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID (default: primary)")

	return cmd
}

func newTodayCmd() *cobra.Command {
	var calendarID string

	cmd := &cobra.Command{
		Use:   "today",
		Short: "List today's events",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newCalendarClient()
			if err != nil {
				return err
			}

			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endOfDay := startOfDay.AddDate(0, 0, 1)

			timeMin := startOfDay.Format(time.RFC3339)
			timeMax := endOfDay.Format(time.RFC3339)

			endpoint := fmt.Sprintf("/calendars/%s/events?timeMin=%s&timeMax=%s&singleEvents=true&orderBy=startTime",
				url.PathEscape(calendarID), url.QueryEscape(timeMin), url.QueryEscape(timeMax))

			body, err := client.doRequest("GET", endpoint, nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Items []calendarEvent `json:"items"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"date":     now.Format("2006-01-02"),
				"calendar": calendarID,
				"count":    len(result.Items),
				"events":   formatEvents(result.Items),
			})
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID (default: primary)")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var title string
	var start string
	var end string
	var description string
	var location string
	var calendarID string
	var allDay bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an event",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newCalendarClient()
			if err != nil {
				return err
			}

			event := map[string]any{
				"summary": title,
			}

			if description != "" {
				event["description"] = description
			}
			if location != "" {
				event["location"] = location
			}

			if allDay {
				// All-day events use date format (YYYY-MM-DD)
				event["start"] = map[string]string{"date": start}
				event["end"] = map[string]string{"date": end}
			} else {
				// Timed events use RFC3339 format
				event["start"] = map[string]string{"dateTime": start}
				event["end"] = map[string]string{"dateTime": end}
			}

			endpoint := fmt.Sprintf("/calendars/%s/events", url.PathEscape(calendarID))

			body, err := client.doRequest("POST", endpoint, event)
			if err != nil {
				return output.PrintError("create_failed", err.Error(), nil)
			}

			var result calendarEvent
			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			startTime := result.Start.DateTime
			if startTime == "" {
				startTime = result.Start.Date
			}

			return output.Print(map[string]any{
				"id":    result.ID,
				"title": result.Summary,
				"start": startTime,
				"url":   result.HTMLURL,
			})
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Event title (required)")
	cmd.Flags().StringVar(&start, "start", "", "Start time - RFC3339 format for timed events (e.g., 2026-02-07T09:00:00-08:00) or YYYY-MM-DD for all-day")
	cmd.Flags().StringVar(&end, "end", "", "End time - same format as start")
	cmd.Flags().StringVar(&description, "desc", "", "Description")
	cmd.Flags().StringVar(&location, "location", "", "Location")
	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "Create an all-day event (use YYYY-MM-DD format)")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("start")
	cmd.MarkFlagRequired("end")

	return cmd
}

func newCalendarsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendars",
		Short: "List available calendars",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newCalendarClient()
			if err != nil {
				return err
			}

			body, err := client.doRequest("GET", "/users/me/calendarList", nil)
			if err != nil {
				return output.PrintError("request_failed", err.Error(), nil)
			}

			var result struct {
				Items []struct {
					ID              string `json:"id"`
					Summary         string `json:"summary"`
					Description     string `json:"description"`
					Primary         bool   `json:"primary"`
					AccessRole      string `json:"accessRole"`
					BackgroundColor string `json:"backgroundColor"`
				} `json:"items"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return output.PrintError("parse_error", err.Error(), nil)
			}

			calendars := make([]map[string]any, len(result.Items))
			for i, c := range result.Items {
				calendars[i] = map[string]any{
					"id":          c.ID,
					"name":        c.Summary,
					"description": c.Description,
					"primary":     c.Primary,
					"access":      c.AccessRole,
					"color":       c.BackgroundColor,
				}
			}

			return output.Print(map[string]any{
				"count":     len(calendars),
				"calendars": calendars,
			})
		},
	}

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var calendarID string

	cmd := &cobra.Command{
		Use:   "delete [event-id]",
		Short: "Delete an event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newCalendarClient()
			if err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/calendars/%s/events/%s",
				url.PathEscape(calendarID), url.PathEscape(args[0]))

			_, err = client.doRequest("DELETE", endpoint, nil)
			if err != nil {
				return output.PrintError("delete_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"deleted":  true,
				"event_id": args[0],
			})
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID")

	return cmd
}
