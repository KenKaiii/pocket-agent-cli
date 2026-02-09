package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// ServiceInfo describes what's needed to set up a service
type ServiceInfo struct {
	Service     string    `json:"service"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // "ready", "missing", "partial"
	Keys        []KeyInfo `json:"keys"`
	SetupGuide  string    `json:"setup_guide"`
	TestCommand string    `json:"test_cmd,omitempty"`
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
			{Key: "x_client_id", Description: "OAuth 2.0 Client ID", Required: true},
		},
		SetupGuide:  "1. Go to https://developer.x.com/en/portal/dashboard\n2. Create a project and app (Free tier works)\n3. Enable OAuth 2.0, select 'Native App' type\n4. Set callback URL to http://127.0.0.1:8765/callback\n5. Copy the Client ID\n6. Run: pocket config set x_client_id <client-id>\n7. Run: pocket social twitter auth\n\nNote: Free tier allows posting tweets. Reading requires paid tier ($200/mo).",
		TestCommand: "pocket social twitter me",
	},
	"reddit": {
		Service: "reddit",
		Name:    "Reddit",
		Keys: []KeyInfo{
			{Key: "reddit_client_id", Description: "OAuth Client ID (installed app)", Required: true},
		},
		SetupGuide:  "1. Go to https://www.reddit.com/prefs/apps\n2. Click 'create another app' at the bottom\n3. Select 'installed app' type, name it\n4. Set redirect URI to http://localhost:8766/callback\n5. Copy the client ID (shown under app name)\n6. Run: pocket config set reddit_client_id <id>\n7. Run: pocket social reddit auth\n\nNote: Free tier allows 100 req/min for non-commercial use.",
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
	"twilio": {
		Service: "twilio",
		Name:    "Twilio (SMS)",
		Keys: []KeyInfo{
			{Key: "twilio_sid", Description: "Account SID", Required: true, Example: "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
			{Key: "twilio_token", Description: "Auth Token", Required: true},
			{Key: "twilio_phone", Description: "Your Twilio phone number", Required: true, Example: "+15551234567"},
		},
		SetupGuide: `1. Sign up at https://www.twilio.com/try-twilio
2. Go to Console Dashboard: https://console.twilio.com/
3. Copy your Account SID and Auth Token from the dashboard
4. Get a phone number from the Phone Numbers section
5. Run these commands:
   pocket config set twilio_sid <your-account-sid>
   pocket config set twilio_token <your-auth-token>
   pocket config set twilio_phone <your-twilio-phone-number>

Note: Phone numbers must be in E.164 format (e.g., +15551234567)
Free trial accounts can only send to verified phone numbers.`,
		TestCommand: "pocket comms twilio account",
	},
	"email": {
		Service: "email",
		Name:    "Email (IMAP/SMTP)",
		Keys: []KeyInfo{
			{Key: "email_address", Description: "Your email address", Required: true, Example: "you@gmail.com"},
			{Key: "email_password", Description: "App password (not regular password)", Required: true, Example: "xxxx xxxx xxxx xxxx"},
			{Key: "imap_server", Description: "IMAP server hostname", Required: true, Example: "imap.gmail.com"},
			{Key: "smtp_server", Description: "SMTP server hostname", Required: true, Example: "smtp.gmail.com"},
			{Key: "imap_port", Description: "IMAP port (default: 993)", Required: false, Example: "993"},
			{Key: "smtp_port", Description: "SMTP port (default: 587)", Required: false, Example: "587"},
		},
		SetupGuide: `For Gmail:
1. Enable 2-Factor Authentication at https://myaccount.google.com/security
2. Go to https://myaccount.google.com/apppasswords
3. Create an app password (select 'Mail' and your device)
4. Run these commands:
   pocket config set email_address your@gmail.com
   pocket config set email_password "xxxx xxxx xxxx xxxx"
   pocket config set imap_server imap.gmail.com
   pocket config set smtp_server smtp.gmail.com

For Outlook/Hotmail:
   pocket config set imap_server outlook.office365.com
   pocket config set smtp_server smtp.office365.com

For Yahoo:
   pocket config set imap_server imap.mail.yahoo.com
   pocket config set smtp_server smtp.mail.yahoo.com`,
		TestCommand: "pocket comms email list -l 1",
	},
	"calendar": {
		Service: "calendar",
		Name:    "Google Calendar",
		Keys: []KeyInfo{
			{Key: "google_client_id", Description: "OAuth 2.0 Client ID", Required: true, Example: "123456789.apps.googleusercontent.com"},
			{Key: "google_client_secret", Description: "OAuth 2.0 Client Secret", Required: true},
			{Key: "google_refresh_token", Description: "OAuth 2.0 Refresh Token", Required: true},
		},
		SetupGuide: `1. Go to https://console.cloud.google.com/
2. Create a new project (or select existing)
3. Enable "Google Calendar API" at:
   https://console.cloud.google.com/apis/library/calendar-json.googleapis.com
4. Go to Credentials > Create Credentials > OAuth client ID
5. Select "Desktop app" as application type
6. Copy Client ID and Client Secret
7. Get a refresh token using OAuth Playground:
   a. Go to https://developers.google.com/oauthplayground/
   b. Click gear icon, check "Use your own OAuth credentials"
   c. Enter your Client ID and Client Secret
   d. In Step 1, select "Google Calendar API v3" scopes
   e. Authorize and exchange for tokens
   f. Copy the Refresh Token from Step 2
8. Run:
   pocket config set google_client_id <your-client-id>
   pocket config set google_client_secret <your-client-secret>
   pocket config set google_refresh_token <your-refresh-token>`,
		TestCommand: "pocket productivity calendar today",
	},
	"notion": {
		Service: "notion",
		Name:    "Notion",
		Keys: []KeyInfo{
			{Key: "notion_token", Description: "Internal integration token", Required: true, Example: "ntn_xxxx"},
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
	"youtube": {
		Service: "youtube",
		Name:    "YouTube",
		Keys: []KeyInfo{
			{Key: "youtube_api_key", Description: "YouTube Data API v3 key", Required: true, Example: "AIzaSy..."},
		},
		SetupGuide: `1. Go to https://console.cloud.google.com/
2. Create a new project (or select existing)
3. Enable "YouTube Data API v3" at:
   https://console.cloud.google.com/apis/library/youtube.googleapis.com
4. Go to Credentials > Create Credentials > API Key
5. (Optional) Restrict key to YouTube Data API v3
6. Copy the API key
7. Run: pocket config set youtube_api_key <your-api-key>

Note: Free tier allows ~10,000 units/day (search=100, video=1, channel=1)`,
		TestCommand: "pocket social youtube trending -l 1",
	},
	"alphavantage": {
		Service: "alphavantage",
		Name:    "Alpha Vantage (Stocks)",
		Keys: []KeyInfo{
			{Key: "alphavantage_key", Description: "API key for stock market data", Required: true},
		},
		SetupGuide: `1. Go to https://www.alphavantage.co/support/#api-key
2. Enter your email to get a free API key
3. Copy the API key from email
4. Run: pocket config set alphavantage_key <your-api-key>

Note: Free tier allows 25 requests/day, 5/min`,
		TestCommand: "pocket utility stocks quote AAPL",
	},
	"jira": {
		Service: "jira",
		Name:    "Jira",
		Keys: []KeyInfo{
			{Key: "jira_url", Description: "Jira instance URL", Required: true, Example: "https://mycompany.atlassian.net"},
			{Key: "jira_email", Description: "Your Atlassian email", Required: true, Example: "you@company.com"},
			{Key: "jira_token", Description: "API token", Required: true},
		},
		SetupGuide: `1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Give it a label and create
4. Copy the token
5. Run these commands:
   pocket config set jira_url https://yourcompany.atlassian.net
   pocket config set jira_email your@email.com
   pocket config set jira_token <your-api-token>`,
		TestCommand: "pocket dev jira projects",
	},
	"cloudflare": {
		Service: "cloudflare",
		Name:    "Cloudflare",
		Keys: []KeyInfo{
			{Key: "cloudflare_token", Description: "API token with Zone permissions", Required: true},
		},
		SetupGuide: `1. Go to https://dash.cloudflare.com/profile/api-tokens
2. Click "Create Token"
3. Use template "Edit zone DNS" or create custom with:
   - Zone:DNS:Edit, Zone:Zone:Read permissions
4. Copy the token
5. Run: pocket config set cloudflare_token <your-token>`,
		TestCommand: "pocket dev cloudflare zones",
	},
	"vercel": {
		Service: "vercel",
		Name:    "Vercel",
		Keys: []KeyInfo{
			{Key: "vercel_token", Description: "Personal access token", Required: true},
		},
		SetupGuide: `1. Go to https://vercel.com/account/tokens
2. Click "Create"
3. Give it a name and set expiration
4. Copy the token
5. Run: pocket config set vercel_token <your-token>`,
		TestCommand: "pocket dev vercel projects",
	},
	"trello": {
		Service: "trello",
		Name:    "Trello",
		Keys: []KeyInfo{
			{Key: "trello_key", Description: "API Key", Required: true},
			{Key: "trello_token", Description: "API Token", Required: true},
		},
		SetupGuide: `1. Go to https://trello.com/power-ups/admin
2. Click "New" to create a Power-Up (or use existing)
3. Copy the API Key
4. Click "Token" link next to the key to generate a token
5. Authorize and copy the token
6. Run:
   pocket config set trello_key <your-api-key>
   pocket config set trello_token <your-token>`,
		TestCommand: "pocket productivity trello boards",
	},
	"pushover": {
		Service: "pushover",
		Name:    "Pushover",
		Keys: []KeyInfo{
			{Key: "pushover_token", Description: "Application API Token", Required: true, Example: "azGDORePK8gMaC0QOYAMyEEuzJnyUi"},
			{Key: "pushover_user", Description: "User Key", Required: true, Example: "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"},
		},
		SetupGuide: `1. Sign up at https://pushover.net/
2. Install Pushover app on your phone (iOS/Android) or desktop
3. Your User Key is shown on the dashboard after login
4. Create an Application at https://pushover.net/apps/build
5. Copy the Application API Token
6. Run:
   pocket config set pushover_token <your-app-token>
   pocket config set pushover_user <your-user-key>

Note: Pushover has a one-time $5 purchase for each platform.
After setup, use: pocket comms notify pushover "Your message"`,
		TestCommand: "pocket comms notify pushover 'Test notification from Pocket CLI'",
	},
	"obsidian": {
		Service: "obsidian",
		Name:    "Obsidian",
		Keys: []KeyInfo{
			{Key: "obsidian_vault", Description: "Path to default Obsidian vault", Required: true, Example: "/Users/you/Documents/MyVault"},
			{Key: "obsidian_vaults", Description: "JSON array of additional vaults (optional)", Required: false, Example: `[{"name":"work","path":"/path/to/work"}]`},
			{Key: "obsidian_daily_format", Description: "Daily note date format (default: 2006-01-02)", Required: false, Example: "2006-01-02"},
		},
		SetupGuide: `Obsidian works with local markdown vaults. No API key required.

1. Find your Obsidian vault path (the folder containing your .obsidian directory)
2. Run: pocket config set obsidian_vault /path/to/your/vault

Optional - Multiple vaults:
   pocket config set obsidian_vaults '[{"name":"work","path":"/path/to/work"},{"name":"personal","path":"/path/to/personal"}]'

Optional - Custom daily note format (Go date format):
   pocket config set obsidian_daily_format "2006-01-02"

Common daily note formats:
   2006-01-02      -> 2024-01-15
   01-02-2006      -> 01-15-2024
   2006/01/02      -> 2024/01/15
   January 2, 2006 -> January 15, 2024`,
		TestCommand: "pocket productivity obsidian vaults",
	},
	"logseq": {
		Service: "logseq",
		Name:    "Logseq",
		Keys: []KeyInfo{
			{Key: "logseq_graph", Description: "Path to default Logseq graph", Required: true, Example: "/Users/you/Documents/logseq-graph"},
			{Key: "logseq_graphs", Description: "JSON array of additional graphs (optional)", Required: false, Example: `[{"name":"work","path":"/path/to/work","format":"md"}]`},
			{Key: "logseq_format", Description: "File format: md or org (default: md)", Required: false, Example: "md"},
		},
		SetupGuide: `Logseq works with local graphs (markdown/org files). No API key required.

1. Find your Logseq graph path (the folder containing pages/ and journals/ directories)
2. Run: pocket config set logseq_graph /path/to/your/graph

Optional - Set file format (md or org):
   pocket config set logseq_format md

Optional - Multiple graphs:
   pocket config set logseq_graphs '[{"name":"work","path":"/path/to/work","format":"md"},{"name":"personal","path":"/path/to/personal","format":"org"}]'

Graph structure:
   your-graph/
   ├── pages/          # Regular pages
   ├── journals/       # Daily journal entries (YYYY_MM_DD.md)
   └── logseq/         # Logseq config (not used by CLI)

Page names with special characters (/, :, ?) are URL-encoded in filenames.`,
		TestCommand: "pocket productivity logseq graphs",
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
