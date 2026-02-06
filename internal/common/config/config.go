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
	YouTubeAPIKey       string `json:"youtube_api_key,omitempty"`

	// Communication
	SlackToken    string `json:"slack_token,omitempty"`
	DiscordToken  string `json:"discord_token,omitempty"`
	TelegramToken string `json:"telegram_token,omitempty"`
	TwilioSID     string `json:"twilio_sid,omitempty"`
	TwilioToken   string `json:"twilio_token,omitempty"`
	TwilioPhone   string `json:"twilio_phone,omitempty"`

	// Email (IMAP/SMTP)
	EmailAddress  string `json:"email_address,omitempty"`
	EmailPassword string `json:"email_password,omitempty"`
	IMAPServer    string `json:"imap_server,omitempty"`
	IMAPPort      string `json:"imap_port,omitempty"`
	SMTPServer    string `json:"smtp_server,omitempty"`
	SMTPPort      string `json:"smtp_port,omitempty"`

	// Dev
	GitHubToken     string `json:"github_token,omitempty"`
	GitLabToken     string `json:"gitlab_token,omitempty"`
	LinearToken     string `json:"linear_token,omitempty"`
	JiraURL         string `json:"jira_url,omitempty"`
	JiraEmail       string `json:"jira_email,omitempty"`
	JiraToken       string `json:"jira_token,omitempty"`
	VercelToken     string `json:"vercel_token,omitempty"`
	CloudflareToken string `json:"cloudflare_token,omitempty"`

	// Productivity
	NotionToken    string `json:"notion_token,omitempty"`
	TodoistToken   string `json:"todoist_token,omitempty"`
	TrelloKey      string `json:"trello_key,omitempty"`
	TrelloToken    string `json:"trello_token,omitempty"`
	GoogleCredPath string `json:"google_cred_path,omitempty"`

	// News
	NewsAPIKey string `json:"newsapi_key,omitempty"`

	// Utility
	AlphaVantageKey string `json:"alphavantage_key,omitempty"`

	// Push Notifications
	PushoverToken string `json:"pushover_token,omitempty"`
	PushoverUser  string `json:"pushover_user,omitempty"`

	// Logseq
	LogseqGraph  string `json:"logseq_graph,omitempty"`
	LogseqGraphs string `json:"logseq_graphs,omitempty"`
	LogseqFormat string `json:"logseq_format,omitempty"`

	// Obsidian
	ObsidianVault       string `json:"obsidian_vault,omitempty"`
	ObsidianVaults      string `json:"obsidian_vaults,omitempty"`
	ObsidianDailyFormat string `json:"obsidian_daily_format,omitempty"`
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
	case "youtube_api_key":
		cfg.YouTubeAPIKey = value
	case "slack_token":
		cfg.SlackToken = value
	case "discord_token":
		cfg.DiscordToken = value
	case "telegram_token":
		cfg.TelegramToken = value
	case "twilio_sid":
		cfg.TwilioSID = value
	case "twilio_token":
		cfg.TwilioToken = value
	case "twilio_phone":
		cfg.TwilioPhone = value
	case "email_address":
		cfg.EmailAddress = value
	case "email_password":
		cfg.EmailPassword = value
	case "imap_server":
		cfg.IMAPServer = value
	case "imap_port":
		cfg.IMAPPort = value
	case "smtp_server":
		cfg.SMTPServer = value
	case "smtp_port":
		cfg.SMTPPort = value
	case "github_token":
		cfg.GitHubToken = value
	case "gitlab_token":
		cfg.GitLabToken = value
	case "linear_token":
		cfg.LinearToken = value
	case "jira_url":
		cfg.JiraURL = value
	case "jira_email":
		cfg.JiraEmail = value
	case "jira_token":
		cfg.JiraToken = value
	case "vercel_token":
		cfg.VercelToken = value
	case "cloudflare_token":
		cfg.CloudflareToken = value
	case "notion_token":
		cfg.NotionToken = value
	case "todoist_token":
		cfg.TodoistToken = value
	case "trello_key":
		cfg.TrelloKey = value
	case "trello_token":
		cfg.TrelloToken = value
	case "google_cred_path":
		cfg.GoogleCredPath = value
	case "newsapi_key":
		cfg.NewsAPIKey = value
	case "alphavantage_key":
		cfg.AlphaVantageKey = value
	case "pushover_token":
		cfg.PushoverToken = value
	case "pushover_user":
		cfg.PushoverUser = value
	case "logseq_graph":
		cfg.LogseqGraph = value
	case "logseq_graphs":
		cfg.LogseqGraphs = value
	case "logseq_format":
		cfg.LogseqFormat = value
	case "obsidian_vault":
		cfg.ObsidianVault = value
	case "obsidian_vaults":
		cfg.ObsidianVaults = value
	case "obsidian_daily_format":
		cfg.ObsidianDailyFormat = value
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
	case "youtube_api_key":
		return cfg.YouTubeAPIKey, nil
	case "slack_token":
		return cfg.SlackToken, nil
	case "discord_token":
		return cfg.DiscordToken, nil
	case "telegram_token":
		return cfg.TelegramToken, nil
	case "twilio_sid":
		return cfg.TwilioSID, nil
	case "twilio_token":
		return cfg.TwilioToken, nil
	case "twilio_phone":
		return cfg.TwilioPhone, nil
	case "email_address":
		return cfg.EmailAddress, nil
	case "email_password":
		return cfg.EmailPassword, nil
	case "imap_server":
		return cfg.IMAPServer, nil
	case "imap_port":
		return cfg.IMAPPort, nil
	case "smtp_server":
		return cfg.SMTPServer, nil
	case "smtp_port":
		return cfg.SMTPPort, nil
	case "github_token":
		return cfg.GitHubToken, nil
	case "gitlab_token":
		return cfg.GitLabToken, nil
	case "linear_token":
		return cfg.LinearToken, nil
	case "jira_url":
		return cfg.JiraURL, nil
	case "jira_email":
		return cfg.JiraEmail, nil
	case "jira_token":
		return cfg.JiraToken, nil
	case "vercel_token":
		return cfg.VercelToken, nil
	case "cloudflare_token":
		return cfg.CloudflareToken, nil
	case "notion_token":
		return cfg.NotionToken, nil
	case "todoist_token":
		return cfg.TodoistToken, nil
	case "trello_key":
		return cfg.TrelloKey, nil
	case "trello_token":
		return cfg.TrelloToken, nil
	case "google_cred_path":
		return cfg.GoogleCredPath, nil
	case "newsapi_key":
		return cfg.NewsAPIKey, nil
	case "alphavantage_key":
		return cfg.AlphaVantageKey, nil
	case "pushover_token":
		return cfg.PushoverToken, nil
	case "pushover_user":
		return cfg.PushoverUser, nil
	case "logseq_graph":
		return cfg.LogseqGraph, nil
	case "logseq_graphs":
		return cfg.LogseqGraphs, nil
	case "logseq_format":
		return cfg.LogseqFormat, nil
	case "obsidian_vault":
		return cfg.ObsidianVault, nil
	case "obsidian_vaults":
		return cfg.ObsidianVaults, nil
	case "obsidian_daily_format":
		return cfg.ObsidianDailyFormat, nil
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
		"youtube_api_key":       redact(c.YouTubeAPIKey),
		"slack_token":           redact(c.SlackToken),
		"discord_token":         redact(c.DiscordToken),
		"telegram_token":        redact(c.TelegramToken),
		"twilio_sid":            redact(c.TwilioSID),
		"twilio_token":          redact(c.TwilioToken),
		"twilio_phone":          c.TwilioPhone,
		"email_address":         c.EmailAddress,
		"email_password":        redact(c.EmailPassword),
		"imap_server":           c.IMAPServer,
		"imap_port":             c.IMAPPort,
		"smtp_server":           c.SMTPServer,
		"smtp_port":             c.SMTPPort,
		"github_token":          redact(c.GitHubToken),
		"gitlab_token":          redact(c.GitLabToken),
		"linear_token":          redact(c.LinearToken),
		"jira_url":              c.JiraURL,
		"jira_email":            c.JiraEmail,
		"jira_token":            redact(c.JiraToken),
		"vercel_token":          redact(c.VercelToken),
		"cloudflare_token":      redact(c.CloudflareToken),
		"notion_token":          redact(c.NotionToken),
		"todoist_token":         redact(c.TodoistToken),
		"trello_key":            redact(c.TrelloKey),
		"trello_token":          redact(c.TrelloToken),
		"google_cred_path":      c.GoogleCredPath,
		"newsapi_key":           redact(c.NewsAPIKey),
		"alphavantage_key":      redact(c.AlphaVantageKey),
		"pushover_token":        redact(c.PushoverToken),
		"pushover_user":         redact(c.PushoverUser),
		"logseq_graph":          c.LogseqGraph,
		"logseq_graphs":         c.LogseqGraphs,
		"logseq_format":         c.LogseqFormat,
		"obsidian_vault":        c.ObsidianVault,
		"obsidian_vaults":       c.ObsidianVaults,
		"obsidian_daily_format": c.ObsidianDailyFormat,
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
