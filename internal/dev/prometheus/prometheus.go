package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// QueryResult is LLM-friendly output for an instant query
type QueryResult struct {
	Query      string         `json:"query"`
	ResultType string         `json:"result_type"`
	Results    []MetricResult `json:"results"`
}

// MetricResult is a single metric result from a Prometheus query
type MetricResult struct {
	Metric    map[string]string `json:"metric"`
	Value     string            `json:"value"`
	Timestamp float64           `json:"timestamp"`
}

// RangeResult is LLM-friendly output for a range query
type RangeResult struct {
	Query      string        `json:"query"`
	ResultType string        `json:"result_type"`
	Results    []RangeMetric `json:"results"`
}

// RangeMetric is a single metric result from a Prometheus range query
type RangeMetric struct {
	Metric map[string]string `json:"metric"`
	Values []TimeValue       `json:"values"`
}

// TimeValue is a timestamp-value pair
type TimeValue struct {
	Timestamp float64 `json:"timestamp"`
	Value     string  `json:"value"`
}

// Alert is LLM-friendly output for a Prometheus alert
type Alert struct {
	Name        string            `json:"name"`
	State       string            `json:"state"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ActiveAt    string            `json:"active_at,omitempty"`
	Value       string            `json:"value,omitempty"`
}

// Target is LLM-friendly output for a Prometheus target
type Target struct {
	Instance       string `json:"instance"`
	Job            string `json:"job"`
	Health         string `json:"health"`
	LastScrape     string `json:"last_scrape"`
	ScrapeInterval string `json:"scrape_interval"`
	ScrapeURL      string `json:"scrape_url"`
}

// NewCmd returns the prometheus parent command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "prometheus",
		Aliases: []string{"prom", "pm"},
		Short:   "Prometheus monitoring commands",
	}

	cmd.AddCommand(newQueryCmd())
	cmd.AddCommand(newRangeCmd())
	cmd.AddCommand(newAlertsCmd())
	cmd.AddCommand(newTargetsCmd())

	return cmd
}

func getBaseURL() string {
	u, err := config.Get("prometheus_url")
	if err != nil || u == "" {
		return "http://localhost:9090"
	}
	return u
}

func getToken() string {
	t, err := config.Get("prometheus_token")
	if err != nil || t == "" {
		return ""
	}
	return t
}

func newQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [promql]",
		Short: "Run an instant PromQL query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			promql := args[0]
			baseURL := getBaseURL()

			apiURL := fmt.Sprintf("%s/api/v1/query?query=%s&time=%d",
				baseURL,
				url.QueryEscape(promql),
				time.Now().Unix(),
			)

			var raw map[string]any
			if err := promGet(apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			status := getString(raw, "status")
			if status != "success" {
				errMsg := getString(raw, "error")
				return output.PrintError("query_failed", errMsg, nil)
			}

			data, _ := raw["data"].(map[string]any)
			resultType := getString(data, "resultType")
			rawResults, _ := data["result"].([]any)

			results := make([]MetricResult, 0, len(rawResults))
			for _, r := range rawResults {
				if m, ok := r.(map[string]any); ok {
					mr := MetricResult{
						Metric: toStringMap(m["metric"]),
					}

					if value, ok := m["value"].([]any); ok && len(value) == 2 {
						if ts, ok := value[0].(float64); ok {
							mr.Timestamp = ts
						}
						if v, ok := value[1].(string); ok {
							mr.Value = v
						}
					}

					results = append(results, mr)
				}
			}

			return output.Print(QueryResult{
				Query:      promql,
				ResultType: resultType,
				Results:    results,
			})
		},
	}

	return cmd
}

func newRangeCmd() *cobra.Command {
	var startStr string
	var endStr string
	var step string

	cmd := &cobra.Command{
		Use:   "range [promql]",
		Short: "Run a range PromQL query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			promql := args[0]
			baseURL := getBaseURL()

			now := time.Now()

			var startTime time.Time
			if startStr == "" {
				startTime = now.Add(-1 * time.Hour)
			} else {
				var err error
				startTime, err = time.Parse(time.RFC3339, startStr)
				if err != nil {
					return output.PrintError("invalid_start", fmt.Sprintf("Invalid start time: %s (use RFC3339 format)", startStr), nil)
				}
			}

			var endTime time.Time
			if endStr == "" {
				endTime = now
			} else {
				var err error
				endTime, err = time.Parse(time.RFC3339, endStr)
				if err != nil {
					return output.PrintError("invalid_end", fmt.Sprintf("Invalid end time: %s (use RFC3339 format)", endStr), nil)
				}
			}

			apiURL := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
				baseURL,
				url.QueryEscape(promql),
				startTime.Unix(),
				endTime.Unix(),
				url.QueryEscape(step),
			)

			var raw map[string]any
			if err := promGet(apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			status := getString(raw, "status")
			if status != "success" {
				errMsg := getString(raw, "error")
				return output.PrintError("query_failed", errMsg, nil)
			}

			data, _ := raw["data"].(map[string]any)
			resultType := getString(data, "resultType")
			rawResults, _ := data["result"].([]any)

			results := make([]RangeMetric, 0, len(rawResults))
			for _, r := range rawResults {
				if m, ok := r.(map[string]any); ok {
					rm := RangeMetric{
						Metric: toStringMap(m["metric"]),
					}

					if values, ok := m["values"].([]any); ok {
						for _, v := range values {
							if pair, ok := v.([]any); ok && len(pair) == 2 {
								tv := TimeValue{}
								if ts, ok := pair[0].(float64); ok {
									tv.Timestamp = ts
								}
								if val, ok := pair[1].(string); ok {
									tv.Value = val
								}
								rm.Values = append(rm.Values, tv)
							}
						}
					}

					results = append(results, rm)
				}
			}

			return output.Print(RangeResult{
				Query:      promql,
				ResultType: resultType,
				Results:    results,
			})
		},
	}

	cmd.Flags().StringVarP(&startStr, "start", "s", "", "Start time (RFC3339, default: 1h ago)")
	cmd.Flags().StringVarP(&endStr, "end", "e", "", "End time (RFC3339, default: now)")
	cmd.Flags().StringVar(&step, "step", "15s", "Query step (e.g., 15s, 1m, 5m)")

	return cmd
}

func newAlertsCmd() *cobra.Command {
	var state string

	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "List Prometheus alerts",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := getBaseURL()

			apiURL := fmt.Sprintf("%s/api/v1/alerts", baseURL)

			var raw map[string]any
			if err := promGet(apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			status := getString(raw, "status")
			if status != "success" {
				errMsg := getString(raw, "error")
				return output.PrintError("query_failed", errMsg, nil)
			}

			data, _ := raw["data"].(map[string]any)
			rawAlerts, _ := data["alerts"].([]any)

			alerts := make([]Alert, 0, len(rawAlerts))
			for _, a := range rawAlerts {
				if m, ok := a.(map[string]any); ok {
					alert := Alert{
						State:       getString(m, "state"),
						Labels:      toStringMap(m["labels"]),
						Annotations: toStringMap(m["annotations"]),
						ActiveAt:    getString(m, "activeAt"),
						Value:       getString(m, "value"),
					}
					alert.Name = alert.Labels["alertname"]

					// Filter by state if specified
					if state != "" && alert.State != state {
						continue
					}

					alerts = append(alerts, alert)
				}
			}

			return output.Print(alerts)
		},
	}

	cmd.Flags().StringVar(&state, "state", "", "Filter by state: firing, pending, inactive")

	return cmd
}

func newTargetsCmd() *cobra.Command {
	var state string

	cmd := &cobra.Command{
		Use:   "targets",
		Short: "List Prometheus scrape targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := getBaseURL()

			apiURL := fmt.Sprintf("%s/api/v1/targets", baseURL)
			if state != "" {
				apiURL += "?state=" + url.QueryEscape(state)
			}

			var raw map[string]any
			if err := promGet(apiURL, &raw); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			status := getString(raw, "status")
			if status != "success" {
				errMsg := getString(raw, "error")
				return output.PrintError("query_failed", errMsg, nil)
			}

			data, _ := raw["data"].(map[string]any)
			activeTargets, _ := data["activeTargets"].([]any)

			targets := make([]Target, 0, len(activeTargets))
			for _, t := range activeTargets {
				if m, ok := t.(map[string]any); ok {
					labels := toStringMap(m["labels"])

					target := Target{
						Instance:       labels["instance"],
						Job:            labels["job"],
						Health:         getString(m, "health"),
						LastScrape:     getString(m, "lastScrape"),
						ScrapeInterval: getString(m, "scrapeInterval"),
						ScrapeURL:      getString(m, "scrapeUrl"),
					}

					targets = append(targets, target)
				}
			}

			return output.Print(targets)
		},
	}

	cmd.Flags().StringVar(&state, "state", "", "Filter by state: active, dropped")

	return cmd
}

func promGet(apiURL string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	token := getToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if errMsg := getString(errResp, "error"); errMsg != "" {
				return fmt.Errorf("%s", errMsg)
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func toStringMap(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	result := make(map[string]string, len(m))
	for k, val := range m {
		if s, ok := val.(string); ok {
			result[k] = s
		}
	}
	return result
}
