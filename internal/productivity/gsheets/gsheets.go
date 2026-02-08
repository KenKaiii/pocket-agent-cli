package gsheets

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

const baseURL = "https://sheets.googleapis.com/v4/spreadsheets"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// SpreadsheetInfo holds metadata about a spreadsheet
type SpreadsheetInfo struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Locale string      `json:"locale"`
	Sheets []SheetInfo `json:"sheets"`
}

// SheetInfo holds metadata about a single sheet
type SheetInfo struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Index    int    `json:"index"`
	RowCount int    `json:"row_count"`
	ColCount int    `json:"col_count"`
}

// SheetData holds cell values from a range
type SheetData struct {
	SpreadsheetID string     `json:"spreadsheet_id"`
	Range         string     `json:"range"`
	Rows          [][]string `json:"rows"`
	RowCount      int        `json:"row_count"`
	ColCount      int        `json:"col_count"`
}

// SearchResult holds search matches
type SearchResult struct {
	Query   string      `json:"query"`
	Matches []CellMatch `json:"matches"`
	Count   int         `json:"count"`
}

// CellMatch holds a single cell match
type CellMatch struct {
	Sheet string `json:"sheet"`
	Cell  string `json:"cell"`
	Value string `json:"value"`
	Row   int    `json:"row"`
	Col   int    `json:"col"`
}

// NewCmd returns the Google Sheets command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gsheets",
		Aliases: []string{"sheets", "spreadsheet"},
		Short:   "Google Sheets commands",
	}

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newSearchCmd())

	return cmd
}

