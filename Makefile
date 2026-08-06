BINARY_NAME   = gitee
NPM_VERSION   = 0.0.0
GO            = go
OSES          = darwin linux windows
ARCHS         = amd64 arm64
BUILD_OUTPUT_DIR = .
BUILD_DISTRIBUTION ?= source
RELEASE_DIR   = dist
VERSION       =
GITEE         = gitee
GITEE_HOSTNAME = gitee.com
GITEE_REPO    = oschina/gitee-cli
GITEE_API_PREFIX = https://$(GITEE_HOSTNAME)/api/v5
RELEASE_NOTES_DIR = release-notes
RELEASE_NOTES_FILE = $(RELEASE_NOTES_DIR)/$(RELEASE_TAG).md
PREVIOUS_TAG =
RELEASE_NOTE_LANGUAGE = English
RELEASE_NOTE_AI = 1
RELEASE_NOTE_FORCE = 0

RELEASE_VERSION = $(patsubst v%,%,$(VERSION))
RELEASE_TAG     = v$(RELEASE_VERSION)
RELEASE_NAME    = Gitee CLI $(RELEASE_TAG)
RELEASE_DIR_ABS = $(abspath $(RELEASE_DIR))

export NPM_VERSION NPM_TOKEN

.PHONY: build install clean lint tidy test release release-check release-note release-publish build-all-platforms npm-copy-binaries npm-publish

GIT_TAG     := $(shell git describe --tags --exact-match 2>/dev/null)
GIT_COMMIT  := $(shell git rev-parse --short HEAD)
GIT_COMMIT_FULL := $(shell git rev-parse HEAD)
GIT_VERSION := $(if $(GIT_TAG),$(GIT_TAG),dev+$(GIT_COMMIT))
BUILD_DATE  := $(shell date -u +%Y-%m-%d)

BUILD_FLAGS := -ldflags "-s -w \
  -X gitee.com/oschina/gitee-cli/internal/build.Version=$(GIT_VERSION) \
  -X gitee.com/oschina/gitee-cli/internal/build.CommitSHA=$(GIT_COMMIT) \
  -X gitee.com/oschina/gitee-cli/internal/build.Date=$(BUILD_DATE) \
  -X gitee.com/oschina/gitee-cli/internal/build.Distribution=$(BUILD_DISTRIBUTION)"

build:
	@mkdir -p ./bin
	@go build $(BUILD_FLAGS) -o ./bin/gitee ./cmd/gitee/main.go
	@echo "Build complete: ./bin/gitee"

install: build
	@mkdir -p $(HOME)/bin
	@cp ./bin/gitee $(HOME)/bin/gitee
#	@codesign --sign - $(HOME)/bin/gitee
	@echo "Installed to $(HOME)/bin/gitee"

uninstall:
	@rm -f $(HOME)/bin/gitee
	@echo "Uninstalled from $(HOME)/bin/gitee"

clean:
	@rm -rf ./bin
	@echo "Clean complete"

lint:
	@golangci-lint run ./...

test:
	@go test -v ./...

tidy:
	@go mod tidy

release-check:
	@test -n "$(RELEASE_VERSION)" || { echo "VERSION is required (for example: make release VERSION=1.2.3)" >&2; exit 2; }
	@printf '%s\n' "$(RELEASE_VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || { echo "VERSION must be valid SemVer without build metadata: $(VERSION)" >&2; exit 2; }
	@test -z "$$(git status --porcelain)" || { echo "Working tree must be clean before building a release" >&2; exit 2; }
	@git tag --points-at HEAD | grep -Fqx "$(RELEASE_TAG)" || { echo "HEAD must be tagged $(RELEASE_TAG) before building a release" >&2; exit 2; }
	@test -z "$$(gofmt -l .)" || { echo "Go files are not formatted; run gofmt before releasing" >&2; exit 2; }
	@go mod tidy -diff
	@go test ./...
	@go test -race ./...
	@go vet ./...

