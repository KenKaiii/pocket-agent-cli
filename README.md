# 🛠️ Pocket CLI

<p align="center">
  <img src="https://raw.githubusercontent.com/KenKaiii/pocket-agent/main/assets/icon_rounded_1024.png" alt="Pocket CLI" width="200">
</p>

<p align="center">
  <strong>Give your AI assistant hands to interact with the internet.</strong>
</p>

<p align="center">
  <a href="https://github.com/KenKaiii/pocket-agent-cli/releases/latest"><img src="https://img.shields.io/github/v/release/KenKaiii/pocket-agent-cli?include_prereleases&style=for-the-badge" alt="GitHub release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge" alt="MIT License"></a>
  <a href="https://youtube.com/@kenkaidoesai"><img src="https://img.shields.io/badge/YouTube-FF0000?style=for-the-badge&logo=youtube&logoColor=white" alt="YouTube"></a>
  <a href="https://skool.com/kenkai"><img src="https://img.shields.io/badge/Skool-Community-7C3AED?style=for-the-badge" alt="Skool"></a>
</p>

**Pocket CLI** gives your AI assistant the power to actually *do things* on the internet — check emails, browse social media, get news, look up information, and more.

Think of it as hands for your AI. Instead of just chatting, your AI can now reach out and interact with real services like Twitter, YouTube, Hacker News, Wikipedia, and dozens more.

No coding required. Just install it, and your AI assistant instantly gains superpowers to help you with real tasks across the web.

---

## 🚀 Install

One command. That's it.

```bash
curl -fsSL https://raw.githubusercontent.com/KenKaiii/pocket-agent-cli/main/scripts/install.sh | bash
```

Works on **macOS** (Intel & Apple Silicon), **Linux**, and **Windows**.

The installer automatically:
- Downloads the right version for your system
- Installs it globally
- Configures your shell
- Restarts your terminal

To update later, just run the same command again.

---

## 🧠 Why this exists

AI assistants are smart but powerless. They can answer questions, but they can't actually *do* anything.

Pocket CLI changes that. It's a universal interface that lets any AI agent interact with the real world:
- Check your emails and send replies
- Send SMS via Twilio, push notifications via ntfy
- Message on Slack, Discord, Telegram
- Search YouTube, get video stats
- Browse Hacker News, Reddit, Twitter
- Look up weather, crypto prices, currency rates
- Query Wikipedia, StackOverflow, dictionaries
- Manage Todoist tasks, Notion pages, Obsidian vaults
- Control macOS apps: Calendar, Reminders, Notes, Contacts, Finder, Safari
- **52 integrations** across 9 categories

All with simple commands that return clean JSON — perfect for AI to understand and act on.

---

## ✨ What you can do

### No setup required (works immediately)
```bash
pocket news hn top -l 5              # Top 5 Hacker News stories
pocket utility weather now "Tokyo"   # Current weather in Tokyo
pocket knowledge wiki summary "AI"   # Wikipedia summary
pocket utility crypto price bitcoin  # Bitcoin price
pocket utility currency convert 100 USD EUR  # Currency conversion
pocket utility translate text "Hello" --to es # Translate to Spanish
pocket dev npm info react            # npm package info
pocket dev dockerhub search nginx    # Search Docker images
pocket comms notify ntfy mytopic "Hello!"    # Push notification (no auth)
pocket comms webhook slack [url] "Message"   # Slack webhook

# macOS only (no auth needed)
pocket system reminders today        # Today's reminders
pocket system notes list             # List Apple Notes
pocket system calendar today         # Today's calendar events
pocket system clipboard get          # Get clipboard content
pocket system finder search "query"  # Spotlight search
```

### With credentials (one-time setup)
```bash
pocket comms email list -l 10        # Your latest emails
pocket comms slack channels          # List Slack channels
pocket comms discord guilds          # List Discord servers
pocket comms telegram send 123 "Hi"  # Send Telegram message
pocket comms twilio send +1234 "SMS" # Send SMS via Twilio
pocket social youtube search "AI"    # Search YouTube
pocket social twitter timeline       # Your Twitter feed
pocket productivity todoist tasks    # Your todo list
pocket productivity trello boards    # Your Trello boards
pocket productivity obsidian notes   # List Obsidian notes
pocket productivity logseq pages     # List Logseq pages
pocket dev github repos              # Your GitHub repos
pocket dev jira issues               # Your Jira issues
```

---

## 🔧 Quick start

### See what's available
```bash
pocket commands                      # All commands (for AI agents)
pocket integrations list             # All integrations + auth status
pocket integrations list --no-auth   # Services that work without setup
```

### Set up credentials
```bash
pocket setup list                    # What needs configuration
pocket setup show email              # Step-by-step setup guide
pocket setup set email imap_server imap.gmail.com
```

### Example workflow
```bash
# Check what integrations work without auth
$ pocket integrations list --no-auth

# Get top tech news
$ pocket news hn top -l 3

# Look up a term
$ pocket knowledge dict define "API"

# Check the weather
$ pocket utility weather now "San Francisco"

# Send yourself a notification (no auth needed!)
$ pocket comms notify ntfy my-alerts "Task completed!"
```

