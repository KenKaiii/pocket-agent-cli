package trello

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

const baseURL = "https://api.trello.com/1"

// Board represents a Trello board
type Board struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"desc"`
	URL            string `json:"url"`
	ShortURL       string `json:"shortUrl"`
	Closed         bool   `json:"closed"`
	IDOrganization string `json:"idOrganization"`
}

// List represents a Trello list
type List struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Closed   bool    `json:"closed"`
	IDBoard  string  `json:"idBoard"`
	Position float64 `json:"pos"`
}

// Card represents a Trello card
type Card struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"desc"`
	URL              string   `json:"url"`
	ShortURL         string   `json:"shortUrl"`
	Closed           bool     `json:"closed"`
	IDBoard          string   `json:"idBoard"`
	IDList           string   `json:"idList"`
	Due              string   `json:"due"`
	DueComplete      bool     `json:"dueComplete"`
	IDLabels         []string `json:"idLabels"`
	IDMembers        []string `json:"idMembers"`
	Position         float64  `json:"pos"`
	DateLastActivity string   `json:"dateLastActivity"`
}

// Label represents a Trello label
type Label struct {
	ID      string `json:"id"`
	IDBoard string `json:"idBoard"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

// BoardDetails includes a board with its lists
type BoardDetails struct {
	Board Board  `json:"board"`
	Lists []List `json:"lists"`
}

// CardDetails includes full card information
type CardDetails struct {
	Card   Card    `json:"card"`
	Labels []Label `json:"labels,omitempty"`
}

// Client handles Trello API requests
type Client struct {
	apiKey string
	token  string
	http   *http.Client
}

// NewClient creates a new Trello client
func NewClient(apiKey, token string) *Client {
	return &Client{
		apiKey: apiKey,
		token:  token,
		http:   &http.Client{},
	}
}

func (c *Client) request(method, endpoint string, params url.Values, body string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if params == nil {
		params = url.Values{}
	}
	params.Set("key", c.apiKey)
	params.Set("token", c.token)

	reqURL := fmt.Sprintf("%s%s?%s", baseURL, endpoint, params.Encode())

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// GetBoards returns all boards for the authenticated user
func (c *Client) GetBoards() ([]Board, error) {
	data, err := c.request("GET", "/members/me/boards", nil, "")
	if err != nil {
		return nil, err
	}

	var boards []Board
	if err := json.Unmarshal(data, &boards); err != nil {
		return nil, fmt.Errorf("failed to parse boards: %w", err)
	}

	return boards, nil
}

// GetBoard returns a single board by ID
func (c *Client) GetBoard(id string) (*Board, error) {
	data, err := c.request("GET", "/boards/"+id, nil, "")
	if err != nil {
		return nil, err
	}

	var board Board
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to parse board: %w", err)
	}

	return &board, nil
}

// GetBoardLists returns all lists on a board
func (c *Client) GetBoardLists(boardID string) ([]List, error) {
	data, err := c.request("GET", "/boards/"+boardID+"/lists", nil, "")
	if err != nil {
		return nil, err
	}

	var lists []List
	if err := json.Unmarshal(data, &lists); err != nil {
		return nil, fmt.Errorf("failed to parse lists: %w", err)
	}

	return lists, nil
}

// GetBoardCards returns all cards on a board
func (c *Client) GetBoardCards(boardID string) ([]Card, error) {
	data, err := c.request("GET", "/boards/"+boardID+"/cards", nil, "")
	if err != nil {
		return nil, err
	}

	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("failed to parse cards: %w", err)
	}

	return cards, nil
}

// GetListCards returns all cards in a list
func (c *Client) GetListCards(listID string) ([]Card, error) {
	data, err := c.request("GET", "/lists/"+listID+"/cards", nil, "")
	if err != nil {
		return nil, err
	}

	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("failed to parse cards: %w", err)
	}

	return cards, nil
}

// GetCard returns a single card by ID
func (c *Client) GetCard(id string) (*Card, error) {
	data, err := c.request("GET", "/cards/"+id, nil, "")
	if err != nil {
		return nil, err
	}

	var card Card
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to parse card: %w", err)
	}

	return &card, nil
}

// GetCardLabels returns labels attached to a card
func (c *Client) GetCardLabels(cardID string) ([]Label, error) {
	data, err := c.request("GET", "/cards/"+cardID+"/labels", nil, "")
	if err != nil {
		return nil, err
	}

	var labels []Label
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, fmt.Errorf("failed to parse labels: %w", err)
	}

	return labels, nil
}

// CreateCard creates a new card on a list
func (c *Client) CreateCard(name, listID, description string) (*Card, error) {
	params := url.Values{}
	params.Set("name", name)
	params.Set("idList", listID)
	if description != "" {
		params.Set("desc", description)
	}

	data, err := c.request("POST", "/cards", params, "")
	if err != nil {
		return nil, err
	}

	var card Card
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to parse created card: %w", err)
	}

	return &card, nil
}

// NewCmd returns the trello command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "trello",
		Aliases: []string{"tr"},
		Short:   "Trello commands",
	}

	cmd.AddCommand(newBoardsCmd())
	cmd.AddCommand(newBoardCmd())
	cmd.AddCommand(newCardsCmd())
	cmd.AddCommand(newCardCmd())
	cmd.AddCommand(newCreateCmd())

	return cmd
}

func getClient() (*Client, error) {
	apiKey, err := config.Get("trello_key")
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, output.PrintError("config_missing", "Trello API key not configured", map[string]any{
			"setup_instructions": "To configure Trello:\n" +
				"1. Go to https://trello.com/power-ups/admin\n" +
				"2. Click 'New' to create a new Power-Up (or use existing)\n" +
				"3. Get your API key from the Power-Up settings\n" +
				"4. Generate a token by clicking 'Token' link on the API key page\n" +
				"5. Run: pocket config set trello_key <your-api-key>\n" +
				"6. Run: pocket config set trello_token <your-token>",
		})
	}

	token, err := config.Get("trello_token")
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, output.PrintError("config_missing", "Trello token not configured", map[string]any{
			"setup_instructions": "To configure Trello token:\n" +
				"1. Go to https://trello.com/power-ups/admin\n" +
				"2. Select your Power-Up and find your API key\n" +
				"3. Click 'Token' link to generate an access token\n" +
				"4. Authorize the token for your account\n" +
				"5. Run: pocket config set trello_token <your-token>",
		})
	}

	return NewClient(apiKey, token), nil
}

func newBoardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boards",
		Short: "List my boards",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			boards, err := client.GetBoards()
			if err != nil {
				return output.PrintError("api_error", "Failed to fetch boards", map[string]any{
					"error": err.Error(),
				})
			}

			// Filter out closed boards by default
			openBoards := make([]Board, 0)
			for _, b := range boards {
				if !b.Closed {
					openBoards = append(openBoards, b)
				}
			}

			return output.Print(map[string]any{
				"boards": openBoards,
				"count":  len(openBoards),
			})
		},
	}

	return cmd
}

func newBoardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "board [id]",
		Short: "Get board details with lists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			boardID := args[0]

			board, err := client.GetBoard(boardID)
			if err != nil {
				return output.PrintError("api_error", "Failed to fetch board", map[string]any{
					"board_id": boardID,
					"error":    err.Error(),
				})
			}

			lists, err := client.GetBoardLists(boardID)
			if err != nil {
				return output.PrintError("api_error", "Failed to fetch board lists", map[string]any{
					"board_id": boardID,
					"error":    err.Error(),
				})
			}

			// Filter out closed lists
			openLists := make([]List, 0)
			for _, l := range lists {
				if !l.Closed {
					openLists = append(openLists, l)
				}
			}

			return output.Print(BoardDetails{
				Board: *board,
				Lists: openLists,
			})
		},
	}

	return cmd
}

func newCardsCmd() *cobra.Command {
	var listID string

	cmd := &cobra.Command{
		Use:   "cards [board-id]",
		Short: "List cards on a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			boardID := args[0]

			var cards []Card

			if listID != "" {
				// Fetch cards from specific list
				cards, err = client.GetListCards(listID)
				if err != nil {
					return output.PrintError("api_error", "Failed to fetch cards from list", map[string]any{
						"list_id": listID,
						"error":   err.Error(),
					})
				}
			} else {
				// Fetch all cards from board
				cards, err = client.GetBoardCards(boardID)
				if err != nil {
					return output.PrintError("api_error", "Failed to fetch cards from board", map[string]any{
						"board_id": boardID,
						"error":    err.Error(),
					})
				}
			}

			// Filter out closed cards
			openCards := make([]Card, 0)
			for _, c := range cards {
				if !c.Closed {
					openCards = append(openCards, c)
				}
			}

			return output.Print(map[string]any{
				"cards":    openCards,
				"count":    len(openCards),
				"board_id": boardID,
				"list_id":  listID,
			})
		},
	}

	cmd.Flags().StringVarP(&listID, "list", "l", "", "Filter by list ID")

	return cmd
}

func newCardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card [id]",
		Short: "Get card details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			cardID := args[0]

			card, err := client.GetCard(cardID)
			if err != nil {
				return output.PrintError("api_error", "Failed to fetch card", map[string]any{
					"card_id": cardID,
					"error":   err.Error(),
				})
			}

			labels, err := client.GetCardLabels(cardID)
			if err != nil {
				// Labels are optional, don't fail if we can't get them
				labels = nil
			}

			return output.Print(CardDetails{
				Card:   *card,
				Labels: labels,
			})
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var boardID string
	var listID string
	var description string

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a card",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			name := args[0]

			// If only board is provided, we need to find a list
			if listID == "" && boardID != "" {
				lists, err := client.GetBoardLists(boardID)
				if err != nil {
					return output.PrintError("api_error", "Failed to fetch board lists", map[string]any{
						"board_id": boardID,
						"error":    err.Error(),
					})
				}

				// Find first open list
				for _, l := range lists {
					if !l.Closed {
						listID = l.ID
						break
					}
				}

				if listID == "" {
					return output.PrintError("no_list", "No open lists found on board", map[string]any{
						"board_id": boardID,
					})
				}
			}

			if listID == "" {
				return output.PrintError("missing_list", "List ID is required (use --list or --board)", map[string]any{
					"hint": "Specify --list with a list ID, or --board to use the first list on that board",
				})
			}

			card, err := client.CreateCard(name, listID, description)
			if err != nil {
				return output.PrintError("api_error", "Failed to create card", map[string]any{
					"error": err.Error(),
				})
			}

			return output.Print(map[string]any{
				"message": "Card created successfully",
				"card":    card,
			})
		},
	}

	cmd.Flags().StringVarP(&boardID, "board", "b", "", "Board ID (uses first list if --list not specified)")
	cmd.Flags().StringVarP(&listID, "list", "l", "", "List ID to create card in")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Card description")

	return cmd
}