# Build archives and checksums for a Gitee Release.
release: release-check
	@command -v zip >/dev/null 2>&1 || { echo "zip is required to package Windows releases" >&2; exit 2; }
	@rm -rf "$(RELEASE_DIR)"
	@mkdir -p "$(RELEASE_DIR)"
	@$(MAKE) build-all-platforms BUILD_OUTPUT_DIR="$(RELEASE_DIR_ABS)/binaries" GIT_VERSION="$(RELEASE_TAG)" BUILD_DISTRIBUTION=release
	@set -eu; \
		stage_root="$(RELEASE_DIR_ABS)/stage"; \
		mkdir -p "$$stage_root"; \
		for os in $(OSES); do \
			for arch in $(ARCHS); do \
				ext=""; binary="$(BINARY_NAME)"; \
				if [ "$$os" = "windows" ]; then ext=".exe"; binary="$(BINARY_NAME).exe"; fi; \
				archive="$(BINARY_NAME)_$(RELEASE_VERSION)_$${os}_$${arch}"; \
				stage="$$stage_root/$${os}_$${arch}"; \
				mkdir -p "$$stage"; \
				cp "$(RELEASE_DIR_ABS)/binaries/$(BINARY_NAME)-$${os}-$${arch}$${ext}" "$$stage/$$binary"; \
				cp README.md LICENSE "$$stage/"; \
				if [ "$$os" = "windows" ]; then \
					(cd "$$stage" && zip -q "$(RELEASE_DIR_ABS)/$$archive.zip" README.md LICENSE "$$binary"); \
				else \
					tar -czf "$(RELEASE_DIR_ABS)/$$archive.tar.gz" -C "$$stage" README.md LICENSE "$$binary"; \
				fi; \
			done; \
		done; \
		rm -rf "$(RELEASE_DIR_ABS)/binaries" "$$stage_root"
	@cd "$(RELEASE_DIR)" && if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum ./*.tar.gz ./*.zip > checksums.txt; \
	else \
		shasum -a 256 ./*.tar.gz ./*.zip > checksums.txt; \
	fi
	@echo "Release artifacts ready in $(RELEASE_DIR)/ for $(RELEASE_TAG)."

# Generate an editable release-note draft from the previous release tag.
release-note:
	@test -n "$(RELEASE_VERSION)" || { echo "VERSION is required (for example: make release-note VERSION=1.2.3)" >&2; exit 2; }
	@printf '%s\n' "$(RELEASE_VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || { echo "VERSION must be valid SemVer without build metadata: $(VERSION)" >&2; exit 2; }
	@GITEE_BIN="$(GITEE)" \
		RELEASE_TAG="$(RELEASE_TAG)" \
		PREVIOUS_TAG="$(PREVIOUS_TAG)" \
		RELEASE_NOTES_FILE="$(RELEASE_NOTES_FILE)" \
		RELEASE_NOTE_LANGUAGE="$(RELEASE_NOTE_LANGUAGE)" \
		RELEASE_NOTE_AI="$(RELEASE_NOTE_AI)" \
		RELEASE_NOTE_FORCE="$(RELEASE_NOTE_FORCE)" \
		./scripts/generate-release-note.sh

# Create or reuse the Gitee Release, then upload missing artifacts by filename.
release-publish:
	@test -n "$(RELEASE_VERSION)" || { echo "VERSION is required (for example: make release-publish VERSION=1.2.3)" >&2; exit 2; }
	@printf '%s\n' "$(RELEASE_VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || { echo "VERSION must be valid SemVer without build metadata: $(VERSION)" >&2; exit 2; }
	@command -v curl >/dev/null 2>&1 || { echo "curl is required to publish release artifacts" >&2; exit 2; }
	@command -v jq >/dev/null 2>&1 || { echo "jq is required to publish release artifacts" >&2; exit 2; }
	@command -v "$(GITEE)" >/dev/null 2>&1 || { echo "$(GITEE) is required to publish the Gitee Release" >&2; exit 2; }
	@test -s "$(RELEASE_NOTES_FILE)" || { echo "Release notes not found: $(RELEASE_NOTES_FILE)" >&2; echo "Run make release-note VERSION=$(RELEASE_VERSION), review the draft, and commit it before publishing." >&2; exit 2; }
	@git ls-remote --exit-code --tags origin "refs/tags/$(RELEASE_TAG)" >/dev/null 2>&1 || { echo "Push $(RELEASE_TAG) to origin before publishing the release" >&2; exit 2; }
	@$(MAKE) release VERSION="$(VERSION)" RELEASE_DIR="$(RELEASE_DIR)"
	@GITEE_BIN="$(GITEE)" \
		GITEE_HOSTNAME="$(GITEE_HOSTNAME)" \
		GITEE_REPO="$(GITEE_REPO)" \
		GITEE_API_PREFIX="$(GITEE_API_PREFIX)" \
		RELEASE_TAG="$(RELEASE_TAG)" \
		RELEASE_NAME="$(RELEASE_NAME)" \
		RELEASE_TARGET="$(GIT_COMMIT_FULL)" \
		RELEASE_NOTES_FILE="$(RELEASE_NOTES_FILE)" \
		RELEASE_DIR="$(RELEASE_DIR_ABS)" \
		./scripts/publish-release.sh

# Cross-compile for all platforms
build-all-platforms:
	@set -e; \
	mkdir -p "$(BUILD_OUTPUT_DIR)"; \
	for os in $(OSES); do \
		for arch in $(ARCHS); do \
			ext=""; \
			if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
			CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" $(GO) build $(BUILD_FLAGS) \
				-o "$(BUILD_OUTPUT_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext" ./cmd/gitee/main.go; \
		done; \
	done

# Build platform binaries and copy them into npm/ directories
npm-copy-binaries:
	@$(MAKE) build-all-platforms BUILD_DISTRIBUTION=npm
	@set -e; \
	for os in $(OSES); do \
		for arch in $(ARCHS); do \
			ext=""; \
			if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
			dirname="$(BINARY_NAME)-cli-$$os-$$arch"; \
			target="$(BINARY_NAME)-cli-$$os-$$arch$$ext"; \
			mkdir -p "./npm/$$dirname/bin"; \
			cp "./$(BINARY_NAME)-$$os-$$arch$$ext" "./npm/$$dirname/bin/$$target"; \
		done; \
	done
	@echo "Binaries copied to npm/ directories."

# Publish all npm packages (requires NPM_TOKEN env var)
npm-publish: npm-copy-binaries
	@./scripts/publish-npm.sh
	@echo "npm publish complete."
