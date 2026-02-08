package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://api.cloudflare.com/client/v4"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Zone is LLM-friendly zone output
type Zone struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	Paused          bool     `json:"paused"`
	Type            string   `json:"type"`
	NameServers     []string `json:"nameservers,omitempty"`
	OriginalNS      []string `json:"original_ns,omitempty"`
	Plan            string   `json:"plan,omitempty"`
	AccountName     string   `json:"account_name,omitempty"`
	CreatedOn       string   `json:"created_on,omitempty"`
	ModifiedOn      string   `json:"modified_on,omitempty"`
	ActivatedOn     string   `json:"activated_on,omitempty"`
	DevelopmentMode int      `json:"development_mode,omitempty"`
}

// DNSRecord is LLM-friendly DNS record output
type DNSRecord struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Proxied    bool   `json:"proxied"`
	TTL        int    `json:"ttl"`
	Priority   int    `json:"priority,omitempty"`
	Locked     bool   `json:"locked,omitempty"`
	CreatedOn  string `json:"created_on,omitempty"`
	ModifiedOn string `json:"modified_on,omitempty"`
}

// Analytics is LLM-friendly analytics output
type Analytics struct {
	Requests     AnalyticsRequests  `json:"requests"`
	Bandwidth    AnalyticsBandwidth `json:"bandwidth"`
	Threats      AnalyticsThreats   `json:"threats"`
	PageViews    AnalyticsPageViews `json:"pageviews"`
	Uniques      AnalyticsUniques   `json:"uniques"`
	HTTPStatus   map[string]int     `json:"http_status,omitempty"`
	ContentTypes map[string]int     `json:"content_types,omitempty"`
	Countries    map[string]int     `json:"countries,omitempty"`
	Since        string             `json:"since"`
	Until        string             `json:"until"`
}

// AnalyticsRequests contains request analytics
type AnalyticsRequests struct {
	All          int `json:"all"`
	Cached       int `json:"cached"`
	Uncached     int `json:"uncached"`
	SSLEncrypted int `json:"ssl_encrypted"`
}

// AnalyticsBandwidth contains bandwidth analytics
type AnalyticsBandwidth struct {
	All      int64 `json:"all"`
	Cached   int64 `json:"cached"`
	Uncached int64 `json:"uncached"`
}

// AnalyticsThreats contains threat analytics
type AnalyticsThreats struct {
	All int `json:"all"`
}

// AnalyticsPageViews contains page view analytics
type AnalyticsPageViews struct {
	All int `json:"all"`
}

// AnalyticsUniques contains unique visitor analytics
type AnalyticsUniques struct {
	All int `json:"all"`
}

