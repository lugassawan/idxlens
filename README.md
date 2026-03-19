<div align="center">

<img src="assets/logo.svg" alt="IDXLens" width="120">

# IDXLens

**Extract structured financial data from Indonesia Stock Exchange (IDX) reports (XLSX, XBRL, PDF)**

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![CI](https://github.com/lugassawan/idxlens/actions/workflows/ci.yml/badge.svg)](https://github.com/lugassawan/idxlens/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-%E2%89%A585%25-brightgreen)](https://github.com/lugassawan/idxlens/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/lugassawan/idxlens)](https://github.com/lugassawan/idxlens/releases)

</div>

IDXLens is a CLI tool that fetches and extracts structured financial data from IDX reports. Supports XLSX, XBRL, and PDF formats with automated downloading from the IDX portal.

## Features

- **Authenticate** with IDX portal via headless Chrome
- **List and fetch** available financial reports for any ticker
- **Extract financial data** from XLSX (excelize), XBRL (ZIP archives), and PDF presentations
- **Full pipeline** (`analyze`): fetch if needed, then extract from the best available format (XLSX > XBRL > PDF)
- **Presentation KV extraction** for corporate presentations (key-value pair detection from PDF layout)
- **Local caching** via `IDXLENS_HOME` (default: `~/.idxlens`)
- **JSON output** with optional pretty-print
- **Self-update** via `idxlens upgrade` from GitHub Releases
- **Pure Go** -- single static binary, no runtime dependencies

## Quick Install

```sh
# One-line install (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/lugassawan/idxlens/main/scripts/install.sh | bash

# Using go install
go install github.com/lugassawan/idxlens/cmd/idxlens@latest

# Or download a prebuilt binary from GitHub Releases
```

## Usage

```sh
# Authenticate with IDX portal
idxlens auth

# List available reports for a ticker
idxlens list BBCA -y 2024

# Fetch reports to local cache
idxlens fetch BBCA -y 2024 -p Q3

# Extract financial data from a local file
idxlens extract path/to/report.xlsx --pretty
idxlens extract path/to/report.zip   # XBRL ZIP
idxlens extract path/to/presentation.pdf --mode presentation

# Full pipeline: fetch (if needed) + extract
idxlens analyze BBCA -y 2024 -p Q3

# Self-update to latest version
idxlens upgrade
```

## Documentation

- [Getting Started](docs/getting-started.md) -- installation and first steps
- [CLI Reference](docs/cli-reference.md) -- all commands and flags
- [Architecture](docs/architecture.md) -- pipeline design and layer details
- [Contributing](docs/contributing.md) -- development setup and guidelines

## Contributing

See [docs/contributing.md](docs/contributing.md) for development setup, code style, and PR guidelines.

## License

[Apache License 2.0](LICENSE)
