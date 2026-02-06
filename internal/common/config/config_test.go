package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// resetConfig resets the config singleton state for isolated tests
func resetConfig(path string) {
	configOnce = sync.Once{}
	configPath = ""
	os.Setenv("POCKET_CONFIG", path)
}

func setupTempConfig(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "config.json")
	resetConfig(tmpPath)
	t.Cleanup(func() {
		os.Unsetenv("POCKET_CONFIG")
		configOnce = sync.Once{}
		configPath = ""
	})
	return tmpPath
}

func TestPathUsesEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "custom.json")
	resetConfig(tmpPath)
	defer func() {
		os.Unsetenv("POCKET_CONFIG")
		configOnce = sync.Once{}
		configPath = ""
	}()

	result := Path()
	if result != tmpPath {
		t.Errorf("expected %s, got %s", tmpPath, result)
	}
}

func TestLoadReturnsEmptyConfigWhenNoFile(t *testing.T) {
	setupTempConfig(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.GitHubToken != "" {
		t.Error("expected empty github_token")
	}
}

func TestSaveAndLoad(t *testing.T) {
	setupTempConfig(t)

	cfg := &Config{
		GitHubToken:  "gh_test_token_123",
		EmailAddress: "test@example.com",
		SlackToken:   "xoxb-test",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.GitHubToken != "gh_test_token_123" {
		t.Errorf("expected gh_test_token_123, got %s", loaded.GitHubToken)
	}
	if loaded.EmailAddress != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", loaded.EmailAddress)
	}
	if loaded.SlackToken != "xoxb-test" {
		t.Errorf("expected xoxb-test, got %s", loaded.SlackToken)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "sub", "dir", "config.json")
	resetConfig(nested)
	defer func() {
		os.Unsetenv("POCKET_CONFIG")
		configOnce = sync.Once{}
		configPath = ""
	}()

	cfg := &Config{GitHubToken: "test"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save should create nested dirs: %v", err)
	}

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("expected config file to exist")
	}
}

func TestSaveFilePermissions(t *testing.T) {
	path := setupTempConfig(t)

	cfg := &Config{GitHubToken: "secret"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}
}

func TestSetAndGet(t *testing.T) {
	setupTempConfig(t)

	tests := []struct {
		key   string
		value string
	}{
		{"github_token", "gh_abc123"},
		{"slack_token", "xoxb-slack"},
		{"email_address", "user@test.com"},
		{"email_password", "secret123"},
		{"imap_server", "imap.test.com"},
		{"smtp_server", "smtp.test.com"},
		{"notion_token", "notion_abc"},
		{"discord_token", "discord_xyz"},
		{"linear_token", "lin_tok"},
		{"newsapi_key", "news_key"},
		{"todoist_token", "todoist_tok"},
		{"twitter_api_key", "tw_key"},
		{"twitter_api_secret", "tw_secret"},
		{"twitter_access_token", "tw_at"},
		{"twitter_access_secret", "tw_as"},
		{"reddit_client_id", "reddit_id"},
		{"reddit_client_secret", "reddit_secret"},
		{"mastodon_server", "mastodon.social"},
		{"mastodon_token", "mast_tok"},
		{"youtube_api_key", "yt_key"},
		{"telegram_token", "tg_tok"},
		{"gitlab_token", "gl_tok"},
		{"jira_url", "https://jira.test.com"},
		{"jira_email", "jira@test.com"},
		{"jira_token", "jira_tok"},
		{"vercel_token", "vc_tok"},
		{"cloudflare_token", "cf_tok"},
		{"trello_key", "trello_k"},
		{"trello_token", "trello_t"},
		{"google_cred_path", "/path/to/cred"},
		{"alphavantage_key", "av_key"},
		{"pushover_token", "po_tok"},
		{"pushover_user", "po_user"},
		{"logseq_graph", "/path/graph"},
		{"logseq_graphs", "a,b,c"},
		{"logseq_format", "markdown"},
		{"obsidian_vault", "/vault"},
		{"obsidian_vaults", "v1,v2"},
		{"obsidian_daily_format", "YYYY-MM-DD"},
		{"twilio_sid", "tw_sid"},
		{"twilio_token", "tw_tok"},
		{"twilio_phone", "+1234567890"},
		{"imap_port", "993"},
		{"smtp_port", "587"},
	}

	for _, tt := range tests {
		if err := Set(tt.key, tt.value); err != nil {
			t.Errorf("Set(%s) failed: %v", tt.key, err)
			continue
		}

		got, err := Get(tt.key)
		if err != nil {
			t.Errorf("Get(%s) failed: %v", tt.key, err)
			continue
		}
		if got != tt.value {
			t.Errorf("Get(%s) = %q, want %q", tt.key, got, tt.value)
		}
	}
}

