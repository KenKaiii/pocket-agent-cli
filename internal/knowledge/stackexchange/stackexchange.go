package stackexchange

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://api.stackexchange.com/2.3"

var (
	client    = &http.Client{Timeout: 10 * time.Second}
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
)

// Question is LLM-friendly question output
type Question struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Score      int      `json:"score"`
	Answers    int      `json:"answers"`
	Accepted   bool     `json:"accepted"`
	Views      int      `json:"views"`
	Tags       []string `json:"tags,omitempty"`
	Author     string   `json:"author"`
	Age        string   `json:"age"`
	URL        string   `json:"url"`
	Body       string   `json:"body,omitempty"`
}

// Answer is LLM-friendly answer output
type Answer struct {
	ID       int    `json:"id"`
	Score    int    `json:"score"`
	Accepted bool   `json:"accepted"`
	Author   string `json:"author"`
	Age      string `json:"age"`
	Body     string `json:"body"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stackexchange",
		Aliases: []string{"se", "stackoverflow", "so"},
		Short:   "StackExchange commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newQuestionCmd())
	cmd.AddCommand(newAnswersCmd())

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int
	var site string
	var tagged string
	var sort string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search questions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			params := url.Values{
				"order":    {"desc"},
				"sort":     {sort},
				"intitle":  {query},
				"site":     {site},
				"pagesize": {fmt.Sprintf("%d", limit)},
				"filter":   {"withbody"},
			}

			if tagged != "" {
				params.Set("tagged", tagged)
			}

			var resp seResponse
			if err := seGet("/search/advanced", params, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			questions := make([]Question, 0, len(resp.Items))
			for _, item := range resp.Items {
				questions = append(questions, toQuestion(item, false))
			}

			return output.Print(questions)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of results")
	cmd.Flags().StringVarP(&site, "site", "s", "stackoverflow", "Site: stackoverflow, serverfault, superuser, etc.")
	cmd.Flags().StringVarP(&tagged, "tagged", "t", "", "Filter by tags (semicolon-separated)")
	cmd.Flags().StringVar(&sort, "sort", "relevance", "Sort: relevance, votes, creation, activity")

	return cmd
}

func newQuestionCmd() *cobra.Command {
	var site string

	cmd := &cobra.Command{
		Use:   "question [id]",
		Short: "Get question details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			params := url.Values{
				"site":   {site},
				"filter": {"withbody"},
			}

			var resp seResponse
			if err := seGet(fmt.Sprintf("/questions/%s", id), params, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			if len(resp.Items) == 0 {
				return output.PrintError("not_found", "Question not found", nil)
			}

			return output.Print(toQuestion(resp.Items[0], true))
		},
	}

	cmd.Flags().StringVarP(&site, "site", "s", "stackoverflow", "Site")

	return cmd
}

func newAnswersCmd() *cobra.Command {
	var site string
	var limit int
	var sort string

	cmd := &cobra.Command{
		Use:   "answers [question-id]",
		Short: "Get answers for a question",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			params := url.Values{
				"order":    {"desc"},
				"sort":     {sort},
				"site":     {site},
				"pagesize": {fmt.Sprintf("%d", limit)},
				"filter":   {"withbody"},
			}

			var resp seResponse
			if err := seGet(fmt.Sprintf("/questions/%s/answers", id), params, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			answers := make([]Answer, 0, len(resp.Items))
			for _, item := range resp.Items {
				answers = append(answers, toAnswer(item))
			}

			return output.Print(answers)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 5, "Number of answers")
	cmd.Flags().StringVarP(&site, "site", "s", "stackoverflow", "Site")
	cmd.Flags().StringVar(&sort, "sort", "votes", "Sort: votes, creation, activity")

	return cmd
}

type seResponse struct {
	Items []seItem `json:"items"`
}

type seItem struct {
	QuestionID       int      `json:"question_id"`
	AnswerID         int      `json:"answer_id"`
	Title            string   `json:"title"`
	Body             string   `json:"body"`
	Score            int      `json:"score"`
	AnswerCount      int      `json:"answer_count"`
	IsAccepted       bool     `json:"is_accepted"`
	AcceptedAnswerID int      `json:"accepted_answer_id"`
	ViewCount        int      `json:"view_count"`
	Tags             []string `json:"tags"`
	Link             string   `json:"link"`
	CreationDate     int64    `json:"creation_date"`
	Owner            struct {
		DisplayName string `json:"display_name"`
	} `json:"owner"`
}

func seGet(endpoint string, params url.Values, result any) error {
	reqURL := baseURL + endpoint + "?" + params.Encode()

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// StackExchange API returns gzip-compressed responses
	var reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	return json.NewDecoder(reader).Decode(result)
}

func toQuestion(item seItem, includeBody bool) Question {
	q := Question{
		ID:       item.QuestionID,
		Title:    item.Title,
		Score:    item.Score,
		Answers:  item.AnswerCount,
		Accepted: item.AcceptedAnswerID != 0,
		Views:    item.ViewCount,
		Tags:     item.Tags,
		Author:   item.Owner.DisplayName,
		Age:      timeAgo(time.Unix(item.CreationDate, 0)),
		URL:      item.Link,
	}

	if includeBody && item.Body != "" {
		q.Body = cleanHTML(item.Body)
		if len(q.Body) > 1000 {
			q.Body = q.Body[:997] + "..."
		}
	}

	return q
}

func toAnswer(item seItem) Answer {
	body := cleanHTML(item.Body)
	if len(body) > 1000 {
		body = body[:997] + "..."
	}

	return Answer{
		ID:       item.AnswerID,
		Score:    item.Score,
		Accepted: item.IsAccepted,
		Author:   item.Owner.DisplayName,
		Age:      timeAgo(time.Unix(item.CreationDate, 0)),
		Body:     body,
	}
}

func cleanHTML(s string) string {
	// Replace common block elements with newlines
	s = strings.ReplaceAll(s, "<p>", "\n")
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "<li>", "\n- ")
	s = strings.ReplaceAll(s, "</li>", "")
	s = strings.ReplaceAll(s, "<code>", "`")
	s = strings.ReplaceAll(s, "</code>", "`")
	s = strings.ReplaceAll(s, "<pre>", "\n```\n")
	s = strings.ReplaceAll(s, "</pre>", "\n```\n")

	// Remove remaining HTML tags
	s = htmlTagRe.ReplaceAllString(s, "")

	// Decode HTML entities
	s = html.UnescapeString(s)

	// Normalize whitespace
	s = strings.TrimSpace(s)

	return s
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
	case diff < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(diff.Hours()/24))
	case diff < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(diff.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(diff.Hours()/(24*365)))
	}
}
