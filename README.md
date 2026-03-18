# IDXLens

[![CI](https://github.com/lugassawan/idxlens/actions/workflows/ci.yml/badge.svg)](https://github.com/lugassawan/idxlens/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

CLI tool for extracting structured financial data from Indonesia Stock Exchange (IDX) PDF reports. Converts unstructured PDF tables into clean, machine-readable JSON or CSV.

## Features

- **Classify** IDX PDF reports by type (balance sheet, income statement, cash flow, equity changes)
- **Extract financial data** with auto-classification and dictionary-based label matching
- **Extract raw text** from PDFs for inspection and debugging
- **Batch processing** with bounded concurrency for multiple files
- **Bilingual support** for Indonesian and English report labels
- **Multiple output formats**: JSON (with pretty-print) and CSV
- **Pure Go** -- single static binary, no runtime dependencies

## Installation

### Binary download

Download a prebuilt binary from the [GitHub Releases](https://github.com/lugassawan/idxlens/releases) page. Binaries are available for Linux, macOS, and Windows (amd64/arm64).

### Using `go install`

```sh
go install github.com/lugassawan/idxlens/cmd/idxlens@latest
```

### Build from source

```sh
git clone https://github.com/lugassawan/idxlens.git
cd idxlens
make build
```

The binary is written to `bin/idxlens`.

## Quick Start

### Classify a report

```sh
idxlens classify report.pdf
```

```
Type:       balance-sheet
Confidence: 95%
Language:   id
```

### Extract financial data

```sh
# Auto-detect type, output JSON to stdout
idxlens extract financial report.pdf

# Pretty-printed JSON
idxlens extract financial report.pdf --pretty

# CSV output to file
idxlens extract financial report.pdf --format csv --output data.csv
```

### Extract raw text

```sh
idxlens extract text report.pdf --pages "1-3"
```

### Batch processing

```sh
# Process all PDFs in a directory with 8 workers
idxlens batch "reports/*.pdf" --workers 8 --output-dir results/
```

## Architecture

IDXLens processes PDFs through a six-layer pipeline:

```
L5 CLI -> L4 Output -> L3 Domain -> L2 Table -> L1 Layout -> L0 PDF
```

Each layer has a single responsibility and communicates through interfaces. Dependencies flow strictly downward.

See [docs/architecture.md](docs/architecture.md) for full details.

## Documentation

- [Getting Started](docs/getting-started.md) -- installation and first steps
- [CLI Reference](docs/cli-reference.md) -- all commands and flags
- [Architecture](docs/architecture.md) -- pipeline design and layer details
- [Dictionaries](docs/dictionaries.md) -- financial label mapping
- [Contributing](docs/contributing.md) -- development setup and guidelines
- [Examples](docs/examples/basic-extraction.md) -- usage examples

## Contributing

See [docs/contributing.md](docs/contributing.md) for development setup, code style, and PR guidelines.

```sh
make init    # Install tools, configure hooks
make test    # Run tests
make lint    # Run linter
```

## License

[Apache License 2.0](LICENSE)
