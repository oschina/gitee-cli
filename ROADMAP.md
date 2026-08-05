# Gitee CLI Roadmap

This roadmap prioritizes predictable automation contracts before expanding API
breadth. Target versions are directional and may be adjusted based on user
feedback and Gitee API availability.

## v0.2: Automation Foundation

- Add `--paginate` and consistent pagination metadata to every list command.
- Add JSON output to every create, update, and delete command.
- Define stable exit codes for usage, authentication, authorization, not found,
  conflict, rate limit, cancellation, and server failures.
- Add optional machine-readable errors without mixing diagnostics into stdout.
- Let `gitee api` read request bodies from stdin and support typed field values.
- Make update checks cached, configurable, and disabled in CI by default.
- Make all AI-assisted creation paths fully non-interactive when input flags or
  stdin are supplied.

Acceptance criteria:

- Every command has a non-interactive integration test.
- JSON commands emit valid JSON on stdout and diagnostics only on stderr.
- No command prompts when stdin is not a terminal.
- Pagination can retrieve more than one API page without local scripting.

## v0.3: Repository Primitives

- Manage branches, tags, commits, and repository contents.
- Manage collaborators, permissions, deploy keys, and protected branches.
- Add repository and code search.
- Upload, list, and delete release assets.

Acceptance criteria:

- Read and mutation commands support repository inference and `--repo`.
- Mutations support JSON output and explicit confirmation where destructive.
- File uploads support stdin and do not require temporary plaintext files.

## v0.4: Collaboration

- Manage organizations and organization memberships.
- Manage labels and milestones.
- Edit and delete issue and pull request comments.
- Add notifications, stars, watches, and follows.
- Add global issue and pull request search.

Acceptance criteria:

- Organization and repository scopes are explicit in help and JSON output.
- Bulk and destructive operations provide dry-run or confirmation controls.
- Search results use the same field-selection contract as list commands.

## v0.5: Delivery and Project Workflows

- Manage webhooks and delivery logs.
- Inspect pipelines, builds, and job logs.
- Manage projects, packages, and wiki content where supported by public APIs.
- Add release provenance, SBOM generation, and signed checksums.

Acceptance criteria:

- CI commands support polling with timeouts and cancellation.
- Log commands support raw output suitable for files and pipelines.
- Release artifacts can be verified independently from the installed CLI.

## Authentication Track

Authentication improvements can ship alongside any milestone:

- Add OAuth device flow when the server API supports it.
- Store tokens in the operating system keychain, with the current YAML file as
  an explicit fallback for headless environments.
- Add token scope inspection and actionable missing-scope errors.
- Document what repository data AI commands send to configured LLM providers.
