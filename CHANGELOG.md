# Changelog

All notable changes to this project will be documented in this file. The
format follows Keep a Changelog, and releases use Semantic Versioning.

## [Unreleased]

### Added

- Add `gitee pr edit` and `gitee release edit` with non-interactive field
  updates and JSON output.
- Add the `gitee-release` Agent Skill for release creation, inspection,
  editing, deletion, and raw API fallback.

### Changed

- Extend `gitee issue edit` with labels and milestones, including explicit
  clearing of descriptions, assignees, and labels.
- Route unsupported Agent Skill operations through `gitee api --search`, with
  a second confirmation before API updates and deletions.
- Remove obsolete Gist references from the English and Chinese READMEs.

### Fixed

- Resolve and send the repository default branch when `gitee release create`
  is used without `--target`.
- Preserve omitted Issue fields instead of serializing unintended defaults.

## [0.2.0] - 2026-08-09

### Added

- Offline `gitee skills list`, `install`, and `uninstall` commands with five
  embedded Agent Skills: `gitee-pr`, `gitee-issue`, `gitee-repo`,
  `gitee-search`, and `gitee-api`.
- JSON output and field selection for `gitee pr diff`.
- Custom Agent Skills installation directories through `--dir` or the
  `AGENTS_SKILLS_DIR` environment variable.

### Changed

- Treat npm as the primary installation channel in the user documentation.
- Mirror managed Agent Skill directories during installation, clean up names
  deprecated by earlier releases, and preserve skills from other sources.
- Validate embedded Agent Skills and compatibility scripts during release
  preflight.
- Relax the source build requirement from Go 1.25.12 to Go 1.25 or newer.

### Removed

- Remove the `gitee gist` command and the associated Gist API client.

### Fixed

- Keep `gitee skills` plain-text output correctly formatted in non-TUI
  environments.

## [0.1.0] - 2026-08-06

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
