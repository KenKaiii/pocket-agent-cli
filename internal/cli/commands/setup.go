package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// ServiceInfo describes what's needed to set up a service
type ServiceInfo struct {
	Service     string   `json:"service"`
	Name        string   `json:"name"`
	Status      string   `json:"status"` // "ready", "missing", "partial"
	Keys        []KeyInfo `json:"keys"`
	SetupGuide  string   `json:"setup_guide"`
	TestCommand string   `json:"test_cmd,omitempty"`
}

// KeyInfo describes a single credential key
type KeyInfo struct {
	Key         string `json:"key"`
	Description string `json:"desc"`
	Required    bool   `json:"required"`
	Set         bool   `json:"set"`
	Example     string `json:"example,omitempty"`
}

// ServiceStatus is a compact status for listing
type ServiceStatus struct {
	Service string `json:"service"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Missing int    `json:"missing,omitempty"`
}

var services = map[string]ServiceInfo{
	"github": {
		Service: "github",
		Name:    "GitHub",
		Keys: []KeyInfo{
			{
				Key:         "github_token",
				Description: "Personal access token with repo, read:org, notifications scopes",
				Required:    true,
				Example:     "ghp_xxxxxxxxxxxxxxxxxxxx",
			},
		},
		SetupGuide:  "1. Go to https://github.com/settings/tokens\n2. Click 'Generate new token (classic)'\n3. Select scopes: repo, read:org, notifications\n4. Generate and copy the token\n5. Run: pocket config set github_token <your-token>",
		TestCommand: "pocket dev github repos -l 1",
	},
	"gitlab": {
		Service: "gitlab",
		Name:    "GitLab",
		Keys: []KeyInfo{
			{
				Key:         "gitlab_token",
				Description: "Personal access token with api scope",
				Required:    true,
				Example:     "glpat-xxxxxxxxxxxxxxxxxxxx",
			},
		},
		SetupGuide:  "1. Go to https://gitlab.com/-/user_settings/personal_access_tokens\n2. Create token with 'api' scope\n3. Copy the token\n4. Run: pocket config set gitlab_token <your-token>",
		TestCommand: "pocket dev gitlab projects -l 1",
	},
	"twitter": {
		Service: "twitter",
		Name:    "Twitter/X",
		Keys: []KeyInfo{
			{Key: "twitter_api_key", Description: "API Key (Consumer Key)", Required: true},
			{Key: "twitter_api_secret", Description: "API Secret (Consumer Secret)", Required: true},
			{Key: "twitter_access_token", Description: "Access Token", Required: true},
			{Key: "twitter_access_secret", Description: "Access Token Secret", Required: true},
		},
		SetupGuide:  "1. Go to https://developer.twitter.com/en/portal/dashboard\n2. Create a project and app\n3. Generate Consumer Keys and Access Tokens\n4. Run:\n   pocket config set twitter_api_key <key>\n   pocket config set twitter_api_secret <secret>\n   pocket config set twitter_access_token <token>\n   pocket config set twitter_access_secret <secret>",
		TestCommand: "pocket social twitter timeline -l 1",
	},
	"reddit": {
		Service: "reddit",
		Name:    "Reddit",
		Keys: []KeyInfo{
			{Key: "reddit_client_id", Description: "OAuth Client ID", Required: true},
			{Key: "reddit_client_secret", Description: "OAuth Client Secret", Required: true},
		},
		SetupGuide:  "1. Go to https://www.reddit.com/prefs/apps\n2. Create an app (script type)\n3. Copy the client ID (under app name) and secret\n4. Run:\n   pocket config set reddit_client_id <id>\n   pocket config set reddit_client_secret <secret>",
		TestCommand: "pocket social reddit feed -l 1",
	},
	"slack": {
		Service: "slack",
		Name:    "Slack",
		Keys: []KeyInfo{
			{Key: "slack_token", Description: "Bot or User OAuth Token (xoxb-* or xoxp-*)", Required: true, Example: "xoxb-xxxx-xxxx-xxxx"},
		},
		SetupGuide:  "1. Go to https://api.slack.com/apps\n2. Create an app or select existing\n3. Go to OAuth & Permissions\n4. Add scopes: channels:read, chat:write, users:read\n5. Install to workspace and copy Bot Token\n6. Run: pocket config set slack_token <token>",
		TestCommand: "pocket comms slack channels",
	},
	"discord": {
		Service: "discord",
		Name:    "Discord",
		Keys: []KeyInfo{
			{Key: "discord_token", Description: "Bot token", Required: true},
		},
		SetupGuide:  "1. Go to https://discord.com/developers/applications\n2. Create application, then create Bot\n3. Copy the bot token\n4. Run: pocket config set discord_token <token>",
		TestCommand: "pocket comms discord guilds",
	},
	"telegram": {
		Service: "telegram",
		Name:    "Telegram",
		Keys: []KeyInfo{
			{Key: "telegram_token", Description: "Bot token from @BotFather", Required: true, Example: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"},
		},
		SetupGuide:  "1. Message @BotFather on Telegram\n2. Send /newbot and follow instructions\n3. Copy the token provided\n4. Run: pocket config set telegram_token <token>",
		TestCommand: "pocket comms telegram chats",
	},
	"gmail": {
		Service: "gmail",
		Name:    "Gmail",
		Keys: []KeyInfo{
			{Key: "gmail_cred_path", Description: "Path to OAuth credentials JSON file", Required: true, Example: "~/.config/pocket/gmail_credentials.json"},
		},
		SetupGuide:  "1. Go to https://console.cloud.google.com/\n2. Create project, enable Gmail API\n3. Create OAuth 2.0 credentials (Desktop app)\n4. Download JSON file\n5. Run: pocket config set gmail_cred_path /path/to/credentials.json",
		TestCommand: "pocket comms email list -l 1",
	},
	"google": {
		Service: "google",
		Name:    "Google (Calendar)",
		Keys: []KeyInfo{
			{Key: "google_cred_path", Description: "Path to OAuth credentials JSON file", Required: true},
		},
		SetupGuide:  "1. Go to https://console.cloud.google.com/\n2. Create project, enable Calendar API\n3. Create OAuth 2.0 credentials (Desktop app)\n4. Download JSON file\n5. Run: pocket config set google_cred_path /path/to/credentials.json",
		TestCommand: "pocket productivity calendar today",
	},
	"notion": {
		Service: "notion",
		Name:    "Notion",
		Keys: []KeyInfo{
			{Key: "notion_token", Description: "Internal integration token", Required: true, Example: "secret_xxxx"},
		},
		SetupGuide:  "1. Go to https://www.notion.so/my-integrations\n2. Create new integration\n3. Copy the Internal Integration Token\n4. Share your pages/databases with the integration\n5. Run: pocket config set notion_token <token>",
		TestCommand: "pocket productivity notion search test",
	},
	"todoist": {
		Service: "todoist",
		Name:    "Todoist",
		Keys: []KeyInfo{
			{Key: "todoist_token", Description: "API token", Required: true},
		},
		SetupGuide:  "1. Go to https://todoist.com/app/settings/integrations/developer\n2. Copy your API token\n3. Run: pocket config set todoist_token <token>",
		TestCommand: "pocket productivity todoist projects",
	},
	"linear": {
		Service: "linear",
		Name:    "Linear",
		Keys: []KeyInfo{
			{Key: "linear_token", Description: "Personal API key", Required: true, Example: "lin_api_xxxx"},
		},
		SetupGuide:  "1. Go to https://linear.app/settings/api\n2. Create a personal API key\n3. Copy the key\n4. Run: pocket config set linear_token <token>",
		TestCommand: "pocket dev linear teams",
	},
	"openai": {
		Service: "openai",
		Name:    "OpenAI",
		Keys: []KeyInfo{
			{Key: "openai_key", Description: "API key", Required: true, Example: "sk-xxxx"},
		},
		SetupGuide:  "1. Go to https://platform.openai.com/api-keys\n2. Create new secret key\n3. Copy the key\n4. Run: pocket config set openai_key <key>",
		TestCommand: "pocket ai openai models",
	},
	"anthropic": {
		Service: "anthropic",
		Name:    "Anthropic",
		Keys: []KeyInfo{
			{Key: "anthropic_key", Description: "API key", Required: true, Example: "sk-ant-xxxx"},
		},
		SetupGuide:  "1. Go to https://console.anthropic.com/settings/keys\n2. Create new API key\n3. Copy the key\n4. Run: pocket config set anthropic_key <key>",
		TestCommand: "pocket ai anthropic chat hello",
	},
	"newsapi": {
		Service: "newsapi",
		Name:    "NewsAPI",
		Keys: []KeyInfo{
			{Key: "newsapi_key", Description: "API key", Required: true},
		},
		SetupGuide:  "1. Go to https://newsapi.org/register\n2. Register for free account\n3. Copy your API key\n4. Run: pocket config set newsapi_key <key>",
		TestCommand: "pocket news newsapi headlines -l 1",
	},
	"mastodon": {
		Service: "mastodon",
		Name:    "Mastodon",
		Keys: []KeyInfo{
			{Key: "mastodon_server", Description: "Server URL (e.g., mastodon.social)", Required: true, Example: "mastodon.social"},
			{Key: "mastodon_token", Description: "Access token", Required: true},
		},
		SetupGuide:  "1. Go to your Mastodon instance's settings\n2. Development > New Application\n3. Create app with read/write scopes\n4. Copy the access token\n5. Run:\n   pocket config set mastodon_server <server>\n   pocket config set mastodon_token <token>",
		TestCommand: "pocket social mastodon timeline -l 1",
	},
}

func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "setup",
		Aliases: []string{"onboard"},
		Short:   "Service setup and onboarding",
	}

	cmd.AddCommand(newSetupListCmd())
	cmd.AddCommand(newSetupShowCmd())
	cmd.AddCommand(newSetupSetCmd())

	return cmd
}

func newSetupListCmd() *cobra.Command {
	var showAll bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all services and their setup status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			result := make([]ServiceStatus, 0)
			for _, svc := range services {
				status := getServiceStatus(cfg, svc)
				if showAll || status.Status != "ready" {
					result = append(result, status)
				}
			}

			// Sort: missing first, then partial, then ready
			sortedResult := make([]ServiceStatus, 0, len(result))
			for _, s := range result {
				if s.Status == "missing" {
					sortedResult = append(sortedResult, s)
				}
			}
			for _, s := range result {
				if s.Status == "partial" {
					sortedResult = append(sortedResult, s)
				}
			}
			for _, s := range result {
				if s.Status == "ready" {
					sortedResult = append(sortedResult, s)
				}
			}

			return output.Print(sortedResult)
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all services including configured ones")

	return cmd
}

func newSetupShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [service]",
		Short: "Show setup instructions for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, ok := services[args[0]]
			if !ok {
				return output.PrintError("unknown_service", "Unknown service: "+args[0], nil)
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			// Update key status
			for i := range svc.Keys {
				val, _ := config.Get(svc.Keys[i].Key)
				svc.Keys[i].Set = val != ""
			}

			// Update service status
			status := getServiceStatus(cfg, svc)
			svc.Status = status.Status

			return output.Print(svc)
		},
	}

	return cmd
}

func newSetupSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [service] [key] [value]",
		Short: "Set a credential for a service",
		Long:  "Set a credential. Use 'pocket setup show <service>' to see required keys.",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			service := args[0]

			svc, ok := services[service]
			if !ok {
				return output.PrintError("unknown_service", "Unknown service: "+service, nil)
			}

			// If only 2 args, it's "service value" for single-key services
			var key, value string
			if len(args) == 2 {
				// Find the single required key
				if len(svc.Keys) == 1 {
					key = svc.Keys[0].Key
					value = args[1]
				} else {
					return output.PrintError("key_required", "Service has multiple keys, specify which key to set", map[string]any{
						"keys": svc.Keys,
					})
				}
			} else {
				key = args[1]
				value = args[2]
			}

			// Validate key belongs to service
			validKey := false
			for _, k := range svc.Keys {
				if k.Key == key {
					validKey = true
					break
				}
			}
			if !validKey {
				return output.PrintError("invalid_key", "Key '"+key+"' is not valid for service '"+service+"'", map[string]any{
					"valid_keys": svc.Keys,
				})
			}

			// Set the value
			if err := config.Set(key, value); err != nil {
				return output.PrintError("set_failed", err.Error(), nil)
			}

			// Check new status
			cfg, _ := config.Load()
			status := getServiceStatus(cfg, svc)

			return output.Print(map[string]any{
				"status":         "saved",
				"service":        service,
				"key":            key,
				"service_status": status.Status,
				"test_cmd":       svc.TestCommand,
			})
		},
	}

	return cmd
}

func getServiceStatus(cfg *config.Config, svc ServiceInfo) ServiceStatus {
	missing := 0
	for _, k := range svc.Keys {
		val, _ := config.Get(k.Key)
		if val == "" && k.Required {
			missing++
		}
	}

	status := "ready"
	if missing == len(svc.Keys) {
		status = "missing"
	} else if missing > 0 {
		status = "partial"
	}

	return ServiceStatus{
		Service: svc.Service,
		Name:    svc.Name,
		Status:  status,
		Missing: missing,
	}
}
