# Changelog

All notable changes to this project will be documented in this file. The
format follows Keep a Changelog, and releases use Semantic Versioning.

## [Unreleased]

## [0.1.0-rc.1] - 2026-08-05

### Added

- Cross-platform binary and npm packaging for Linux, macOS, and Windows.
- Non-interactive JSON output and destructive-operation safeguards.
- Cached, configurable update checks that are disabled in CI.
- Open-source governance, security, and release documentation.
- Editable release-note generation based on changes since the previous version.

### Changed

- TUI mode is opt-in and defaults to disabled.
- Pull request creation uses the API-aligned `--assignees` flag.
- Authentication accepts tokens through hidden input, stdin, or environment
  variables instead of command-line arguments.
- HTTP requests identify the running build as `gitee-cli@version` for
  service-side usage statistics.
- Release publishing uses reviewed release notes and synchronizes them when
  reusing an existing Gitee Release.

### Security

- Prevented credentials from appearing in clone URLs and process arguments.
- Prevented raw API credentials from being forwarded across origins.
- Hardened remote image loading and terminal rendering.
