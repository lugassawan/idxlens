# IDXLens

CLI tool for extracting structured financial data from Indonesia Stock Exchange (IDX) reports (XLSX, XBRL, PDF).

## Tech Stack

- **Language**: Go 1.26 (pure Go, single static binary, no CGO)
- **CLI**: Cobra
- **PDF**: pdfcpu
- **XLSX**: excelize
- **Browser automation**: chromedp (headless Chrome for IDX portal)
- **Linting**: golangci-lint v2 with custom plugin (tidygo)
- **Git hooks**: `.githooks/` with `core.hooksPath`
- **Tool versioning**: mise (Go, golangci-lint, golines)
- **Release**: goreleaser (cross-platform: linux/darwin/windows x amd64/arm64)

## Directory Layout

```
idxlens/
├── cmd/idxlens/        # Entry point (main.go calls cli.Execute())
├── internal/
│   ├── cli/            # Cobra CLI commands (auth, list, fetch, extract, analyze, upgrade, version)
│   ├── service/        # Orchestration service (registry, presentation extraction, analyze pipeline)
│   ├── idx/            # IDX API client (auth, listing, fetching, downloading, retry)
│   ├── upgrade/        # Self-update from GitHub Releases
│   ├── safefile/       # Atomic file write utilities
│   ├── xlsx/           # XLSX parser (excelize-based financial statement extraction)
│   ├── xbrl/           # XBRL parser (ZIP-based taxonomy extraction)
│   ├── domain/         # Presentation KV extractor (PDF key-value pair extraction)
│   ├── layout/         # Text & Layout Engine (PDF text block analysis)
│   └── pdf/            # PDF Parser (pdfcpu)
├── registry/           # Report registry data (presentations.json)
├── docs/               # Documentation site
├── testdata/           # Sample PDFs for testing
├── Dockerfile          # Docker image with Chrome for headless auth
├── .github/workflows/  # CI/CD pipelines
└── .githooks/          # Git hooks (pre-commit, commit-msg)
```

### Architecture

- All logic in `internal/` — no public API surface
- Extraction pipeline: `cli/` -> `service/` -> `idx/`/`xlsx/`/`xbrl/` -> (IDX API or local files)
- Extractor registry: `cli/extractor.go` defines `Extractor` interface + `map[string]Extractor` registry (OCP pattern)
- Analyze pipeline: `cli/analyze.go` (thin wrapper) -> `service/analyze.go` (`FetchForAnalyze`) -> `idx/` (fetch + retry)
- Presentation extraction: `cli/` -> `service/` -> `domain/kvextractor` -> `layout/` -> `pdf/`
- Self-update: `cli/` -> `upgrade/` -> GitHub Releases API
- Atomic file writes: `idx/` and `upgrade/` use `safefile/` for safe downloads
- HTTP retry: `idx/retry.go` wraps API calls with exponential backoff (3 attempts, retries on 429/5xx)
- Each layer defines interfaces at its boundary; implementations live in the layer below
- `cmd/idxlens/main.go` is Framework layer — it only calls `cli.Execute()`
- Common flag helpers: `cli/flags.go` provides `registerYearPeriodFlags`, `registerOutputFlags`, `parseYearPeriodFlags`, `parseOutputFlags`
- `IDXLENS_HOME` env var controls local cache directory (default: `~/.idxlens`)

## Commands

```sh
make init       # Install tools, build custom linter, configure git hooks
make build      # Build binary -> bin/idxlens
make lint       # Build custom linter + run golangci-lint with tidygo
make fmt        # Auto-format Go code (gofmt + golines)
make test       # Run all tests
make coverage   # Generate coverage report -> coverage/
make clean      # Remove build artifacts
docker build -t idxlens .    # Build Docker image
```

### CLI Commands

```sh
idxlens auth              # Authenticate with IDX portal (headless Chrome)
idxlens list TICKER       # List available reports for a ticker
idxlens fetch TICKER      # Download reports to local cache
idxlens fetch TICKER --dry-run  # Preview files without downloading
idxlens extract FILE      # Extract financial data from XLSX/XBRL/PDF
idxlens analyze TICKER    # Full pipeline: fetch + extract (best format)
idxlens upgrade           # Self-update from GitHub Releases
idxlens version           # Print version information
```

### Global Flags

- `--verbose` — enable verbose output (slog-based structured logging to stderr)

### Environment Variables

- `IDXLENS_HOME` — local cache directory (default: `~/.idxlens`)
- `IDXLENS_AUTH_TIMEOUT` — authentication timeout duration (default: `30s`, e.g. `60s`, `2m`)
- `NO_COLOR` — disable colored output in banner

## Conventions

- **Commits**: `type: description` — valid types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`, `chore`, `revert` (enforced by `.githooks/commit-msg`; no scopes, no `!`)
- **Commit splitting**: Split changes into logical commits — separate infra/config, core logic, tests, and wiring. Never bundle unrelated changes into a single commit.
- **Direct commits to main/master are blocked** by the pre-commit hook
- **Go**: Standard library style, `gofmt` formatting, tab indentation
- **Branches**: `feat/`, `fix/`, `chore/` prefixes
- **Worktrees**: Do NOT default to worktrees. Only use git worktrees when running parallel agents working on independent tasks simultaneously — a single agent on one task should use a regular branch, never a worktree. Never target main/master (pre-commit hook blocks direct commits).
- **Lint warnings**: Always fix the root cause before considering suppression. Refactor code, extract helpers, or restructure to satisfy the linter. Only use `//nolint` as a last resort, and always include a justification comment.
- **Code review**: Run code review before creating PRs unless one was already performed in the current session.
- **PRs**: Title uses `type: description` (same types as commits); body follows `.github/pull_request_template.md`
- **Internal only**: All packages live under `internal/` — no public API surface
- **Error wrapping**: Use `fmt.Errorf("context: %w", err)` for error chains
- **Tests**: Table-driven tests with `t.Run()` subtests, standard `testing` package only (no testify)
- **Environment variables**: All env var names are centralized as constants (`envHome` in `idx/home.go`, `envAuthTimeout` in `idx/auth.go`, `envNoColor` in `cli/constants.go`)
- **IDXLENS_HOME**: Controls local cache directory (default: `~/.idxlens`); used by auth, fetch, and analyze commands
- **Naming**: Standard Go naming conventions (MixedCaps, no underscores in names)

## Custom Linter (tidygo)

External module `github.com/lugassawan/tidygo` registered as a golangci-lint v2 module plugin via `.custom-gcl.yml`. Contains five analyzers:

- **funcname**: forbids underscores in function names
- **maxparams**: forbids functions with >7 parameters
- **nolateconst**: forbids package-level const/var declarations after function declarations
- **nolocalstruct**: forbids named struct declarations inside function bodies
- **nolateexport**: forbids exported standalone functions after unexported ones

`make lint` builds the custom binary (`custom-gcl`) via `.custom-gcl.yml` before running.

## Testing

- Standard library `testing` package — no testify
- Table-driven tests with `t.Run()` subtests
- Test files: `*_test.go` alongside source files
- Run: `make test` or `go test ./...`
- Coverage: `make coverage` -> `coverage/`

## CI Pipeline

- `.github/workflows/ci.yml` — runs on push/PRs: lint (ubuntu only), vet, test (race), build, binary size check
- `.github/workflows/release.yml` — triggered by `v*` tags: goreleaser cross-platform build, creates GitHub Release
- Uses `jdx/mise-action@v2` for tool version management in CI