func TestGetUnknownKey(t *testing.T) {
	setupTempConfig(t)

	_, err := Get("nonexistent_key")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestSetUnknownKey(t *testing.T) {
	setupTempConfig(t)

	err := Set("nonexistent_key", "value")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestMustGetEmpty(t *testing.T) {
	setupTempConfig(t)

	_, err := MustGet("github_token")
	if err == nil {
		t.Error("expected error for unset key")
	}
}

func TestMustGetSet(t *testing.T) {
	setupTempConfig(t)

	Set("github_token", "test_value")

	val, err := MustGet("github_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "test_value" {
		t.Errorf("expected test_value, got %s", val)
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github_token", "github_token"},
		{"GITHUB_TOKEN", "github_token"},
		{"GitHub-Token", "github_token"},
		{"github-token", "github_token"},
		{"SLACK-TOKEN", "slack_token"},
	}

	for _, tt := range tests {
		got := normalizeKey(tt.input)
		if got != tt.want {
			t.Errorf("normalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSetWithNormalizedKey(t *testing.T) {
	setupTempConfig(t)

	// Set with hyphenated key
	if err := Set("github-token", "normalized_test"); err != nil {
		t.Fatalf("Set with hyphen failed: %v", err)
	}

	// Get with underscored key
	val, err := Get("github_token")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "normalized_test" {
		t.Errorf("expected normalized_test, got %s", val)
	}
}

func TestRedacted(t *testing.T) {
	cfg := &Config{
		GitHubToken:  "ghp_1234567890abcdef",
		EmailAddress: "user@example.com",
		SlackToken:   "xoxb-12345678",
		DiscordToken: "short",
		NotionToken:  "",
	}

	redacted := cfg.Redacted()

	// Long tokens should be partially masked
	if redacted["github_token"] == cfg.GitHubToken {
		t.Error("github_token should be redacted")
	}
	if redacted["github_token"] != "ghp_****cdef" {
		t.Errorf("expected ghp_****cdef, got %s", redacted["github_token"])
	}

	// Non-sensitive fields should not be redacted
	if redacted["email_address"] != "user@example.com" {
		t.Errorf("email_address should not be redacted, got %s", redacted["email_address"])
	}

	// Short tokens should be fully masked
	if redacted["discord_token"] != "****" {
		t.Errorf("short token should be ****, got %s", redacted["discord_token"])
	}

	// Empty values should show "(not set)"
	if redacted["notion_token"] != "(not set)" {
		t.Errorf("empty token should be (not set), got %s", redacted["notion_token"])
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := setupTempConfig(t)

	// Write invalid JSON
	os.WriteFile(path, []byte("not valid json{{{"), 0600)

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSetOverwritesExisting(t *testing.T) {
	setupTempConfig(t)

	Set("github_token", "first")
	Set("github_token", "second")

	val, _ := Get("github_token")
	if val != "second" {
		t.Errorf("expected 'second', got %s", val)
	}
}

func TestSetPreservesOtherKeys(t *testing.T) {
	setupTempConfig(t)

	Set("github_token", "gh_value")
	Set("slack_token", "slack_value")

	gh, _ := Get("github_token")
	sl, _ := Get("slack_token")

	if gh != "gh_value" {
		t.Errorf("github_token was overwritten: %s", gh)
	}
	if sl != "slack_value" {
		t.Errorf("slack_token not set: %s", sl)
	}
}
