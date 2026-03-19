# Architecture

IDXLens extracts structured financial data from IDX reports (XLSX, XBRL, PDF) with automated downloading from the IDX portal. All logic lives under `internal/` with no public API surface.

## Layer overview

```
┌─────────────────────────────────────────────────┐
│  Framework: cmd/idxlens/main.go                 │  Calls cli.Execute()
├─────────────────────────────────────────────────┤
│  CLI              internal/cli/                 │  Cobra commands, flag parsing, IO
├─────────────────────────────────────────────────┤
│  Service          internal/service/             │  Registry, presentation extraction
├─────────────────────────────────────────────────┤
│  IDX API          internal/idx/                 │  Auth, listing, fetching, downloading
│  Upgrade          internal/upgrade/             │  GitHub Releases self-update
│  SafeFile         internal/safefile/            │  Atomic file writes
├─────────────────────────────────────────────────┤
│  Extraction       internal/{xlsx,xbrl}/         │  Format-specific parsing
├─────────────────────────────────────────────────┤
│  PDF Pipeline     internal/pdf/                 │  Raw PDF parsing (pdfcpu)
│                   internal/layout/              │  Text grouping, line detection
│                   internal/domain/              │  Presentation KV extraction
└─────────────────────────────────────────────────┘
```

## Dependency rule

Dependencies flow strictly downward. Each layer may only import from layers below it. No layer imports from a layer above it. This keeps the architecture testable and maintainable.

Key dependency chains:

```
cli -> service -> idx -> safefile
cli -> service -> {xlsx, xbrl}
cli -> service -> domain -> layout -> pdf
cli -> upgrade -> safefile
```

## Data flows

### Auth flow

```
idxlens auth
  │
  ▼
cli/auth.go  →  idx.NewAuthenticatedClient()  →  chromedp (headless Chrome)
  │                                                    │
  ▼                                                    ▼
Stores session to IDXLENS_HOME                  IDX portal login
```

### List / Fetch flow

```
idxlens list BBCA,BMRI -y 2024
idxlens fetch BBCA,BMRI -y 2024 -p Q3
  │
  ▼
cli/{list,fetch}.go  →  idx.Client (parallel per ticker)
  │                         │
  ▼                         ▼
Print report list      Download to IDXLENS_HOME (via safefile atomic writes)
```

### Extract flow

```
idxlens extract report.xlsx
idxlens extract report.zip
idxlens extract presentation.pdf --mode presentation
  │
  ▼
cli/extract.go  →  service layer (format detection)
  │
  ├──  XLSX  →  xlsx.Parse()
  ├──  XBRL  →  xbrl.Parse()
  └──  PDF   →  domain/kvextractor  →  layout.Analyzer  →  pdf.Reader
  │
  ▼
JSON output (stdout or file)
```

### Analyze pipeline

```
idxlens analyze BBCA -y 2024 -p Q3
  │
  ▼
cli/analyze.go  →  idx.Client.Fetch() (if not cached)
  │
  ▼
service layer  →  Try XLSX → XBRL → PDF (best available format)
  │
  ▼
JSON output
```

### Upgrade flow

```
idxlens upgrade
  │
  ▼
cli/upgrade.go  →  upgrade.Updater
  │
  ▼
GitHub Releases API  →  Download binary  →  safefile atomic replace
```

## Package details

### CLI (`internal/cli/`)

Cobra-based command definitions. Wires the pipeline together, handles flag parsing, and manages I/O.

**Commands:** `auth`, `list`, `fetch`, `extract`, `analyze`, `upgrade`, `version`

### Service (`internal/service/`)

Orchestration layer between CLI and lower packages.

- **Registry provider**: Loads report registry data for presentation extraction
- **Presentation extraction**: Coordinates the PDF pipeline (domain -> layout -> pdf)

### IDX API (`internal/idx/`)

Client for the IDX portal API. Uses `NewAuthenticatedClient()` factory for session management.

- **Auth**: Headless Chrome login via chromedp
- **Listing**: Query available reports by ticker, year, period
- **Fetching**: Download reports with parallel workers
- **Downloading**: Atomic file writes via safefile

### Upgrade (`internal/upgrade/`)

Self-update mechanism using the GitHub Releases API.

- Check for latest version
- Download platform-specific binary
- Atomic binary replacement via safefile

### SafeFile (`internal/safefile/`)

Atomic file write utilities used by both `idx` (downloads) and `upgrade` (binary replacement). Writes to a temporary file first, then atomically renames to the target path.

### Extraction (`internal/xlsx/`, `internal/xbrl/`)

Format-specific parsers for financial report extraction.

- **xlsx**: Excelize-based financial statement extraction from XLSX files
- **xbrl**: ZIP-based taxonomy extraction from XBRL archives

### PDF Pipeline (`internal/pdf/`, `internal/layout/`, `internal/domain/`)

Three-layer pipeline for extracting key-value pairs from corporate presentations.

- **pdf**: Wraps pdfcpu to extract raw text with position coordinates
- **layout**: Groups text elements into lines and blocks based on spatial relationships
- **domain**: KV extractor that identifies key-value pairs from the structured layout

## Design principles

- **Pure Go, no CGO**: Single static binary with no external dependencies at runtime
- **Internal only**: All packages live under `internal/` -- no public API surface
- **Interface-driven**: Cross-layer boundaries use interfaces for testability and decoupling
- **Atomic writes**: File operations use safefile to prevent partial writes
- **Local caching**: Downloaded reports are cached in `IDXLENS_HOME` (default: `~/.idxlens`)
