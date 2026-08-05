<div align="center">

<img src="logo.png" alt="Gitee CLI logo" width="360">

# Gitee CLI

**Work with Gitee without leaving your terminal.**

Pull requests · Issues · Repositories · Releases · AI workflows · Gitee API

[![Go Version](https://img.shields.io/badge/Go-1.25.12+-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-MIT-2F6FEB)](LICENSE)
![Platforms](https://img.shields.io/badge/Platforms-macOS%20%7C%20Linux%20%7C%20Windows-4C566A)

**English** · [简体中文](README_zh.md)

[Quick start](#quick-start) · [Commands](#commands) · [TUI](#tui-mode) · [AI](#ai-features) · [Configuration](#configuration) · [Development](#development)

</div>

---

## Why Gitee CLI

| Core capability |
|---|
| **Daily workflows** · Manage PRs, issues, repositories, releases, gists, SSH keys, and raw API calls. |
| **Terminal UI** · Browse and act on results in interactive tables with contextual key bindings. |
| **Automation** · Use JSON output, explicit non-interactive flags, aliases, and shell completion. |
| **AI assistance** · Draft PRs and issues, review code, chat, and connect any OpenAI-compatible provider. |
| **Multiple hosts** · Work with `gitee.com` and private Gitee deployments from the same configuration. |
| **Resilient operations** · Retry transient failures, track rate limits, cancel cleanly, and inspect requests with `--verbose`. |

## Installation

> **Pre-release:** Gitee CLI has not published its first release yet. Install the current `main` branch from source.

Go 1.25.12 or later is required.

```bash
git clone https://gitee.com/oschina/gitee-cli.git
cd gitee-cli
make install
```

`make install` builds the CLI and installs it to `$HOME/bin/gitee`.

<details>
<summary><strong>Build without make</strong></summary>

```bash
git clone https://gitee.com/oschina/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
```

</details>

## Quick Start

```console
$ gitee auth login
$ gitee pr list
$ gitee pr checkout 123
$ gitee issue list -s open
$ gitee api /user
```

| Goal | Command |
|---|---|
| Authenticate | `gitee auth login` |
| Work outside a repository | `gitee pr list -R owner/repo` |
| Enable interactive tables | `gitee config set tui true` |
| Switch the interface to Chinese | `gitee config set locale zh_CN` |
| Inspect a command | `gitee <command> --help` |

---

## Authentication

Gitee CLI uses a Personal Access Token (PAT) for authentication.

1. Create a token at [Gitee Settings / Personal Access Tokens](https://gitee.com/profile/personal_access_tokens)
2. Run `gitee auth login` and paste your token
3. The token is stored in `~/.config/gitee/credentials.yml`

```bash
gitee auth status                          # show login status
gitee auth token                           # print the stored token
gitee auth logout                          # remove stored credentials
```

For non-interactive environments, read the token from standard input:

```bash
printf '%s' "$GITEE_TOKEN" | gitee auth login --with-token
```

### Multi-Host (Private Deployments)

Log in to multiple Gitee instances simultaneously:

```bash
gitee auth login --hostname git.company.com
gitee pr list --hostname git.company.com -R owner/repo
gitee auth status
gitee auth logout --hostname git.company.com
```

Tokens for non-default hosts are stored in `~/.config/gitee/hosts.yml`.

---

## Commands

| Command | Subcommands | Description |
|---|---|---|
| `auth` | `login`, `logout`, `status`, `token` | Manage authentication |
| `config` | `get`, `set`, `list` | Manage configuration values |
| `pr` | `list`, `view`, `create`, `close`, `reopen`, `merge`, `review`, `comment`, `diff`, `fetch`, `checkout` | Manage pull requests |
| `issue` | `list`, `view`, `create`, `close`, `reopen`, `edit`, `assign`, `comment` | Manage issues |
| `repo` | `list`, `view`, `clone`, `create`, `fork`, `delete` | Manage repositories |
| `release` | `list`, `view`, `create`, `delete` | Manage releases |
| `gist` | `list`, `view`, `create`, `delete` | Manage gists |
| `ssh-key` | `list`, `add`, `delete` | Manage SSH keys (supports stdin input) |
| `user` | `search` | Search users |
| `alias` | `list`, `set`, `delete` | Manage command aliases |
| `api` | `<endpoint>` | Make raw API requests |
| `version` | — | Show version information |
| `completion` | `bash`, `zsh`, `fish`, `powershell` | Generate shell completion scripts |

### Global Flags

These flags are available on every command:

| Flag | Description |
|---|---|
| `--hostname <host>` | Gitee hostname to use (default: `gitee.com`) |
| `--no-tui` | Disable TUI mode for this command |
| `-q, --quiet` | Suppress all output except errors |
| `-V, --verbose` | Enable detailed debug logging (shows HTTP requests, rate limits, retries) |

The following flags are available on `pr` and `issue` subcommands:

| Flag | Description |
|---|---|
| `-R, --repo <owner/repo>` | Override the target repository (default: inferred from git remote) |

Most list and view commands also accept:

| Flag | Description |
|---|---|
| `-j, --json` | Output results as JSON |

### Common Examples

```bash
# List open PRs as JSON
gitee pr list -s open --json

# Fetch PR #42 to a local branch and switch to it
gitee pr checkout 42

# Create a PR with AI-generated description
gitee pr create --ai

# AI code review of a PR
gitee pr review 42 --ai

# Raw API call
gitee api /user

# Set a handy alias
gitee alias set prs "pr list -s open"
gitee prs
```

<details>
<summary><strong>More command examples</strong></summary>

```bash
# Filter PRs by branch, labels, people, and milestone
gitee pr list --base main --labels bug,performance \
  --author alice --assignee bob --tester carol --milestone-number 7

# Fetch without switching branches
gitee pr fetch 42 --branch review-42

# Create a PR with reviewers and testers
gitee pr create --title "Fix login" --assignees alice,bob --testers carol

# Create or assign an issue
gitee issue create --ai
gitee issue assign IJEE10 alice

# Work with releases and SSH keys
gitee release view v1.2.0
gitee ssh-key add --title "My laptop" --file ~/.ssh/id_rsa.pub
cat ~/.ssh/id_rsa.pub | gitee ssh-key add --title "CI server" --file -

# Send an API mutation or target another host
gitee api -X PATCH /repos/owner/repo/issues/1 -f title="Updated"
gitee api /repos/owner/repo/pulls --hostname git.company.com

# Suppress output (useful in scripts)
gitee issue close ICX4FO --quiet

# Debug API calls with verbose mode
gitee pr list --verbose
# [DEBUG] GET /api/v5/repos/owner/repo/pulls?state=open&per_page=100
# [DEBUG] Rate limit: 4998/5000
# [DEBUG] Response 200 OK (0.234s)
```

</details>

---

## TUI Mode

Enable TUI mode to browse data in interactive tables:

```bash
gitee config set tui true
```

When active, list commands open an interactive interface with a help bar showing available key bindings.

### Key Bindings

| Key | Action | Available in |
|---|---|---|
| `enter` | Open selected item in browser | all lists |
| `v` | Preview body/description in pager | pr, issue, repo, release, user lists |
| `c` | Copy item URL to clipboard | all lists |
| `d` | Show diff in pager | pr list |
| `f` | Fetch PR to local branch (`pr_<n>`) | pr list |
| `e` | Edit issue inline | issue list |
| `q` / `ctrl+c` | Quit | all lists |
| `↑` / `↓` | Navigate rows | all lists |

Disable TUI for a single command:

```bash
gitee pr list --no-tui
```

Disable globally:

```bash
gitee config set tui false
```

---

## `pr checkout` vs `pr fetch`

| | `pr fetch` | `pr checkout` |
|---|---|---|
| Fetches PR into local branch | ✅ | ✅ |
| Switches to the branch | ❌ | ✅ |
| Branch already exists | error (use `--force`) | switches directly (use `--force` to re-fetch) |

```bash
gitee pr checkout 123                  # fetch + switch
gitee pr checkout 123 --branch review  # custom branch name
gitee pr fetch 123                     # fetch only
gitee pr fetch 123 --force             # overwrite existing branch
```

---

## AI Features

Gitee CLI integrates with any OpenAI-compatible LLM API.

### Setup

```bash
gitee config set ai.base_url https://api.openai.com/v1
gitee config set ai.model gpt-4o-mini
gitee config set ai.token              # hidden prompt; stored in credentials.yml
gitee config set ai.language Chinese   # optional, default: English

# In scripts
printf '%s' "$AI_TOKEN" | gitee config set ai.token --stdin

# Or use an environment variable (not written to disk) for any provider
export GITEE_AI_TOKEN=sk-xxx

# OPENAI_API_KEY is also accepted when ai.base_url uses https://api.openai.com
export OPENAI_API_KEY=sk-xxx
```

### `pr create --ai`

Reads your local `git diff` and commit log, generates a PR title and body draft, then lets you confirm, edit, regenerate, or skip:

```
✦ Collecting git context...
✦ Generating PR description with gpt-4o-mini...

────────────────────────────────────────
Title: fix(auth): resolve token expiry race condition
...
────────────────────────────────────────
[y] Use   [e] Edit in editor   [r] Regenerate   [n] Write manually
Choice [y/e/r/n]:
```

### `pr review --ai`

Fetches the PR diff and asks the LLM for a structured code review (summary, blocking issues, suggestions, verdict). Displayed in the pager when TUI is enabled:

```bash
gitee pr review 42 --ai
```

### `issue create --ai`

Type a one-sentence description; the LLM expands it into a formatted bug report or feature request:

```bash
gitee issue create --ai
# Describe the issue in one or two sentences: login page crashes on empty password
```

### `gitee ai`

Ask the AI anything in a single shot (streaming by default) or start a multi-turn chat:

```bash
# Single question — streamed output
gitee ai "what is a rebase?"

# Non-streaming (useful for scripting / piping)
gitee ai -n "summarise this diff: $(git diff HEAD~1)"

# Multi-turn chat session (type `exit` to quit)
gitee ai --chat
```

### AI-powered commit messages (git hook)

Set up a `prepare-commit-msg` hook that calls `gitee ai` to generate a commit message from the staged diff. The hook is **opt-in**: it only runs when the `USE_GITEE_AI` environment variable is set, so your normal `git commit` workflow is unaffected.

**1. Configure the hook**

```bash
# Point this repo at your custom hooks directory
git config core.hooksPath /path/to/custom_hooks

# Make the hook executable
chmod +x /path/to/custom_hooks/prepare-commit-msg
```

**2. Use it**

```bash
# Normal commit — hook is skipped, no AI involved
git commit -m "fix: handle nil pointer"

# AI-generated commit message — opens your editor pre-filled by the LLM
USE_GITEE_AI=1 git commit
```

**3. Optional: add a git alias for convenience**

```bash
git config alias.aic '!USE_GITEE_AI=1 git commit'

# Now just run:
git aic
```

---

## `gitee api`

Make authenticated requests to any Gitee V5 API endpoint:

```bash
gitee api /user
gitee api -X POST /repos/owner/repo/issues -f title="Bug" -f body="Details"
gitee api --body '{"state":"closed"}' -X PATCH /repos/owner/repo/issues/1
gitee api -H "Accept: application/json" /user
gitee api /repos/owner/repo/pulls --hostname git.company.com
```

| Flag | Default | Description |
|---|---|---|
| `-X, --method` | `GET` | HTTP method |
| `-H, --header` | — | Add request header (`Key: Value`), repeatable |
| `-f, --field` | — | Add JSON body field (`key=value`), repeatable |
| `--body` | — | Raw JSON request body |
| `-p, --pretty` | `false` | Pretty-print JSON response |

---

## Reliability & Debugging

Gitee CLI is built to handle real-world network conditions gracefully.

### Automatic Retry

Transient errors (HTTP 429 rate limits, 5xx server errors, network timeouts) are automatically retried with exponential backoff. This is transparent — you'll only see the final result.

```bash
# No special flags needed — retries happen automatically
gitee pr list    # if rate limited, CLI waits and retries
```

### Signal Handling

Long-running operations (AI calls, large diffs, paginated queries) can be cancelled with `Ctrl+C`. The CLI cleans up resources and exits gracefully.

```bash
# Start a long AI operation
gitee ai "analyze all the commits in this repo"

# Oops, wrong question — press Ctrl+C to cancel
^C
# CLI exits cleanly, no hanging process
```

### Verbose Mode

Use `--verbose` (or `-V`) to see detailed debug information including HTTP requests, rate limits, and retry attempts:

```bash
gitee pr list --verbose
# [DEBUG] GET /api/v5/repos/owner/repo/pulls
# [DEBUG] Rate limit: 4998/5000 (resets in 3598s)
# [DEBUG] Response 200 OK (0.234s)
```

Verbose output goes to stderr, so it doesn't interfere with JSON parsing or piping:

```bash
# Debug output on stderr, JSON on stdout
gitee pr list --verbose --json | jq '.[] | .number'
```

### Rate Limit Tracking

The CLI automatically tracks Gitee API rate limits via `X-RateLimit-*` headers. Check your remaining quota with verbose mode:

```bash
gitee user search alice --verbose
# [DEBUG] X-RateLimit-Limit: 5000
# [DEBUG] X-RateLimit-Remaining: 4997
# [DEBUG] X-RateLimit-Reset: 1234567890
```

---

## Configuration

Configuration files live in `~/.config/gitee/`.

| File | Purpose |
|---|---|
| `config.yml` | General settings (managed via `gitee config`) |
| `credentials.yml` | Default host PAT and AI token (`0600`) |
| `hosts.yml` | Per-host tokens for multi-host setups (`0600`) |

Sensitive values are shown as `<redacted>` by `gitee config get/list`. Existing tokens in
`config.yml` are moved to `credentials.yml` automatically on the next configuration write.
Reading configuration never creates files or changes permissions, so help and environment-only
commands work with a read-only home directory.

### Config Keys

| Key | Default | Description |
|---|---|---|
| `tui` | `false` | Enable interactive TUI mode |
| `colorize` | `false` | Enable syntax colorization |
| `update_check` | `true` | Check for new releases at most once every 24 hours (disabled in CI) |
| `locale` | auto | UI language: `en`, `zh_CN` (auto-detected from `$LANG` if not set) |
| `theme` | `default` | Color theme: `auto`, `dark`, `light`, `dracula`, `tokyo-night`, `pink` |
| `host` | `gitee.com` | Default Gitee host |
| `api_prefix` | `https://gitee.com/api/v5` | API base URL |
| `pager` | — | Pager program for long output |
| `editor` | — | Editor for interactive text input (fallback: `$GIT_EDITOR` / `$VISUAL` / `$EDITOR` / `vim`) |
| `default_repo` | — | Default `owner/repo` used outside a git repository |
| `ai.base_url` | — | OpenAI-compatible API base URL |
| `ai.model` | `gpt-4o-mini` | Model name |
| `ai.token` | — | Token for the configured provider (`GITEE_AI_TOKEN`; `OPENAI_API_KEY` is accepted only for `api.openai.com`) |
| `ai.language` | `English` | Language for AI-generated content |

```bash
gitee config set tui true
gitee config set locale zh_CN
gitee config set theme dracula
gitee config set api_prefix https://my-gitee.example.com/api/v5
gitee config set editor nano
gitee config set update_check false
gitee config list
```

Set `GITEE_NO_UPDATE_NOTIFIER=1` to disable update checks without changing configuration.

---

## Command Aliases

Aliases let you define shortcuts for frequently used commands:

```bash
gitee alias set prs "pr list -s open"
gitee alias set myissues "issue list -A me -s open"
gitee alias set merged "pr list -s merged -l 10"

gitee prs
gitee myissues

gitee alias list
gitee alias delete prs
```

Prefix the expansion with `!` to run an arbitrary shell command. This unlocks
multi-step workflows by composing gitee commands with shell features like
`$(...)`, `&&`, and pipes:

```bash
# Create a PR against a given base branch, then comment ci_deploy to trigger CI
gitee alias set deploy "!PR=\$(gitee pr create --base \$1 --json | jq -r .number) && gitee pr comment \$PR -b ci_deploy"
gitee deploy main
```

Positional variables (`$1`, `$2`, ...) and extra-argument appending work the
same as for regular aliases. Shell aliases are executed via `sh -c` on
Unix/macOS and `cmd /c` on Windows.

Aliases are stored in `~/.config/gitee/config.yml` and persist across sessions.

---

## Development

### Prerequisites

- Go 1.25.12 or later
- Make (optional)
- zip (for building Windows release archives)

### Architecture

The project follows a clean separation of concerns:

```
gitee-cli/
├── cmd/gitee/          # Entry point (main.go)
├── internal/           # Private packages (config, i18n, build version)
├── pkg/
│   ├── cmd/           # Command implementations (auth, pr, issue, etc.)
│   ├── cmdutil/       # Command utilities (Factory, errors, prompts)
│   ├── gitee/         # Gitee V5 API client (typed endpoints and pagination helpers)
│   ├── tui/           # Terminal UI components (Bubble Tea tables, pagers)
│   └── git/           # Git remote and repository utilities
```

Key design decisions:
- **Error handling:** Type-safe with `errors.As()`, no string matching
- **HTTP client:** Centralized `baseClient` with retry/backoff/rate-limit tracking
- **TUI:** Consistent Charmbracelet ecosystem (Bubble Tea + Glamour + Huh)
- **i18n:** Core command messages support English and Simplified Chinese

### Build

```bash
make build          # builds to ./bin/gitee
make install        # installs to $HOME/bin/gitee
make test           # go test ./...
make lint           # golangci-lint run
```

Or directly:

```bash
go build -o gitee ./cmd/gitee
```

### Testing

The test suite includes API client tests, table-driven command tests, and reusable
test factories. Coverage is measured with the command below rather than maintained as
a fixed percentage claim.

Run the full test suite:

```bash
make test
# or
go test ./... -cover
```

For TDD, use `entr` to watch for changes:

```bash
find . -name "*.go" | entr -c go test ./pkg/gitee/...
```

### Release

The complete preflight, tagging, Gitee Release upload, npm, and verification procedure is documented in
[docs/releasing.md](docs/releasing.md). Releases build Linux/macOS/Windows × amd64/arm64 and
publish checksums alongside the archives.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development and pull request requirements. Security
reports follow [SECURITY.md](SECURITY.md), and all participation follows
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

---

## License

[MIT](LICENSE)
