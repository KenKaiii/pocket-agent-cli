package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

// Cmd represents a command for LLM consumption
type Cmd struct {
	Command string `json:"cmd"`
	Desc    string `json:"desc"`
	Args    string `json:"args,omitempty"`
	Flags   string `json:"flags,omitempty"`
}

// Group represents a command group
type Group struct {
	Name     string `json:"name"`
	Commands []Cmd  `json:"commands"`
}

func NewCommandsCmd() *cobra.Command {
	var group string

	cmd := &cobra.Command{
		Use:     "commands",
		Aliases: []string{"cmds", "ls"},
		Short:   "List all commands (LLM-friendly)",
		RunE: func(cmd *cobra.Command, args []string) error {
			all := getAllCommands()

			if group != "" {
				for _, g := range all {
					if g.Name == group {
						return output.Print(g.Commands)
					}
				}
				return output.PrintError("not_found", "group not found", nil)
			}

			return output.Print(all)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Filter by group: social, comms, dev, productivity, news, knowledge, utility, system")

	return cmd
}

func getAllCommands() []Group {
	return []Group{
		{
			Name: "social",
			Commands: []Cmd{
				{Command: "pocket social twitter timeline", Desc: "Get home timeline", Flags: "-l limit"},
				{Command: "pocket social twitter post", Desc: "Post a tweet", Args: "[message]"},
				{Command: "pocket social twitter search", Desc: "Search tweets", Args: "[query]", Flags: "-l limit"},
				{Command: "pocket social twitter user", Desc: "Get user info", Args: "[username]"},
				{Command: "pocket social reddit feed", Desc: "Get home feed", Flags: "-l limit, -s sort"},
				{Command: "pocket social reddit subreddit", Desc: "Get subreddit posts", Args: "[name]", Flags: "-l limit, -s sort"},
				{Command: "pocket social reddit search", Desc: "Search Reddit", Args: "[query]", Flags: "-l limit, -r subreddit"},
				{Command: "pocket social mastodon timeline", Desc: "Get timeline", Flags: "-l limit, -t type"},
				{Command: "pocket social mastodon post", Desc: "Post a toot", Args: "[content]", Flags: "-V visibility"},
				{Command: "pocket social mastodon search", Desc: "Search Mastodon", Args: "[query]", Flags: "-l limit, -t type"},
				{Command: "pocket social youtube search", Desc: "Search videos", Args: "[query]", Flags: "-l limit, -s order, -a after, -d duration"},
				{Command: "pocket social youtube video", Desc: "Get video details", Args: "[id]"},
				{Command: "pocket social youtube channel", Desc: "Get channel info", Args: "[id-or-handle]"},
				{Command: "pocket social youtube videos", Desc: "List channel videos", Args: "[channel-id]", Flags: "-l limit"},
				{Command: "pocket social youtube comments", Desc: "Get video comments", Args: "[video-id]", Flags: "-l limit, -s order"},
				{Command: "pocket social youtube trending", Desc: "Get trending videos", Flags: "-l limit, -r region, -c category"},
			},
		},
		{
			Name: "comms",
			Commands: []Cmd{
				{Command: "pocket comms email list", Desc: "List emails", Flags: "-l limit, -m mailbox"},
				{Command: "pocket comms email read", Desc: "Read an email", Args: "[uid]", Flags: "-m mailbox"},
				{Command: "pocket comms email send", Desc: "Send an email", Args: "[body]", Flags: "--to, --subject, --cc"},
				{Command: "pocket comms email reply", Desc: "Reply to an email", Args: "[uid] [body]", Flags: "-m mailbox, -a reply-all"},
				{Command: "pocket comms email search", Desc: "Search emails", Args: "[query]", Flags: "-l limit, -m mailbox"},
				{Command: "pocket comms email mailboxes", Desc: "List mailboxes/folders"},
				{Command: "pocket comms slack channels", Desc: "List Slack channels"},
				{Command: "pocket comms slack messages", Desc: "Get channel messages", Args: "[channel]", Flags: "-l limit"},
				{Command: "pocket comms slack send", Desc: "Send Slack message", Args: "[message]", Flags: "-c channel"},
				{Command: "pocket comms discord guilds", Desc: "List Discord servers"},
				{Command: "pocket comms discord channels", Desc: "List guild channels", Args: "[guild-id]"},
				{Command: "pocket comms discord messages", Desc: "Get channel messages", Args: "[channel-id]", Flags: "-l limit"},
				{Command: "pocket comms discord send", Desc: "Send Discord message", Args: "[message]", Flags: "-c channel"},
				{Command: "pocket comms telegram chats", Desc: "List Telegram chats"},
				{Command: "pocket comms telegram messages", Desc: "Get chat messages", Args: "[chat-id]", Flags: "-l limit"},
				{Command: "pocket comms telegram send", Desc: "Send Telegram message", Args: "[message]", Flags: "-c chat"},
			},
		},
		{
			Name: "dev",
			Commands: []Cmd{
				{Command: "pocket dev github repos", Desc: "List repositories", Flags: "-l limit, -s sort, -u user"},
				{Command: "pocket dev github repo", Desc: "Get repo details", Args: "[owner/name]"},
				{Command: "pocket dev github issues", Desc: "List issues", Flags: "-r repo, -s state, -l limit, --labels"},
				{Command: "pocket dev github issue", Desc: "Get issue details", Args: "[owner/repo] [number]"},
				{Command: "pocket dev github prs", Desc: "List pull requests", Flags: "-r repo, -s state, -l limit"},
				{Command: "pocket dev github pr", Desc: "Get PR details", Args: "[owner/repo] [number]"},
				{Command: "pocket dev github notifications", Desc: "List notifications", Flags: "-l limit, -a all"},
				{Command: "pocket dev github search", Desc: "Search GitHub", Args: "[query]", Flags: "-t type, -l limit"},
				{Command: "pocket dev gitlab projects", Desc: "List projects", Flags: "-l limit"},
				{Command: "pocket dev gitlab issues", Desc: "List issues", Flags: "-p project, -s state, -l limit"},
				{Command: "pocket dev gitlab mrs", Desc: "List merge requests", Flags: "-p project, -s state, -l limit"},
				{Command: "pocket dev linear issues", Desc: "List Linear issues", Flags: "-t team, -s status, -l limit"},
				{Command: "pocket dev linear teams", Desc: "List Linear teams"},
				{Command: "pocket dev linear create", Desc: "Create Linear issue", Args: "[description]", Flags: "-t team, --title"},
				{Command: "pocket dev npm search", Desc: "Search npm packages", Args: "[query]", Flags: "-l limit"},
				{Command: "pocket dev npm info", Desc: "Get package info", Args: "[package]"},
				{Command: "pocket dev npm versions", Desc: "List package versions", Args: "[package]", Flags: "-l limit"},
				{Command: "pocket dev npm deps", Desc: "List dependencies", Args: "[package]", Flags: "-d dev"},
				{Command: "pocket dev pypi search", Desc: "Search PyPI packages", Args: "[query]"},
				{Command: "pocket dev pypi info", Desc: "Get package info", Args: "[package]"},
				{Command: "pocket dev pypi versions", Desc: "List package versions", Args: "[package]", Flags: "-l limit"},
				{Command: "pocket dev pypi deps", Desc: "List dependencies", Args: "[package]"},
			},
		},
		{
			Name: "productivity",
			Commands: []Cmd{
				{Command: "pocket productivity calendar events", Desc: "List upcoming events", Flags: "-d days, -l limit"},
				{Command: "pocket productivity calendar today", Desc: "List today's events"},
				{Command: "pocket productivity calendar create", Desc: "Create event", Flags: "--title, --start, --end, --desc"},
				{Command: "pocket productivity notion search", Desc: "Search Notion", Args: "[query]", Flags: "-l limit"},
				{Command: "pocket productivity notion page", Desc: "Get page content", Args: "[page-id]"},
				{Command: "pocket productivity notion database", Desc: "Query database", Args: "[database-id]", Flags: "-l limit"},
				{Command: "pocket productivity todoist tasks", Desc: "List tasks", Flags: "-p project, -f filter"},
				{Command: "pocket productivity todoist projects", Desc: "List projects"},
				{Command: "pocket productivity todoist add", Desc: "Add a task", Args: "[content]", Flags: "-p project, -d due, --priority"},
				{Command: "pocket productivity todoist complete", Desc: "Complete a task", Args: "[task-id]"},
				{Command: "pocket productivity logseq graphs", Desc: "List configured graphs"},
				{Command: "pocket productivity logseq pages", Desc: "List pages in graph", Flags: "-g graph, -l limit"},
				{Command: "pocket productivity logseq read", Desc: "Read page content", Args: "[page]", Flags: "-g graph"},
				{Command: "pocket productivity logseq write", Desc: "Create/update page", Args: "[page] [content]", Flags: "-g graph, -a append"},
				{Command: "pocket productivity logseq search", Desc: "Search pages by content", Args: "[query]", Flags: "-g graph, -l limit, -c case-sensitive"},
				{Command: "pocket productivity logseq journal", Desc: "Get/create journal entry", Flags: "-g graph, -d date, -c content"},
				{Command: "pocket productivity logseq recent", Desc: "List recently modified pages", Flags: "-g graph, -l limit, -d days"},
			},
		},
		{
			Name: "news",
			Commands: []Cmd{
				{Command: "pocket news hn top", Desc: "HN top stories", Flags: "-l limit"},
				{Command: "pocket news hn new", Desc: "HN new stories", Flags: "-l limit"},
				{Command: "pocket news hn best", Desc: "HN best stories", Flags: "-l limit"},
				{Command: "pocket news hn ask", Desc: "Ask HN stories", Flags: "-l limit"},
				{Command: "pocket news hn show", Desc: "Show HN stories", Flags: "-l limit"},
				{Command: "pocket news hn item", Desc: "Get item with comments", Args: "[id]", Flags: "-c comments"},
				{Command: "pocket news feeds fetch", Desc: "Fetch RSS/Atom feed", Args: "[url]", Flags: "-l limit, -s summary-len"},
				{Command: "pocket news feeds list", Desc: "List saved feeds"},
				{Command: "pocket news feeds add", Desc: "Save a feed", Args: "[url]", Flags: "-n name"},
				{Command: "pocket news feeds read", Desc: "Fetch saved feed by name", Args: "[name]", Flags: "-l limit, -s summary-len"},
				{Command: "pocket news feeds remove", Desc: "Remove saved feed", Args: "[name-or-url]"},
				{Command: "pocket news newsapi headlines", Desc: "Get top headlines", Flags: "--country, --category, -l limit"},
				{Command: "pocket news newsapi search", Desc: "Search news", Args: "[query]", Flags: "--sort, -l limit"},
				{Command: "pocket news newsapi sources", Desc: "List news sources", Flags: "--category, --country"},
			},
		},
		{
			Name: "knowledge",
			Commands: []Cmd{
				{Command: "pocket knowledge wiki search", Desc: "Search Wikipedia", Args: "[query]", Flags: "-l limit"},
				{Command: "pocket knowledge wiki summary", Desc: "Get article summary", Args: "[title]", Flags: "-s sentences"},
				{Command: "pocket knowledge wiki article", Desc: "Get full article", Args: "[title]", Flags: "-c chars"},
				{Command: "pocket knowledge so search", Desc: "Search StackOverflow", Args: "[query]", Flags: "-l limit, -t tagged, -s site"},
				{Command: "pocket knowledge so question", Desc: "Get question details", Args: "[id]", Flags: "-s site"},
				{Command: "pocket knowledge so answers", Desc: "Get answers", Args: "[question-id]", Flags: "-l limit, -s site"},
				{Command: "pocket knowledge dict define", Desc: "Get word definition", Args: "[word]", Flags: "-l limit"},
				{Command: "pocket knowledge dict synonyms", Desc: "Get synonyms", Args: "[word]"},
				{Command: "pocket knowledge dict antonyms", Desc: "Get antonyms", Args: "[word]"},
			},
		},
		{
			Name: "utility",
			Commands: []Cmd{
				{Command: "pocket utility weather now", Desc: "Current weather", Args: "[location]"},
				{Command: "pocket utility weather forecast", Desc: "Weather forecast", Args: "[location]", Flags: "-d days"},
				{Command: "pocket utility crypto price", Desc: "Get crypto prices", Args: "[coins...]"},
				{Command: "pocket utility crypto info", Desc: "Get coin details", Args: "[coin]"},
				{Command: "pocket utility crypto top", Desc: "Top coins by market cap", Flags: "-l limit"},
				{Command: "pocket utility crypto trending", Desc: "Trending coins"},
				{Command: "pocket utility crypto search", Desc: "Search for coins", Args: "[query]", Flags: "-l limit"},
				{Command: "pocket utility ip me", Desc: "Get your public IP and location"},
				{Command: "pocket utility ip lookup", Desc: "Lookup IP geolocation", Args: "[ip]"},
			},
		},
		{
			Name: "setup",
			Commands: []Cmd{
				{Command: "pocket setup list", Desc: "List services needing setup", Flags: "-a all"},
				{Command: "pocket setup show", Desc: "Show setup instructions", Args: "[service]"},
				{Command: "pocket setup set", Desc: "Set credential for service", Args: "[service] [key] [value]"},
			},
		},
		{
			Name: "config",
			Commands: []Cmd{
				{Command: "pocket config path", Desc: "Show config file path"},
				{Command: "pocket config list", Desc: "List all config (redacted)"},
				{Command: "pocket config set", Desc: "Set a config value", Args: "[key] [value]"},
				{Command: "pocket config get", Desc: "Get a config value", Args: "[key]"},
			},
		},
		{
			Name: "system",
			Commands: []Cmd{
				{Command: "pocket system clipboard get", Desc: "Get clipboard content", Flags: "-m max-length, -r raw"},
				{Command: "pocket system clipboard set", Desc: "Set clipboard content", Args: "[text]"},
				{Command: "pocket system clipboard clear", Desc: "Clear clipboard"},
				{Command: "pocket system clipboard copy", Desc: "Copy file to clipboard", Args: "[file]"},
				{Command: "pocket system clipboard history", Desc: "Clipboard history info"},
				{Command: "pocket system notes list", Desc: "List all notes", Flags: "-f folder, -l limit"},
				{Command: "pocket system notes folders", Desc: "List all folders"},
				{Command: "pocket system notes read", Desc: "Read a note by name", Args: "[name]", Flags: "-f folder"},
				{Command: "pocket system notes create", Desc: "Create a new note", Args: "[name] [body]", Flags: "-f folder"},
				{Command: "pocket system notes search", Desc: "Search notes", Args: "[query]", Flags: "-l limit"},
				{Command: "pocket system notes append", Desc: "Append text to note", Args: "[name] [text]", Flags: "-f folder"},
				{Command: "pocket system mail accounts", Desc: "List mail accounts"},
				{Command: "pocket system mail mailboxes", Desc: "List mailboxes/folders", Flags: "-a account"},
				{Command: "pocket system mail list", Desc: "List recent messages", Flags: "-m mailbox, -a account, -l limit, -u unread"},
				{Command: "pocket system mail read", Desc: "Read a message by ID", Args: "[id]"},
				{Command: "pocket system mail search", Desc: "Search messages", Args: "[query]", Flags: "-l limit, -m mailbox, -a account"},
				{Command: "pocket system mail send", Desc: "Send an email", Flags: "--to, --subject, --body, --cc, --bcc, -a account"},
				{Command: "pocket system mail unread", Desc: "List unread messages", Flags: "-l limit, -a account"},
				{Command: "pocket system mail count", Desc: "Get unread message count", Flags: "-a account"},
				{Command: "pocket system safari tabs", Desc: "List all open tabs", Flags: "-w window"},
				{Command: "pocket system safari url", Desc: "Get URL of current tab", Flags: "-w window, -t tab"},
				{Command: "pocket system safari title", Desc: "Get title of current tab"},
				{Command: "pocket system safari open", Desc: "Open URL in new tab", Args: "[url]", Flags: "-n new-window"},
				{Command: "pocket system safari close", Desc: "Close current tab", Flags: "-w window, -t tab, --window-close"},
				{Command: "pocket system safari bookmarks", Desc: "List bookmarks", Flags: "-f folder, -l limit"},
				{Command: "pocket system safari reading-list", Desc: "List Reading List items", Flags: "-l limit"},
				{Command: "pocket system safari add-reading", Desc: "Add URL to Reading List", Args: "[url]", Flags: "-t title"},
				{Command: "pocket system safari history", Desc: "Get recent history", Flags: "-l limit, -d days, -s search"},
			},
		},
	}
}
