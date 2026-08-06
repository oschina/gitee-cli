# Gitee CLI Practice Guide

[Back to README](../README.md) · [简体中文](usage_zh.md)

This guide contains the detailed workflows and reference material kept out of
the main README. Use `gitee <command> --help` as the source of truth for every
command and flag.

- [Agent and automation workflows](#agent-and-automation-workflows)
- [TUI mode](#tui-mode)
- [Pull request branches](#pull-request-branches)
- [AI workflows](#ai-workflows)
- [Raw API requests](#raw-api-requests)
- [Configuration](#configuration)
- [Reliability and debugging](#reliability-and-debugging)

## Agent and Automation Workflows

Gitee CLI is designed to compose cleanly with coding agents, shell scripts,
and CI jobs:

- Use `--json` on supported list and view commands for structured output.
- Select repositories and hosts explicitly with `--repo` and `--hostname`.
- Supply mutation inputs through flags or stdin to avoid interactive prompts.
- Use `gitee api` when an agent needs an endpoint that has no dedicated command.
- Keep diagnostics on stderr so JSON on stdout remains machine-readable.

```bash
# Inspect open PRs in a known repository.
gitee pr list --repo owner/repo --json

# Extract a PR number for the next tool call.
PR_NUMBER=$(gitee pr list -R owner/repo -s open --json | jq -r '.[0].number')
gitee pr view "$PR_NUMBER" -R owner/repo --json

# Target a private deployment explicitly.
gitee api /user --hostname git.company.com
```

### Common Flags

| Flag | Scope | Description |
|---|---|---|
| `--hostname <host>` | all commands | Select a Gitee host; defaults to `gitee.com`. |
| `-R, --repo <owner/repo>` | PR and issue commands | Override repository inference from the git remote. |
| `-j, --json` | supported list and view commands | Emit structured JSON. |
| `--no-tui` | all commands | Disable interactive TUI mode for one command. |
| `-q, --quiet` | all commands | Suppress output except errors. |
| `-V, --verbose` | all commands | Show request, retry, and rate-limit diagnostics. |

### Command Recipes

```bash
# Filter pull requests.
gitee pr list --base main --labels bug,performance \
  --author alice --assignee bob --tester carol --milestone-number 7

# Show the least recently updated issues first.
gitee issue list --sort updated --direction asc

# Search across Gitee repositories and issues.
gitee search repos cli --language Go --sort stars_count
gitee search issues timeout --repo owner/repo --state open

# Fetch a PR without switching branches.
gitee pr fetch 42 --branch review-42

# Create a PR with reviewers and testers.
gitee pr create --title "Fix login" --assignees alice,bob --testers carol

# Create or assign an issue.
gitee issue create --ai
gitee issue assign IJEE10 alice

# Add an SSH key from a file or stdin.
gitee ssh-key add --title "My laptop" --file ~/.ssh/id_rsa.pub
cat ~/.ssh/id_rsa.pub | gitee ssh-key add --title "CI server" --file -

# Suppress normal output in scripts.
gitee issue close ICX4FO --quiet
```

### Command Aliases

Aliases persist in `~/.config/gitee/config.yml`:

```bash
gitee alias set prs "pr list -s open"
gitee alias set myissues "issue list -A me -s open"
gitee prs
gitee myissues
gitee alias list
gitee alias delete prs
```

Prefix an expansion with `!` to compose a shell workflow. Positional variables
such as `$1` and extra arguments are supported.

```bash
gitee alias set deploy "!PR=\$(gitee pr create --base \$1 --json | jq -r .number) && gitee pr comment \$PR -b ci_deploy"
gitee deploy main
```

Shell aliases run through `sh -c` on Unix/macOS and `cmd /c` on Windows.

## TUI Mode

Enable interactive tables globally:

```bash
gitee config set tui true
```

Use `--no-tui` for one command, or disable the mode globally with
`gitee config set tui false`.

### Key Bindings

| Key | Action | Available in |
|---|---|---|
| `enter` | Open the selected item in a browser | all lists |
| `v` | Preview the body or description | PR, issue, repository, release, and user lists |
| `c` | Copy the item URL | all lists |
| `d` | Show the diff | PR list |
| `f` | Fetch the PR to `pr_<n>` | PR list |
| `e` | Edit an issue inline | issue list |
| `i` | Open the image viewer | supported pager content |
| `q` / `ctrl+c` | Quit | all lists |
| `↑` / `↓` | Navigate rows | all lists |

### Image Rendering

When an issue or PR contains images, press `i` in the pager to open the image
viewer.

| Protocol | Supported terminals |
|---|---|
| Kitty Graphics | Kitty, Ghostty |
| iTerm2 Inline Images | iTerm2 |
| chafa | any terminal with `chafa` installed |
| OSC 8 hyperlink | fallback clickable link |

For tmux 3.3 or later, add the following setting to `~/.tmux.conf`, then reload
the file or restart tmux:

```tmux
set -g allow-passthrough on
```

Supported outer terminals in tmux currently include iTerm2, Kitty, and Ghostty.

## Pull Request Branches

### `pr checkout` vs `pr fetch`

| Behavior | `pr fetch` | `pr checkout` |
|---|---|---|
| Fetches the PR into a local branch | yes | yes |
| Switches to the branch | no | yes |
| Branch already exists | errors unless `--force` is used | switches directly; `--force` re-fetches |

```bash
gitee pr checkout 123                  # fetch and switch
gitee pr checkout 123 --branch review  # use a custom branch
gitee pr fetch 123                     # fetch only
gitee pr fetch 123 --force             # overwrite an existing branch
```

## AI Workflows

Gitee CLI works with OpenAI and compatible LLM APIs.

### Setup

```bash
gitee config set ai.base_url https://api.openai.com/v1
gitee config set ai.model gpt-4o-mini
gitee config set ai.token              # hidden prompt
gitee config set ai.language Chinese   # optional; defaults to English

# Non-interactive configuration.
printf '%s' "$AI_TOKEN" | gitee config set ai.token --stdin

# Environment-only credentials for any provider.
export GITEE_AI_TOKEN=sk-xxx

# Also accepted for api.openai.com.
export OPENAI_API_KEY=sk-xxx
```

Local models and other compatible providers can use the same configuration:

```bash
# Ollama
gitee config set ai.base_url http://localhost:11434/v1
gitee config set ai.model qwen2.5:7b
printf '%s' 'ollama' | gitee config set ai.token --stdin

# DeepSeek
gitee config set ai.base_url https://api.deepseek.com/v1
gitee config set ai.model deepseek-chat
gitee config set ai.token
```

### PR and Issue Assistance

`pr create --ai` reads the local diff and commit log, then generates an editable
title and body draft. `pr review --ai` fetches the PR diff and returns a
structured review. `issue create --ai` expands a short description into a
formatted issue.

```bash
gitee pr create --ai
gitee pr review 42 --ai
gitee issue create --ai
```

### Direct Chat

```bash
gitee ai "what is a rebase?"                         # streamed response
gitee ai -n "summarise this diff: $(git diff HEAD~1)" # non-streaming
gitee ai --chat                                      # multi-turn chat
```

Type `exit` to leave a multi-turn chat session.

### AI Commit Messages

The bundled `prepare-commit-msg` workflow is opt-in and only calls `gitee ai`
when `USE_GITEE_AI` is set.

```bash
git config core.hooksPath /path/to/custom_hooks
chmod +x /path/to/custom_hooks/prepare-commit-msg

git commit -m "fix: handle nil pointer"  # normal commit; AI is skipped
USE_GITEE_AI=1 git commit                # AI draft opens in the editor

git config alias.aic '!USE_GITEE_AI=1 git commit'
git aic
```

## Raw API Requests

`gitee api` sends authenticated requests to any Gitee V5 endpoint.

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
| `-H, --header` | — | Add a repeatable `Key: Value` request header. |
| `-f, --field` | — | Add a repeatable `key=value` JSON field. |
| `--body` | — | Send a raw JSON body. |
| `-p, --pretty` | `false` | Pretty-print a JSON response. |

## Configuration

Configuration files live in `~/.config/gitee/`.

| File | Purpose |
|---|---|
| `config.yml` | General settings managed through `gitee config` |
| `credentials.yml` | Default-host PAT and AI token; mode `0600` |
| `hosts.yml` | Tokens for additional hosts; mode `0600` |

Sensitive values are displayed as `<redacted>`. Existing tokens in `config.yml`
move to `credentials.yml` on the next configuration write. Reading configuration
does not create files or modify permissions.

### Configuration Keys

| Key | Default | Description |
|---|---|---|
| `tui` | `false` | Enable interactive TUI mode. |
| `colorize` | `false` | Enable syntax colorization. |
| `update_check` | `true` | Check for releases at most once every 24 hours; disabled in CI. |
| `locale` | auto | `en` or `zh_CN`; inferred from `$LANG` when unset. |
| `theme` | `default` | `auto`, `dark`, `light`, `dracula`, `tokyo-night`, or `pink`. |
| `host` | `gitee.com` | Default Gitee host. |
| `api_prefix` | `https://gitee.com/api/v5` | API base URL. |
| `pager` | — | Pager for long output. |
| `editor` | — | Interactive editor; falls back through Git and shell editor variables. |
| `default_repo` | — | Default `owner/repo` outside a git repository. |
| `ai.base_url` | — | OpenAI-compatible API base URL. |
| `ai.model` | `gpt-4o-mini` | Model name. |
| `ai.token` | — | Provider token; supports `GITEE_AI_TOKEN`. |
| `ai.language` | `English` | Language for generated content. |

```bash
gitee config set tui true
gitee config set locale zh_CN
gitee config set theme dracula
gitee config set default_repo myorg/myrepo
gitee config set editor nano
gitee config set update_check false
gitee config list
```

Set `GITEE_NO_UPDATE_NOTIFIER=1` to disable update checks without changing the
configuration.

### CLI Updates

Background update checks only print a notification. They never install a
release or show an interactive prompt. Disable them permanently with:

```bash
gitee config set update_check false
```

An explicit update remains available even when background checks are disabled:

```bash
gitee update --check       # check without installing
gitee update               # show the detected installation method and confirm
gitee update --yes         # non-interactive update
```

Global npm installations are updated through npm. Release, tagged-source, and
standalone binaries are downloaded from the Gitee Release, verified with
`checksums.txt`, and atomically replaced. Local npm dependencies must be
updated by the project that owns them. The updater never invokes `sudo`.

## Reliability and Debugging

Transient HTTP 429, 5xx, timeout, and network failures are retried with
exponential backoff. Long-running operations can be cancelled with `Ctrl+C`.

Use `--verbose` or `-V` to inspect HTTP requests, retries, and rate limits:

```bash
gitee pr list --verbose
# [DEBUG] GET /api/v5/repos/owner/repo/pulls
# [DEBUG] Rate limit: 4998/5000 (resets in 3598s)
# [DEBUG] Response 200 OK (0.234s)
```

Verbose diagnostics go to stderr and do not interfere with JSON pipelines:

```bash
gitee pr list --verbose --json | jq '.[] | .number'
```

The client tracks `X-RateLimit-*` headers automatically.
