package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	configPath string
	configOnce sync.Once
)

// Config holds all configuration
type Config struct {
	// Social
	TwitterAPIKey       string `json:"twitter_api_key,omitempty"`
	TwitterAPISecret    string `json:"twitter_api_secret,omitempty"`
	TwitterAccessToken  string `json:"twitter_access_token,omitempty"`
	TwitterAccessSecret string `json:"twitter_access_secret,omitempty"`
	RedditClientID      string `json:"reddit_client_id,omitempty"`
	RedditClientSecret  string `json:"reddit_client_secret,omitempty"`
	MastodonServer      string `json:"mastodon_server,omitempty"`
	MastodonToken       string `json:"mastodon_token,omitempty"`

	// Communication
	SlackToken    string `json:"slack_token,omitempty"`
	DiscordToken  string `json:"discord_token,omitempty"`
	TelegramToken string `json:"telegram_token,omitempty"`
	GmailCredPath string `json:"gmail_cred_path,omitempty"`

	// Dev
	GitHubToken string `json:"github_token,omitempty"`
	GitLabToken string `json:"gitlab_token,omitempty"`
	LinearToken string `json:"linear_token,omitempty"`

	// Productivity
	NotionToken   string `json:"notion_token,omitempty"`
	TodoistToken  string `json:"todoist_token,omitempty"`
	GoogleCredPath string `json:"google_cred_path,omitempty"`

	// News
	NewsAPIKey string `json:"newsapi_key,omitempty"`

	// AI
	OpenAIKey    string `json:"openai_key,omitempty"`
	AnthropicKey string `json:"anthropic_key,omitempty"`
}

// Path returns the config file path
func Path() string {
	configOnce.Do(func() {
		if p := os.Getenv("POCKET_CONFIG"); p != "" {
			configPath = p
			return
		}

		home, err := os.UserHomeDir()
		if err != nil {
			configPath = ".pocket.json"
			return
		}
		configPath = filepath.Join(home, ".config", "pocket", "config.json")
	})
	return configPath
}

// Load reads the config file
func Load() (*Config, error) {
	path := Path()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config file
func Save(cfg *Config) error {
	path := Path()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Set sets a config value by key
func Set(key, value string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	key = normalizeKey(key)

	switch key {
	case "twitter_api_key":
		cfg.TwitterAPIKey = value
	case "twitter_api_secret":
		cfg.TwitterAPISecret = value
	case "twitter_access_token":
		cfg.TwitterAccessToken = value
	case "twitter_access_secret":
		cfg.TwitterAccessSecret = value
	case "reddit_client_id":
		cfg.RedditClientID = value
	case "reddit_client_secret":
		cfg.RedditClientSecret = value
	case "mastodon_server":
		cfg.MastodonServer = value
	case "mastodon_token":
		cfg.MastodonToken = value
	case "slack_token":
		cfg.SlackToken = value
	case "discord_token":
		cfg.DiscordToken = value
	case "telegram_token":
		cfg.TelegramToken = value
	case "gmail_cred_path":
		cfg.GmailCredPath = value
	case "github_token":
		cfg.GitHubToken = value
	case "gitlab_token":
		cfg.GitLabToken = value
	case "linear_token":
		cfg.LinearToken = value
	case "notion_token":
		cfg.NotionToken = value
	case "todoist_token":
		cfg.TodoistToken = value
	case "google_cred_path":
		cfg.GoogleCredPath = value
	case "newsapi_key":
		cfg.NewsAPIKey = value
	case "openai_key":
		cfg.OpenAIKey = value
	case "anthropic_key":
		cfg.AnthropicKey = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(cfg)
}

// Get gets a config value by key
func Get(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	key = normalizeKey(key)

	switch key {
	case "twitter_api_key":
		return cfg.TwitterAPIKey, nil
	case "twitter_api_secret":
		return cfg.TwitterAPISecret, nil
	case "twitter_access_token":
		return cfg.TwitterAccessToken, nil
	case "twitter_access_secret":
		return cfg.TwitterAccessSecret, nil
	case "reddit_client_id":
		return cfg.RedditClientID, nil
	case "reddit_client_secret":
		return cfg.RedditClientSecret, nil
	case "mastodon_server":
		return cfg.MastodonServer, nil
	case "mastodon_token":
		return cfg.MastodonToken, nil
	case "slack_token":
		return cfg.SlackToken, nil
	case "discord_token":
		return cfg.DiscordToken, nil
	case "telegram_token":
		return cfg.TelegramToken, nil
	case "gmail_cred_path":
		return cfg.GmailCredPath, nil
	case "github_token":
		return cfg.GitHubToken, nil
	case "gitlab_token":
		return cfg.GitLabToken, nil
	case "linear_token":
		return cfg.LinearToken, nil
	case "notion_token":
		return cfg.NotionToken, nil
	case "todoist_token":
		return cfg.TodoistToken, nil
	case "google_cred_path":
		return cfg.GoogleCredPath, nil
	case "newsapi_key":
		return cfg.NewsAPIKey, nil
	case "openai_key":
		return cfg.OpenAIKey, nil
	case "anthropic_key":
		return cfg.AnthropicKey, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Redacted returns config with sensitive values masked
func (c *Config) Redacted() map[string]string {
	redact := func(s string) string {
		if s == "" {
			return "(not set)"
		}
		if len(s) <= 8 {
			return "****"
		}
		return s[:4] + "****" + s[len(s)-4:]
	}

	return map[string]string{
		"twitter_api_key":       redact(c.TwitterAPIKey),
		"twitter_api_secret":    redact(c.TwitterAPISecret),
		"twitter_access_token":  redact(c.TwitterAccessToken),
		"twitter_access_secret": redact(c.TwitterAccessSecret),
		"reddit_client_id":      redact(c.RedditClientID),
		"reddit_client_secret":  redact(c.RedditClientSecret),
		"mastodon_server":       c.MastodonServer,
		"mastodon_token":        redact(c.MastodonToken),
		"slack_token":           redact(c.SlackToken),
		"discord_token":         redact(c.DiscordToken),
		"telegram_token":        redact(c.TelegramToken),
		"gmail_cred_path":       c.GmailCredPath,
		"github_token":          redact(c.GitHubToken),
		"gitlab_token":          redact(c.GitLabToken),
		"linear_token":          redact(c.LinearToken),
		"notion_token":          redact(c.NotionToken),
		"todoist_token":         redact(c.TodoistToken),
		"google_cred_path":      c.GoogleCredPath,
		"newsapi_key":           redact(c.NewsAPIKey),
		"openai_key":            redact(c.OpenAIKey),
		"anthropic_key":         redact(c.AnthropicKey),
	}
}

// MustGet gets a config value or returns an error if not set
func MustGet(key string) (string, error) {
	val, err := Get(key)
	if err != nil {
		return "", err
	}
	if val == "" {
		return "", errors.New("config key not set: " + key + " (use: pocket config set " + key + " <value>)")
	}
	return val, nil
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "-", "_"))
}
