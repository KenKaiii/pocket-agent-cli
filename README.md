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
- Search YouTube, get video stats
- Browse Hacker News, Reddit, Twitter
- Look up weather, crypto prices, IP info
- Query Wikipedia, StackOverflow, dictionaries
- Manage Todoist tasks, Notion pages
- And 50+ more integrations

All with simple commands that return clean JSON — perfect for AI to understand and act on.

---

## ✨ What you can do

### No setup required (works immediately)
```bash
pocket news hn top -l 5              # Top 5 Hacker News stories
pocket utility weather now "Tokyo"   # Current weather in Tokyo
pocket knowledge wiki summary "AI"   # Wikipedia summary
pocket utility crypto price bitcoin  # Bitcoin price
pocket dev npm info react            # npm package info
```

### With credentials (one-time setup)
```bash
pocket comms email list -l 10        # Your latest emails
pocket social youtube search "AI"    # Search YouTube
pocket social twitter timeline       # Your Twitter feed
pocket productivity todoist tasks    # Your todo list
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
```

---

## 📦 All integrations

| Category | Services |
|----------|----------|
| **Social** | Twitter, Reddit, Mastodon, YouTube |
| **Communication** | Email (IMAP/SMTP), Slack, Discord, Telegram |
| **News** | Hacker News, RSS feeds, NewsAPI |
| **Knowledge** | Wikipedia, StackOverflow, Dictionary |
| **Dev Tools** | GitHub, GitLab, Linear, npm, PyPI |
| **Productivity** | Todoist, Notion, Calendar |
| **Utility** | Weather, Crypto prices, IP lookup |
| **AI** | OpenAI, Anthropic |

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