// PurgeResult is LLM-friendly purge result output
type PurgeResult struct {
	ID string `json:"id"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cloudflare",
		Aliases: []string{"cf"},
		Short:   "Cloudflare commands",
	}

	cmd.AddCommand(newZonesCmd())
	cmd.AddCommand(newZoneCmd())
	cmd.AddCommand(newDNSCmd())
	cmd.AddCommand(newPurgeCmd())
	cmd.AddCommand(newAnalyticsCmd())

	return cmd
}

func newZonesCmd() *cobra.Command {
	var limit int
	var status string
	var name string

	cmd := &cobra.Command{
		Use:   "zones",
		Short: "List all zones/domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/zones?per_page=%d", baseURL, limit)
			if status != "" {
				url += "&status=" + status
			}
			if name != "" {
				url += "&name=" + name
			}

			var resp cfResponse
			if err := cfGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			if !resp.Success {
				return output.PrintError("api_error", formatErrors(resp.Errors), nil)
			}

			zones, err := parseZones(resp.Result)
			if err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			return output.Print(zones)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of zones")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status: active, pending, initializing, moved, deleted, deactivated")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Filter by domain name")

	return cmd
}

func newZoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zone [zone-id]",
		Short: "Get zone details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/zones/%s", baseURL, args[0])

			var resp cfResponse
			if err := cfGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			if !resp.Success {
				return output.PrintError("api_error", formatErrors(resp.Errors), nil)
			}

			zone, err := parseZone(resp.Result)
			if err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			return output.Print(zone)
		},
	}

	return cmd
}

func newDNSCmd() *cobra.Command {
	var recordType string
	var limit int
	var name string

	cmd := &cobra.Command{
		Use:   "dns [zone-id]",
		Short: "List DNS records for a zone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/zones/%s/dns_records?per_page=%d", baseURL, args[0], limit)
			if recordType != "" {
				url += "&type=" + strings.ToUpper(recordType)
			}
			if name != "" {
				url += "&name=" + name
			}

			var resp cfResponse
			if err := cfGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			if !resp.Success {
				return output.PrintError("api_error", formatErrors(resp.Errors), nil)
			}

			records, err := parseDNSRecords(resp.Result)
			if err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			return output.Print(records)
		},
	}

	cmd.Flags().StringVarP(&recordType, "type", "t", "", "Filter by record type: A, AAAA, CNAME, MX, TXT, NS, etc.")
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Number of records")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Filter by record name")

	return cmd
}

func newPurgeCmd() *cobra.Command {
	var purgeAll bool
	var urls []string
	var tags []string
	var hosts []string
	var prefixes []string

	cmd := &cobra.Command{
		Use:   "purge [zone-id]",
		Short: "Purge cache for a zone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/zones/%s/purge_cache", baseURL, args[0])

			var body map[string]any
			if purgeAll {
				body = map[string]any{"purge_everything": true}
			} else if len(urls) > 0 {
				body = map[string]any{"files": urls}
			} else if len(tags) > 0 {
				body = map[string]any{"tags": tags}
			} else if len(hosts) > 0 {
				body = map[string]any{"hosts": hosts}
			} else if len(prefixes) > 0 {
				body = map[string]any{"prefixes": prefixes}
			} else {
				return output.PrintError("missing_option", "Specify --all, --urls, --tags, --hosts, or --prefixes", nil)
			}

			var resp cfResponse
			if err := cfPost(token, url, body, &resp); err != nil {
				return output.PrintError("purge_failed", err.Error(), nil)
			}

			if !resp.Success {
				return output.PrintError("api_error", formatErrors(resp.Errors), nil)
			}

			result, err := parsePurgeResult(resp.Result)
			if err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"message": "Cache purge initiated",
				"id":      result.ID,
			})
		},
	}

	cmd.Flags().BoolVarP(&purgeAll, "all", "a", false, "Purge everything")
	cmd.Flags().StringSliceVar(&urls, "urls", nil, "Specific URLs to purge (comma-separated)")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Cache tags to purge (Enterprise only)")
	cmd.Flags().StringSliceVar(&hosts, "hosts", nil, "Hostnames to purge (Enterprise only)")
	cmd.Flags().StringSliceVar(&prefixes, "prefixes", nil, "URL prefixes to purge (Enterprise only)")

	return cmd
}

func newAnalyticsCmd() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "analytics [zone-id]",
		Short: "Get zone analytics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := getToken()
			if err != nil {
				return err
			}

			// Calculate time range
			now := time.Now().UTC()
			since := now.AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")
			until := now.Format("2006-01-02T15:04:05Z")

			url := fmt.Sprintf("%s/zones/%s/analytics/dashboard?since=%s&until=%s&continuous=true",
				baseURL, args[0], since, until)

			var resp cfResponse
			if err := cfGet(token, url, &resp); err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			if !resp.Success {
				return output.PrintError("api_error", formatErrors(resp.Errors), nil)
			}

			analytics, err := parseAnalytics(resp.Result, since, until)
			if err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			return output.Print(analytics)
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 1, "Number of days of analytics (max 30)")

	return cmd
}

// cfResponse is the standard Cloudflare API response
type cfResponse struct {
	Success  bool            `json:"success"`
	Errors   []cfError       `json:"errors"`
	Messages []string        `json:"messages"`
	Result   json.RawMessage `json:"result"`
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func getToken() (string, error) {
	token, err := config.Get("cloudflare_token")
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", output.PrintError("missing_config", "Cloudflare token not configured", map[string]string{
			"setup":       "Run: pocket config set cloudflare_token <your-api-token>",
			"docs":        "Create an API token at: https://dash.cloudflare.com/profile/api-tokens",
			"permissions": "Required permissions depend on commands: Zone:Read for zones/dns, Cache Purge for purge, Analytics:Read for analytics",
		})
	}
	return token, nil
}

func cfGet(token, url string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp cfResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		if len(errResp.Errors) > 0 {
			return fmt.Errorf("%s", formatErrors(errResp.Errors))
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func cfPost(token, url string, body any, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp cfResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		if len(errResp.Errors) > 0 {
			return fmt.Errorf("%s", formatErrors(errResp.Errors))
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func formatErrors(errors []cfError) string {
	if len(errors) == 0 {
		return "unknown error"
	}
	msgs := make([]string, len(errors))
	for i, e := range errors {
		msgs[i] = fmt.Sprintf("[%d] %s", e.Code, e.Message)
	}
	return strings.Join(msgs, "; ")
}

func parseZones(raw json.RawMessage) ([]Zone, error) {
	var zones []map[string]any
	if err := json.Unmarshal(raw, &zones); err != nil {
		return nil, err
	}

	result := make([]Zone, 0, len(zones))
	for _, z := range zones {
		result = append(result, toZone(z))
	}
	return result, nil
}

func parseZone(raw json.RawMessage) (Zone, error) {
	var z map[string]any
	if err := json.Unmarshal(raw, &z); err != nil {
		return Zone{}, err
	}
	return toZone(z), nil
}

func toZone(z map[string]any) Zone {
	zone := Zone{
		ID:              getString(z, "id"),
		Name:            getString(z, "name"),
		Status:          getString(z, "status"),
		Paused:          getBool(z, "paused"),
		Type:            getString(z, "type"),
		DevelopmentMode: getInt(z, "development_mode"),
	}

	if ns, ok := z["name_servers"].([]any); ok {
		for _, n := range ns {
			if s, ok := n.(string); ok {
				zone.NameServers = append(zone.NameServers, s)
			}
		}
	}

	if ons, ok := z["original_name_servers"].([]any); ok {
		for _, n := range ons {
			if s, ok := n.(string); ok {
				zone.OriginalNS = append(zone.OriginalNS, s)
			}
		}
	}

	if plan, ok := z["plan"].(map[string]any); ok {
		zone.Plan = getString(plan, "name")
	}

	if account, ok := z["account"].(map[string]any); ok {
		zone.AccountName = getString(account, "name")
	}

	if created := getString(z, "created_on"); created != "" {
		zone.CreatedOn = parseTime(created)
	}
	if modified := getString(z, "modified_on"); modified != "" {
		zone.ModifiedOn = parseTime(modified)
	}
	if activated := getString(z, "activated_on"); activated != "" {
		zone.ActivatedOn = parseTime(activated)
	}

	return zone
}

func parseDNSRecords(raw json.RawMessage) ([]DNSRecord, error) {
	var records []map[string]any
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, err
	}

	result := make([]DNSRecord, 0, len(records))
	for _, r := range records {
		result = append(result, toDNSRecord(r))
	}
	return result, nil
}

func toDNSRecord(r map[string]any) DNSRecord {
	record := DNSRecord{
		ID:       getString(r, "id"),
		Type:     getString(r, "type"),
		Name:     getString(r, "name"),
		Content:  getString(r, "content"),
		Proxied:  getBool(r, "proxied"),
		TTL:      getInt(r, "ttl"),
		Priority: getInt(r, "priority"),
		Locked:   getBool(r, "locked"),
	}

	if created := getString(r, "created_on"); created != "" {
		record.CreatedOn = parseTime(created)
	}
	if modified := getString(r, "modified_on"); modified != "" {
		record.ModifiedOn = parseTime(modified)
	}

	return record
}

func parsePurgeResult(raw json.RawMessage) (PurgeResult, error) {
	var r map[string]any
	if err := json.Unmarshal(raw, &r); err != nil {
		return PurgeResult{}, err
	}
	return PurgeResult{
		ID: getString(r, "id"),
	}, nil
}

func parseAnalytics(raw json.RawMessage, since, until string) (Analytics, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return Analytics{}, err
	}

	analytics := Analytics{
		Since: since,
		Until: until,
	}

	// Parse totals
	if totals, ok := data["totals"].(map[string]any); ok {
		// Requests
		if requests, ok := totals["requests"].(map[string]any); ok {
			analytics.Requests.All = getInt(requests, "all")
			analytics.Requests.Cached = getInt(requests, "cached")
			analytics.Requests.Uncached = getInt(requests, "uncached")
			if ssl, ok := requests["ssl"].(map[string]any); ok {
				analytics.Requests.SSLEncrypted = getInt(ssl, "encrypted")
			}
		}

		// Bandwidth
		if bandwidth, ok := totals["bandwidth"].(map[string]any); ok {
			analytics.Bandwidth.All = getInt64(bandwidth, "all")
			analytics.Bandwidth.Cached = getInt64(bandwidth, "cached")
			analytics.Bandwidth.Uncached = getInt64(bandwidth, "uncached")
		}

		// Threats
		if threats, ok := totals["threats"].(map[string]any); ok {
			analytics.Threats.All = getInt(threats, "all")
		}

		// Page views
		if pageviews, ok := totals["pageviews"].(map[string]any); ok {
			analytics.PageViews.All = getInt(pageviews, "all")
		}

		// Uniques
		if uniques, ok := totals["uniques"].(map[string]any); ok {
			analytics.Uniques.All = getInt(uniques, "all")
		}

		// HTTP status breakdown
		if requests, ok := totals["requests"].(map[string]any); ok {
			if httpStatus, ok := requests["http_status"].(map[string]any); ok {
				analytics.HTTPStatus = make(map[string]int)
				for k, v := range httpStatus {
					if num, ok := v.(float64); ok {
						analytics.HTTPStatus[k] = int(num)
					}
				}
			}

			if contentTypes, ok := requests["content_type"].(map[string]any); ok {
				analytics.ContentTypes = make(map[string]int)
				for k, v := range contentTypes {
					if num, ok := v.(float64); ok {
						analytics.ContentTypes[k] = int(num)
					}
				}
			}

			if countries, ok := requests["country"].(map[string]any); ok {
				analytics.Countries = make(map[string]int)
				for k, v := range countries {
					if num, ok := v.(float64); ok {
						analytics.Countries[k] = int(num)
					}
				}
			}
		}
	}

	return analytics, nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func parseTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}