### Communication examples
```bash
# Send SMS (requires Twilio setup)
pocket comms twilio send "+15551234567" "Hello from Pocket CLI"

# Discord bot commands
pocket comms discord guilds              # List servers
pocket comms discord channels 123456     # List channels in server
pocket comms discord send 789 "Hello!"   # Send message to channel

# Slack integration
pocket comms slack channels              # List channels
pocket comms slack send general "Hi!"    # Post to channel
pocket comms slack search "important"    # Search messages

# Telegram bot
pocket comms telegram chats              # List chats
pocket comms telegram send 123 "Hello"   # Send message

# Push notifications (ntfy.sh - no auth!)
pocket comms notify ntfy alerts "Server is down!" --priority 5

# Webhooks (no auth)
pocket comms webhook discord [url] "Deployment complete"
```

### macOS system examples (no auth needed)
```bash
# Apple Reminders
pocket system reminders lists            # List all reminder lists
pocket system reminders today            # Today's reminders
pocket system reminders add "Buy milk"   # Add a reminder
pocket system reminders complete "Buy milk"  # Mark complete

# Apple Notes
pocket system notes folders              # List folders
pocket system notes list                 # List all notes
pocket system notes read "Shopping"      # Read a note
pocket system notes create "Ideas" "My brilliant idea"

# Apple Calendar
pocket system apple-calendar calendars   # List calendars
pocket system apple-calendar today       # Today's events
pocket system apple-calendar upcoming    # Next 7 days

# Apple Contacts
pocket system contacts search "John"     # Search contacts
pocket system contacts get "John Doe"    # Get full details

# Finder & Clipboard
pocket system finder search "project"    # Spotlight search
pocket system finder info ~/Documents    # Get folder info
pocket system clipboard get              # Get clipboard
pocket system clipboard set "Hello"      # Set clipboard

# Safari (requires Safari to be running)
pocket system safari tabs                # List open tabs
pocket system safari bookmarks           # List bookmarks
pocket system safari history --limit 10  # Recent history
```

### Obsidian & Logseq examples
```bash
# Obsidian (configure vault path first)
pocket config set obsidian_vault ~/Documents/MyVault
pocket productivity obsidian notes       # List all notes
pocket productivity obsidian daily       # Today's daily note
pocket productivity obsidian search "AI" # Search notes
pocket productivity obsidian read "Ideas"  # Read a note

# Logseq (configure graph path first)
pocket config set logseq_graph ~/Documents/MyGraph
pocket productivity logseq pages         # List pages
pocket productivity logseq journal       # Today's journal
pocket productivity logseq search "todo" # Search pages
```

---

## 📦 All 51 integrations

| Category | Services |
|----------|----------|
| **Social** (4) | Twitter, Reddit, Mastodon, YouTube |
| **Communication** (7) | Email, Slack, Discord, Telegram, Twilio SMS, Push Notifications (ntfy/Pushover), Webhooks |
| **News** (3) | Hacker News, RSS feeds, NewsAPI |
| **Knowledge** (3) | Wikipedia, StackOverflow, Dictionary |
| **Dev Tools** (9) | GitHub, GitLab, Linear, Jira, Cloudflare, Vercel, npm, PyPI, Docker Hub |
| **Productivity** (6) | Todoist, Notion, Google Calendar, Trello, Obsidian, Logseq |
| **Utility** (10) | Weather, Crypto, Currency, IP lookup, DNS/WHOIS/SSL, Wayback Machine, Holidays, Translation, URL Shortener, Stocks |
| **System** (9) | Apple Calendar, Apple Reminders, Apple Notes, Apple Contacts, Apple Mail, Safari, Finder, Clipboard, iMessage *(macOS only)* |

### 27 integrations work without any setup:
Hacker News, RSS, Wikipedia, StackOverflow, Dictionary, Weather, Crypto, Currency, IP lookup, Domain tools, Wayback Machine, Holidays, Translation, URL Shortener, npm, PyPI, Docker Hub, ntfy notifications, Webhooks, plus all 8 macOS System integrations

---

## 🤖 Built for AI agents

Every command outputs clean JSON:

```json
{
  "success": true,
  "data": {
    "title": "Show HN: I built a CLI for AI agents",
    "score": 142,
    "url": "https://..."
  }
}
```

Errors are structured too:

```json
{
  "success": false,
  "error": {
    "code": "setup_required",
    "message": "Email not configured",
    "setup_cmd": "pocket setup show email"
  }
}
```

Your AI knows exactly what went wrong and how to fix it.

---

## 🔒 Privacy

- Credentials stored locally in `~/.config/pocket/config.json`
- No telemetry, no analytics
- API calls go directly to the services you configure
- Open source — inspect every line

---

## 🛠️ For developers

```bash
git clone https://github.com/KenKaiii/pocket-agent-cli.git
cd pocket-agent-cli
make install
```

Build releases for all platforms:
```bash
make release
```

Stack: Go + Cobra CLI + zero external dependencies at runtime

---

## 👥 Community

- [YouTube @kenkaidoesai](https://youtube.com/@kenkaidoesai) — tutorials and demos
- [Skool community](https://skool.com/kenkai) — come hang out

---

## 📄 License

MIT

---

<p align="center">
  <strong>Give your AI the power to actually do things.</strong>
</p>

<p align="center">
  <a href="https://github.com/KenKaiii/pocket-agent-cli/releases/latest"><img src="https://img.shields.io/badge/Install-One%20Command-blue?style=for-the-badge" alt="Install"></a>
</p>
