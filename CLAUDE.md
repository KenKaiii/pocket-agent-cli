# Pocket Agent CLI

Universal CLI tool for LLM agents to interact with external services. Provides terminal-based access to social media, email, news, productivity tools, knowledge bases, and dev utilities through a unified interface.

## Project Structure

```
cmd/pocket/           # Entry point (main.go)
internal/
  ├── cli/            # CLI framework and command definitions
  │   └── commands/   # All subcommands (social, news, dev, etc.)
  ├── common/config/  # Configuration management (~/.config/pocket/)
  ├── social/         # Twitter, Reddit, Mastodon, YouTube
  ├── communication/  # Email (IMAP/SMTP), Slack, Discord, Telegram
  ├── news/           # HackerNews, NewsAPI, RSS feeds
  ├── knowledge/      # Wikipedia, StackExchange, Dictionary
  ├── dev/            # GitHub, GitLab, Linear, npm, PyPI
  ├── productivity/   # Notion, Todoist, Calendar
  ├── utility/        # Weather, Crypto, IP lookup
  └── ai/             # OpenAI, Anthropic integrations
pkg/output/           # JSON/text/table output formatting
```

## Organization Rules

**Keep code organized and modularized:**
- Each integration → own package under `internal/<domain>/<service>/`
- CLI commands → `internal/cli/commands/`
- Shared config → `internal/common/config/`
- Output utilities → `pkg/output/`

**Modularity principles:**
- One service per package (e.g., `internal/social/youtube/`)
- Each package has a `NewCmd()` returning cobra.Command
- Use `output.Print()` and `output.PrintError()` for all responses
- Config keys via `config.Get()` / `config.Set()`

## Code Quality - Zero Tolerance

After editing ANY file, run:

```bash
go vet ./...
go build ./...
gofmt -l .
```

Fix ALL errors/warnings before continuing.

For comprehensive linting:
```bash
golangci-lint run ./...
```

## Key Patterns

- All commands return JSON by default (LLM-friendly)
- Credentials stored in `~/.config/pocket/config.json`
- Use `output.PrintError("code", "message", details)` for errors
- Setup guides via `pocket setup show <service>`
