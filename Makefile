# ── project ───────────────────────────────────────────────────────────────────
BINARY  := brio
MODULE  := github.com/luca-trifilio/brio
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(MODULE)/internal/cli.version=$(VERSION) \
	-X $(MODULE)/internal/cli.commit=$(COMMIT) \
	-X $(MODULE)/internal/cli.date=$(DATE)

# ── colours ───────────────────────────────────────────────────────────────────
BOLD  := $(shell tput bold  2>/dev/null || true)
RESET := $(shell tput sgr0  2>/dev/null || true)
GREEN := $(shell tput setaf 2 2>/dev/null || true)
CYAN  := $(shell tput setaf 6 2>/dev/null || true)
RED   := $(shell tput setaf 1 2>/dev/null || true)

.DEFAULT_GOAL := help

# ── help ──────────────────────────────────────────────────────────────────────
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(BOLD)$(CYAN)brio$(RESET) dev commands\n\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-18s$(RESET) %s\n", $$1, $$2 } \
		/^##@/ { printf "\n$(BOLD)%s$(RESET)\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo ""

##@ Development

.PHONY: build
build: ## Build the binary for the current platform
	@go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .
	@echo "$(GREEN)✓$(RESET) $(BINARY) $(VERSION)"

.PHONY: install
install: ## Install to GOPATH/bin
	@go install -trimpath -ldflags="$(LDFLAGS)" .
	@echo "$(GREEN)✓$(RESET) installed $(BINARY) $(VERSION)"

.PHONY: run
run: build ## Build and run (pass ARGS="..." for arguments)
	@./$(BINARY) $(ARGS)

##@ Quality

.PHONY: test
test: ## Run all tests with race detector and coverage
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: test-short
test-short: ## Run tests without race detector (fast iteration)
	@go test -short ./...

.PHONY: cover
cover: test ## Open coverage report in the browser
	@go tool cover -html=coverage.txt

.PHONY: lint
lint: ## Run golangci-lint
	@golangci-lint run ./...

.PHONY: fmt
fmt: ## Format all Go source files
	@gofmt -s -w .
	@command -v goimports >/dev/null && goimports -w . || true

.PHONY: vet
vet: ## Run go vet
	@go vet ./...

.PHONY: check
check: fmt vet lint test ## Run all quality gates (fmt + vet + lint + test)
	@echo "$(GREEN)✓$(RESET) all checks passed"

##@ Release

.PHONY: snapshot
snapshot: ## Build a local snapshot (no git tag required, nothing published)
	@goreleaser release --snapshot --clean

.PHONY: release-dry
release-dry: ## Dry-run the full release pipeline (no publish)
	@goreleaser release --skip=publish --clean

.PHONY: tag
tag: ## Emergency: manually create a release tag — usage: make tag VERSION=1.2.3
ifndef VERSION
	$(error VERSION is required. Usage: make tag VERSION=1.2.3)
endif
	@git diff --exit-code > /dev/null || \
		(echo "$(RED)✗ Uncommitted changes — commit or stash first$(RESET)" && exit 1)
	@echo "$(BOLD)NOTE: under normal flow, release-please creates tags automatically$(RESET)"
	@echo "$(BOLD)      when the Release PR is merged. Only use this as an escape hatch.$(RESET)"
	@read -p "Tag v$(VERSION) and push? [y/N] " ans && [ "$$ans" = "y" ]
	@git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	@git push origin "v$(VERSION)"
	@echo "$(GREEN)✓$(RESET) pushed v$(VERSION)"

##@ Maintenance

.PHONY: tidy
tidy: ## Tidy go modules
	@go mod tidy
	@echo "$(GREEN)✓$(RESET) go mod tidy"

.PHONY: clean
clean: ## Remove build artefacts
	@rm -f $(BINARY) coverage.txt
	@rm -rf dist/
	@echo "$(GREEN)✓$(RESET) cleaned"

.PHONY: setup
setup: ## Install required dev tools
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@echo ""
	@echo "The following tools are best installed via Homebrew:"
	@echo "  brew install goreleaser cosign gh lefthook"
	@echo ""
	@lefthook install
	@echo "$(GREEN)✓$(RESET) setup complete"
