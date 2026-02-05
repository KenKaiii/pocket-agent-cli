package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// Integration describes an available integration
type Integration struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Group       string   `json:"group"`
	Description string   `json:"desc"`
	AuthNeeded  bool     `json:"auth_needed"`
	Status      string   `json:"status"` // "ready", "needs_setup", "no_auth"
	Commands    []string `json:"commands"`
	SetupCmd    string   `json:"setup_cmd,omitempty"`
}

var allIntegrations = []Integration{
	// News - No Auth
	{
		ID:          "hackernews",
		Name:        "Hacker News",
		Group:       "news",
		Description: "Tech news, stories, and discussions from Hacker News",
		AuthNeeded:  false,
		Commands:    []string{"pocket news hn top", "pocket news hn new", "pocket news hn best", "pocket news hn ask", "pocket news hn show", "pocket news hn item [id]"},
	},
	{
		ID:          "rss",
		Name:        "RSS/Atom Feeds",
		Group:       "news",
		Description: "Fetch and manage RSS/Atom feeds from any source",
		AuthNeeded:  false,
		Commands:    []string{"pocket news feeds fetch [url]", "pocket news feeds list", "pocket news feeds add [url]", "pocket news feeds read [name]", "pocket news feeds remove [name]"},
	},
	{
		ID:          "newsapi",
		Name:        "NewsAPI",
		Group:       "news",
		Description: "Search news articles and get headlines from 80,000+ sources",
		AuthNeeded:  true,
		Commands:    []string{"pocket news newsapi headlines", "pocket news newsapi search [query]", "pocket news newsapi sources"},
		SetupCmd:    "pocket setup show newsapi",
	},

	// Knowledge - No Auth
	{
		ID:          "wikipedia",
		Name:        "Wikipedia",
		Group:       "knowledge",
		Description: "Search and read Wikipedia articles",
		AuthNeeded:  false,
		Commands:    []string{"pocket knowledge wiki search [query]", "pocket knowledge wiki summary [title]", "pocket knowledge wiki article [title]"},
	},
	{
		ID:          "stackexchange",
		Name:        "StackOverflow",
		Group:       "knowledge",
		Description: "Search programming Q&A from StackOverflow and StackExchange sites",
		AuthNeeded:  false,
		Commands:    []string{"pocket knowledge so search [query]", "pocket knowledge so question [id]", "pocket knowledge so answers [id]"},
	},
	{
		ID:          "dictionary",
		Name:        "Dictionary",
		Group:       "knowledge",
		Description: "Word definitions, synonyms, antonyms, and pronunciations",
		AuthNeeded:  false,
		Commands:    []string{"pocket knowledge dict define [word]", "pocket knowledge dict synonyms [word]", "pocket knowledge dict antonyms [word]"},
	},

	// Utility - No Auth
	{
		ID:          "weather",
		Name:        "Weather",
		Group:       "utility",
		Description: "Current weather and forecasts for any location",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility weather now [location]", "pocket utility weather forecast [location]"},
	},

	// Dev - No Auth
	{
		ID:          "npm",
		Name:        "npm Registry",
		Group:       "dev",
		Description: "Search npm packages, get info, versions, and dependencies",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev npm search [query]", "pocket dev npm info [package]", "pocket dev npm versions [package]", "pocket dev npm deps [package]"},
	},
	{
		ID:          "pypi",
		Name:        "PyPI Registry",
		Group:       "dev",
		Description: "Search Python packages, get info, versions, and dependencies",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev pypi search [query]", "pocket dev pypi info [package]", "pocket dev pypi versions [package]", "pocket dev pypi deps [package]"},
	},

	// Dev - Auth Required
	{
		ID:          "github",
		Name:        "GitHub",
		Group:       "dev",
		Description: "Repos, issues, PRs, notifications, and search on GitHub",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev github repos", "pocket dev github repo [owner/name]", "pocket dev github issues", "pocket dev github issue [repo] [num]", "pocket dev github prs -r [repo]", "pocket dev github pr [repo] [num]", "pocket dev github notifications", "pocket dev github search [query]"},
		SetupCmd:    "pocket setup show github",
	},
	{
		ID:          "gitlab",
		Name:        "GitLab",
		Group:       "dev",
		Description: "Projects, issues, and merge requests on GitLab",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev gitlab projects", "pocket dev gitlab issues", "pocket dev gitlab mrs"},
		SetupCmd:    "pocket setup show gitlab",
	},
	{
		ID:          "linear",
		Name:        "Linear",
		Group:       "dev",
		Description: "Issues and project management with Linear",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev linear issues", "pocket dev linear teams", "pocket dev linear create [desc]"},
		SetupCmd:    "pocket setup show linear",
	},

	// Social - Auth Required
	{
		ID:          "twitter",
		Name:        "Twitter/X",
		Group:       "social",
		Description: "Post tweets, read timeline, search, and get user info",
		AuthNeeded:  true,
		Commands:    []string{"pocket social twitter timeline", "pocket social twitter post [msg]", "pocket social twitter search [query]", "pocket social twitter user [name]"},
		SetupCmd:    "pocket setup show twitter",
	},
	{
		ID:          "reddit",
		Name:        "Reddit",
		Group:       "social",
		Description: "Browse feeds, subreddits, search, and post",
		AuthNeeded:  true,
		Commands:    []string{"pocket social reddit feed", "pocket social reddit subreddit [name]", "pocket social reddit search [query]", "pocket social reddit post [content]"},
		SetupCmd:    "pocket setup show reddit",
	},
	{
		ID:          "mastodon",
		Name:        "Mastodon",
		Group:       "social",
		Description: "Fediverse: timelines, posting, and search",
		AuthNeeded:  true,
		Commands:    []string{"pocket social mastodon timeline", "pocket social mastodon post [content]", "pocket social mastodon search [query]"},
		SetupCmd:    "pocket setup show mastodon",
	},

	// Communication - Auth Required
	{
		ID:          "gmail",
		Name:        "Gmail",
		Group:       "comms",
		Description: "Read, search, and send emails via Gmail",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms email list", "pocket comms email read [id]", "pocket comms email send [body]", "pocket comms email search [query]"},
		SetupCmd:    "pocket setup show gmail",
	},
	{
		ID:          "slack",
		Name:        "Slack",
		Group:       "comms",
		Description: "Channels, messages, and sending in Slack workspaces",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms slack channels", "pocket comms slack messages [channel]", "pocket comms slack send [msg]"},
		SetupCmd:    "pocket setup show slack",
	},
	{
		ID:          "discord",
		Name:        "Discord",
		Group:       "comms",
		Description: "Servers, channels, and messages in Discord",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms discord guilds", "pocket comms discord channels [guild]", "pocket comms discord messages [channel]", "pocket comms discord send [msg]"},
		SetupCmd:    "pocket setup show discord",
	},
	{
		ID:          "telegram",
		Name:        "Telegram",
		Group:       "comms",
		Description: "Chats and messages via Telegram bot",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms telegram chats", "pocket comms telegram messages [chat]", "pocket comms telegram send [msg]"},
		SetupCmd:    "pocket setup show telegram",
	},

	// Productivity - Auth Required
	{
		ID:          "calendar",
		Name:        "Google Calendar",
		Group:       "productivity",
		Description: "View and create calendar events",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity calendar events", "pocket productivity calendar today", "pocket productivity calendar create"},
		SetupCmd:    "pocket setup show google",
	},
	{
		ID:          "notion",
		Name:        "Notion",
		Group:       "productivity",
		Description: "Search pages and query databases in Notion",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity notion search [query]", "pocket productivity notion page [id]", "pocket productivity notion database [id]"},
		SetupCmd:    "pocket setup show notion",
	},
	{
		ID:          "todoist",
		Name:        "Todoist",
		Group:       "productivity",
		Description: "Tasks and projects in Todoist",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity todoist tasks", "pocket productivity todoist projects", "pocket productivity todoist add [task]", "pocket productivity todoist complete [id]"},
		SetupCmd:    "pocket setup show todoist",
	},

	// AI - Auth Required
	{
		ID:          "openai",
		Name:        "OpenAI",
		Group:       "ai",
		Description: "Chat completions with GPT models",
		AuthNeeded:  true,
		Commands:    []string{"pocket ai openai chat [prompt]", "pocket ai openai models"},
		SetupCmd:    "pocket setup show openai",
	},
	{
		ID:          "anthropic",
		Name:        "Anthropic",
		Group:       "ai",
		Description: "Chat with Claude models",
		AuthNeeded:  true,
		Commands:    []string{"pocket ai anthropic chat [prompt]"},
		SetupCmd:    "pocket setup show anthropic",
	},
}

func NewIntegrationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "integrations",
		Aliases: []string{"int", "services"},
		Short:   "List all available integrations",
	}

	cmd.AddCommand(newIntListCmd())
	cmd.AddCommand(newIntReadyCmd())
	cmd.AddCommand(newIntGroupCmd())

	return cmd
}

func newIntListCmd() *cobra.Command {
	var noAuth bool
	var group string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all integrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			result := make([]Integration, 0)

			for _, integ := range allIntegrations {
				// Filter by auth requirement
				if noAuth && integ.AuthNeeded {
					continue
				}

				// Filter by group
				if group != "" && integ.Group != group {
					continue
				}

				// Set status
				integ.Status = getIntegrationStatus(cfg, integ)
				result = append(result, integ)
			}

			return output.Print(result)
		},
	}

	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Only show integrations that don't need authentication")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Filter by group: news, knowledge, utility, dev, social, comms, productivity, ai")

	return cmd
}

func newIntReadyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ready",
		Short: "List integrations ready to use (configured or no auth needed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			result := make([]Integration, 0)

			for _, integ := range allIntegrations {
				status := getIntegrationStatus(cfg, integ)
				if status == "ready" || status == "no_auth" {
					integ.Status = status
					result = append(result, integ)
				}
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newIntGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "List integration groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			groups := map[string]struct {
				Name  string `json:"name"`
				Desc  string `json:"desc"`
				Count int    `json:"count"`
			}{
				"news":         {Name: "News", Desc: "News feeds and articles", Count: 0},
				"knowledge":    {Name: "Knowledge", Desc: "Research and reference", Count: 0},
				"utility":      {Name: "Utility", Desc: "Weather, tools", Count: 0},
				"dev":          {Name: "Dev", Desc: "Developer tools and package registries", Count: 0},
				"social":       {Name: "Social", Desc: "Social media platforms", Count: 0},
				"comms":        {Name: "Comms", Desc: "Email and messaging", Count: 0},
				"productivity": {Name: "Productivity", Desc: "Calendar, tasks, notes", Count: 0},
				"ai":           {Name: "AI", Desc: "AI/LLM providers", Count: 0},
			}

			for _, integ := range allIntegrations {
				if g, ok := groups[integ.Group]; ok {
					g.Count++
					groups[integ.Group] = g
				}
			}

			type GroupInfo struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Desc  string `json:"desc"`
				Count int    `json:"count"`
			}

			result := []GroupInfo{
				{ID: "news", Name: groups["news"].Name, Desc: groups["news"].Desc, Count: groups["news"].Count},
				{ID: "knowledge", Name: groups["knowledge"].Name, Desc: groups["knowledge"].Desc, Count: groups["knowledge"].Count},
				{ID: "utility", Name: groups["utility"].Name, Desc: groups["utility"].Desc, Count: groups["utility"].Count},
				{ID: "dev", Name: groups["dev"].Name, Desc: groups["dev"].Desc, Count: groups["dev"].Count},
				{ID: "social", Name: groups["social"].Name, Desc: groups["social"].Desc, Count: groups["social"].Count},
				{ID: "comms", Name: groups["comms"].Name, Desc: groups["comms"].Desc, Count: groups["comms"].Count},
				{ID: "productivity", Name: groups["productivity"].Name, Desc: groups["productivity"].Desc, Count: groups["productivity"].Count},
				{ID: "ai", Name: groups["ai"].Name, Desc: groups["ai"].Desc, Count: groups["ai"].Count},
			}

			return output.Print(result)
		},
	}

	return cmd
}

