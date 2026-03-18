<div align="center">

<img src="assets/logo.svg" alt="IDXLens" width="120">

# IDXLens

**Extract structured financial data from Indonesia Stock Exchange (IDX) PDF reports**

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![CI](https://github.com/lugassawan/idxlens/actions/workflows/ci.yml/badge.svg)](https://github.com/lugassawan/idxlens/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![Version](https://img.shields.io/badge/version-1.0.0-blue)

</div>

IDXLens is a CLI tool that parses IDX financial report PDFs and outputs structured data in JSON or CSV. No network calls -- everything runs locally.

## Features

- **Classify** IDX PDF reports by type (balance sheet, income statement, cash flow, equity changes)
- **Extract financial data** with auto-classification and dictionary-based label matching
- **Extract raw text** from PDFs for inspection and debugging
- **Batch processing** with bounded concurrency for multiple files
- **Bilingual support** for Indonesian and English report labels (PSAK/IFRS)
- **Multiple output formats**: JSON (with pretty-print) and CSV
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
# Classify a report
idxlens classify report.pdf

# Extract financial data (auto-detect type, JSON output)
idxlens extract financial report.pdf

# Extract raw text
idxlens extract text report.pdf --pages "1-3"

# Batch process a directory
idxlens batch "reports/*.pdf" --workers 8 --output-dir results/
```

## Documentation

- [Getting Started](docs/getting-started.md) -- installation and first steps
- [CLI Reference](docs/cli-reference.md) -- all commands and flags
- [Architecture](docs/architecture.md) -- pipeline design and layer details
- [Dictionaries](docs/dictionaries.md) -- financial label mapping
- [Contributing](docs/contributing.md) -- development setup and guidelines
- [Examples](docs/examples/basic-extraction.md) -- usage examples

## Contributing

See [docs/contributing.md](docs/contributing.md) for development setup, code style, and PR guidelines.

## License

[Apache License 2.0](LICENSE)
