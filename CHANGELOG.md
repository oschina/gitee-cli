# Changelog

All notable changes to this project will be documented in this file. The
format follows Keep a Changelog, and releases use Semantic Versioning.

## [Unreleased]

### Added

- Core Gitee workflows for pull requests, issues, repositories, releases,
  gists, SSH keys, users, and raw API requests.
- Issue list sorting by creation or update time in ascending or descending order.
- Interactive terminal tables, pagers, themes, and optional image rendering.
- JSON output and non-interactive flags for scripting and CI environments.
- Multi-host authentication for gitee.com and private Gitee deployments.
- AI-assisted pull request descriptions, issue drafts, code reviews, and chat.
- Shell completion and persistent command aliases.
- English and Simplified Chinese interface support.
- Cross-platform binary and npm packaging for Linux, macOS, and Windows.
- Release checksums, release-note generation, and resumable publishing tools.
- Contribution, security, conduct, and release documentation.
- Five embedded Agent Skills with offline `gitee skills` management,
  migration-aware installation, and validation integrated into release preflight.

### Fixed

- Align plain-text list columns by Unicode terminal display width.
- Handle Chinese and other wide Unicode characters correctly when editing
  input in `gitee ai --chat`, and keep submitted user messages visible.

### Security

- Store credentials in files restricted to the current user and redact them
  from configuration output.
- Keep generated HTTPS clone credentials out of clone URLs and command-line
  arguments.
- Do not forward authentication headers across origins or redirects.
- Require confirmation or an explicit `--yes` flag for destructive commands.
- Restrict remote image downloads by protocol, address range, content type,
  response size, redirect count, and timeout.