func getIntegrationStatus(cfg *config.Config, integ Integration) string {
	if !integ.AuthNeeded {
		return "no_auth"
	}

	// Check if required keys are set
	switch integ.ID {
	case "github":
		if v, _ := config.Get("github_token"); v != "" {
			return "ready"
		}
	case "gitlab":
		if v, _ := config.Get("gitlab_token"); v != "" {
			return "ready"
		}
	case "linear":
		if v, _ := config.Get("linear_token"); v != "" {
			return "ready"
		}
	case "twitter":
		if v, _ := config.Get("twitter_api_key"); v != "" {
			return "ready"
		}
	case "reddit":
		if v, _ := config.Get("reddit_client_id"); v != "" {
			return "ready"
		}
	case "mastodon":
		if v, _ := config.Get("mastodon_token"); v != "" {
			return "ready"
		}
	case "gmail":
		if v, _ := config.Get("gmail_cred_path"); v != "" {
			return "ready"
		}
	case "slack":
		if v, _ := config.Get("slack_token"); v != "" {
			return "ready"
		}
	case "discord":
		if v, _ := config.Get("discord_token"); v != "" {
			return "ready"
		}
	case "telegram":
		if v, _ := config.Get("telegram_token"); v != "" {
			return "ready"
		}
	case "calendar":
		if v, _ := config.Get("google_cred_path"); v != "" {
			return "ready"
		}
	case "notion":
		if v, _ := config.Get("notion_token"); v != "" {
			return "ready"
		}
	case "todoist":
		if v, _ := config.Get("todoist_token"); v != "" {
			return "ready"
		}
	case "openai":
		if v, _ := config.Get("openai_key"); v != "" {
			return "ready"
		}
	case "anthropic":
		if v, _ := config.Get("anthropic_key"); v != "" {
			return "ready"
		}
	case "newsapi":
		if v, _ := config.Get("newsapi_key"); v != "" {
			return "ready"
		}
	}

	return "needs_setup"
}