func getAPIKey() (string, error) {
	key, err := config.Get("google_api_key")
	if err != nil || key == "" {
		return "", output.PrintError("setup_required", "Google API key not configured", map[string]any{
			"missing":   []string{"google_api_key"},
			"setup_cmd": "pocket config set google_api_key <your-key>",
			"hint":      "Get an API key from https://console.cloud.google.com/apis/credentials",
		})
	}
	return key, nil
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [spreadsheet-id]",
		Short: "Get spreadsheet metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			spreadsheetID := args[0]
			reqURL := fmt.Sprintf("%s/%s?key=%s&fields=properties,sheets.properties",
				baseURL, url.PathEscape(spreadsheetID), url.QueryEscape(apiKey))

			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Properties struct {
					Title  string `json:"title"`
					Locale string `json:"locale"`
				} `json:"properties"`
				Sheets []struct {
					Properties struct {
						SheetID   int    `json:"sheetId"`
						Title     string `json:"title"`
						Index     int    `json:"index"`
						GridProps struct {
							RowCount    int `json:"rowCount"`
							ColumnCount int `json:"columnCount"`
						} `json:"gridProperties"`
					} `json:"properties"`
				} `json:"sheets"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
			}

			sheets := make([]SheetInfo, 0, len(resp.Sheets))
			for _, s := range resp.Sheets {
				sheets = append(sheets, SheetInfo{
					ID:       s.Properties.SheetID,
					Title:    s.Properties.Title,
					Index:    s.Properties.Index,
					RowCount: s.Properties.GridProps.RowCount,
					ColCount: s.Properties.GridProps.ColumnCount,
				})
			}

			result := SpreadsheetInfo{
				ID:     spreadsheetID,
				Title:  resp.Properties.Title,
				Locale: resp.Properties.Locale,
				Sheets: sheets,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [spreadsheet-id] [range]",
		Short: "Read cell values from a range",
		Long:  `Read cell values. Range format: "Sheet1!A1:D10" or "A1:D10"`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			spreadsheetID := args[0]
			rangeStr := args[1]

			reqURL := fmt.Sprintf("%s/%s/values/%s?key=%s",
				baseURL, url.PathEscape(spreadsheetID),
				url.PathEscape(rangeStr), url.QueryEscape(apiKey))

			data, err := doRequest(reqURL)
			if err != nil {
				return err
			}

			var resp struct {
				Range  string          `json:"range"`
				Values [][]interface{} `json:"values"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse response: %s", err.Error()), nil)
			}

			// Convert to string slices
			rows := make([][]string, 0, len(resp.Values))
			maxCols := 0
			for _, row := range resp.Values {
				strRow := make([]string, 0, len(row))
				for _, cell := range row {
					strRow = append(strRow, fmt.Sprintf("%v", cell))
				}
				rows = append(rows, strRow)
				if len(strRow) > maxCols {
					maxCols = len(strRow)
				}
			}

			result := SheetData{
				SpreadsheetID: spreadsheetID,
				Range:         resp.Range,
				Rows:          rows,
				RowCount:      len(rows),
				ColCount:      maxCols,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [spreadsheet-id] [query]",
		Short: "Search for a value across all sheets",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			spreadsheetID := args[0]
			query := args[1]

			// First, get sheet list
			metaURL := fmt.Sprintf("%s/%s?key=%s&fields=sheets.properties.title",
				baseURL, url.PathEscape(spreadsheetID), url.QueryEscape(apiKey))

			metaData, err := doRequest(metaURL)
			if err != nil {
				return err
			}

			var metaResp struct {
				Sheets []struct {
					Properties struct {
						Title string `json:"title"`
					} `json:"properties"`
				} `json:"sheets"`
			}

			if err := json.Unmarshal(metaData, &metaResp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse metadata: %s", err.Error()), nil)
			}

			if len(metaResp.Sheets) == 0 {
				return output.Print(SearchResult{
					Query:   query,
					Matches: []CellMatch{},
					Count:   0,
				})
			}

			// Build batch read URL with ranges for all sheets
			params := url.Values{
				"key": {apiKey},
			}
			for _, sheet := range metaResp.Sheets {
				params.Add("ranges", sheet.Properties.Title+"!A:ZZ")
			}

			batchURL := fmt.Sprintf("%s/%s/values:batchGet?%s",
				baseURL, url.PathEscape(spreadsheetID), params.Encode())

			batchData, err := doRequest(batchURL)
			if err != nil {
				return err
			}

			var batchResp struct {
				ValueRanges []struct {
					Range  string          `json:"range"`
					Values [][]interface{} `json:"values"`
				} `json:"valueRanges"`
			}

			if err := json.Unmarshal(batchData, &batchResp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse batch data: %s", err.Error()), nil)
			}

			queryLower := strings.ToLower(query)
			var matches []CellMatch

			for _, vr := range batchResp.ValueRanges {
				// Extract sheet name from range like "Sheet1!A1:ZZ1000"
				sheetName := vr.Range
				if idx := strings.Index(sheetName, "!"); idx >= 0 {
					sheetName = sheetName[:idx]
				}
				// Remove surrounding single quotes if present
				sheetName = strings.Trim(sheetName, "'")

				for rowIdx, row := range vr.Values {
					for colIdx, cell := range row {
						cellStr := fmt.Sprintf("%v", cell)
						if strings.Contains(strings.ToLower(cellStr), queryLower) {
							cellRef := fmt.Sprintf("%s%d", colToLetter(colIdx), rowIdx+1)
							matches = append(matches, CellMatch{
								Sheet: sheetName,
								Cell:  cellRef,
								Value: cellStr,
								Row:   rowIdx + 1,
								Col:   colIdx + 1,
							})
						}
					}
				}
			}

			if matches == nil {
				matches = []CellMatch{}
			}

			result := SearchResult{
				Query:   query,
				Matches: matches,
				Count:   len(matches),
			}

			return output.Print(result)
		},
	}

	return cmd
}

func doRequest(reqURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("request_failed", fmt.Sprintf("Failed to create request: %s", err.Error()), nil)
	}

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

	if resp.StatusCode == 403 {
		return nil, output.PrintError("forbidden", "Access denied. Ensure the spreadsheet is public and the API key is valid", nil)
	}

	if resp.StatusCode == 404 {
		return nil, output.PrintError("not_found", "Spreadsheet or range not found", nil)
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

func colToLetter(colIndex int) string {
	result := ""
	for colIndex >= 0 {
		result = string(rune('A'+colIndex%26)) + result
		colIndex = colIndex/26 - 1
	}
	return result
}
