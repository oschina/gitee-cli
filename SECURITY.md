# Security Policy

## Supported Versions

Before v1.0, only the latest release receives security fixes. After v1.0, the
latest minor release will be supported unless a release announcement says
otherwise.

## Reporting a Vulnerability

Do not disclose suspected vulnerabilities in a public issue. Contact the
repository maintainers privately through their Gitee profiles and include:

- the affected version and platform;
- reproduction steps or a minimal proof of concept;
- the expected impact;
- any suggested mitigation.

Maintainers will acknowledge a report within five business days, provide a
status update after initial triage, and coordinate disclosure after a fix is
available. Never include real access tokens or private repository data in a
report.

Security-sensitive dependencies are reviewed with `govulncheck` before each
release.
