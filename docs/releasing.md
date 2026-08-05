# Release Process

Releases are built from a clean, annotated tag. The Makefile produces binary
archives and checksums, creates the Gitee Release, and uploads its attachments.
npm publishing is a separate resumable step because npm does not support
transactions across multiple packages.

## Preflight

Publishing requires `curl`, `jq`, and an authenticated Gitee CLI. Run
`gitee auth login` before the first release, or provide `GITEE_TOKEN` in the
publishing environment.

1. Confirm `main` is clean and CI is green.
2. Generate an editable release-note draft:

```bash
make release-note VERSION=1.2.3
```

The command compares `HEAD` with the nearest reachable version tag and writes
`release-notes/v1.2.3.md`. It uses the configured `gitee ai` provider to turn
the commit log, diff statistics, changed-file list, and a truncated patch into
concise user-facing notes. Review and edit the file before publishing; source
changes included in the comparison are sent to that configured AI provider.

For unusual histories, set the comparison tag explicitly. To generate a local
commit-based draft without sending source data to an AI provider, disable AI:

```bash
make release-note VERSION=1.2.3 PREVIOUS_TAG=v1.1.0
make release-note VERSION=1.2.3 RELEASE_NOTE_AI=0
```

The command will not overwrite an existing draft. Set `RELEASE_NOTE_FORCE=1`
when intentional regeneration is required. Update `CHANGELOG.md` separately,
then commit both files and verify the version follows Semantic Versioning.

3. Create an annotated tag on the commit being released:

```bash
git tag -a v1.2.3 -m "gitee-cli v1.2.3"
```

4. To inspect the release archives locally without publishing, run:

```bash
make release VERSION=1.2.3
```

This optional check validates formatting and module tidiness, runs tests, the
race detector, and `go vet`, then writes six platform archives and
`checksums.txt` to `dist/`.

5. Validate all npm packages without publishing:

```bash
make npm-copy-binaries
NPM_VERSION=1.2.3 NPM_TOKEN=validation-only NPM_DRY_RUN=1 ./scripts/publish-npm.sh
```

## Publish

Push the commit and tag:

```bash
git push origin main
git push origin v1.2.3
```

Create or reuse the Gitee Release, synchronize its body from the reviewed
release-note file, and upload all missing attachments. This single command runs
the complete validation and build before publishing:

```bash
make release-publish VERSION=1.2.3
```

The default notes file is `release-notes/v1.2.3.md`; override
`RELEASE_NOTES_FILE` to use another reviewed Markdown file. The command uses
`GITEE_TOKEN` when set, otherwise it reads the token saved by `gitee auth login`.
It calls `PATCH /repos/{owner}/{repo}/releases/{id}` to synchronize the release
body and `POST /repos/{owner}/{repo}/releases/{release_id}/attach_files` for
uploads.
Existing attachments with the same filename are skipped, so an interrupted
publish can be resumed by running the same command again.

Publish npm packages only after the binary release succeeds:

```bash
make npm-copy-binaries
NPM_VERSION=1.2.3 NPM_TOKEN=... ./scripts/publish-npm.sh
```

Pre-release versions such as `1.2.3-rc.1` use the npm `next` dist-tag by
default. Set `NPM_TAG` explicitly to override it. The script validates every
package before publishing, publishes the main package last, and skips exact
versions already present so an interrupted release can be resumed.

Finally, verify the checksums, install through every published channel, and
run `gitee version` plus one authenticated read command.
