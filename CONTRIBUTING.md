# Contributing

Thank you for contributing to Gitee CLI.

## Development

Requirements:

- Go 1.25.12 or newer
- Make
- Node.js 22 and jq when changing npm packaging

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

Report security issues according to [SECURITY.md](SECURITY.md), not in public
issues. Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
