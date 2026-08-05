# Contributing

Thank you for contributing to Gitee CLI.

## Development

Requirements:

- Go 1.25.12 or newer
- Make
- Node.js 22 and jq when changing npm packaging
- zip when building Windows release archives

### Architecture

The project keeps command behavior, shared services, and platform integrations
separate:

```text
gitee-cli/
├── cmd/gitee/          # CLI entry point
├── internal/           # Private configuration, i18n, and version packages
└── pkg/
    ├── cmd/            # Command implementations
    ├── cmdutil/        # Shared command factories, errors, and prompts
    ├── gitee/          # Typed Gitee V5 API client and pagination
    ├── git/            # Git repository and remote utilities
    └── tui/            # Terminal tables, pagers, and viewers
```

- Errors are matched by type with `errors.As` rather than message text.
- HTTP behavior is centralized around retries, cancellation, and rate-limit tracking.
- TUI components use the Charmbracelet ecosystem consistently.
- Core command messages support English and Simplified Chinese.

### Build

```bash
make build          # build ./bin/gitee
make install        # install to $HOME/bin/gitee
make test           # run go test ./...
make lint           # run golangci-lint
```

Without Make:

```bash
go build -o gitee ./cmd/gitee
```

Fork the repository, create a focused branch, and keep unrelated changes out
of the same pull request. Commit messages use conventional prefixes such as
`feat:`, `fix:`, `docs:`, `test:`, and `chore:`.

Before opening a pull request, run:

```bash
go test ./...
go test -race ./...
go vet ./...
```

New commands and behavior changes must include tests for non-interactive use.
Machine-readable output must remain valid JSON on stdout, with diagnostics on
stderr. Destructive commands must require confirmation or an explicit
`--yes` flag outside a terminal.

### Release

The preflight, tagging, Gitee Release, npm publishing, and verification workflow
is documented in [docs/releasing.md](docs/releasing.md).

Report security issues according to [SECURITY.md](SECURITY.md), not in public
issues. Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
