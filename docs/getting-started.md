# Getting Started

IDXLens is a CLI tool for extracting structured financial data from Indonesia Stock Exchange (IDX) PDF reports. It converts unstructured PDF tables into clean, machine-readable JSON or CSV.

## Installation

### Quick install (recommended)

```sh
curl -fsSL https://raw.githubusercontent.com/lugassawan/idxlens/main/scripts/install.sh | bash
```

This downloads the latest release binary for your OS and architecture, installs it to `~/.local/bin`, and sets up your PATH.

To install a specific version:

```sh
VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/lugassawan/idxlens/main/scripts/install.sh | bash
```

### Using `go install`

```sh
go install github.com/lugassawan/idxlens/cmd/idxlens@latest
```

This installs the `idxlens` binary to your `$GOPATH/bin` directory.

### Binary download

Download a prebuilt binary from the [GitHub Releases](https://github.com/lugassawan/idxlens/releases) page. Binaries are available for:

| OS      | Architecture |
|---------|-------------|
| Linux   | amd64, arm64 |
| macOS   | amd64, arm64 |
| Windows | amd64        |

Extract the archive and place the `idxlens` binary somewhere in your `$PATH`.

### Build from source

```sh
git clone https://github.com/lugassawan/idxlens.git
cd idxlens
make build
```

The binary is written to `bin/idxlens`.

## Verify installation

```sh
idxlens version
```

Expected output:

```
idxlens v0.1.0 (commit: abc1234, built: 2025-01-01T00:00:00Z)
```

## Quick start

### Classify a report

Identify what type of financial report a PDF contains:

```sh
idxlens classify report.pdf
```

Output:

```
Type:       balance-sheet
Confidence: 95%
Language:   id
```

### Extract financial data

Extract structured financial data from a PDF:

```sh
idxlens extract financial report.pdf
```

This runs the full pipeline (PDF parsing, layout analysis, classification, table detection, financial mapping) and outputs JSON to stdout.

### Specify report type

If auto-classification is not needed or gives unexpected results, specify the type explicitly:

```sh
idxlens extract financial report.pdf --type balance-sheet
```

### Change output format

```sh
# JSON (default)
idxlens extract financial report.pdf --format json

# Pretty-printed JSON
idxlens extract financial report.pdf --format json --pretty

# CSV
idxlens extract financial report.pdf --format csv
```

### Save to a file

```sh
idxlens extract financial report.pdf --output result.json
```

### Extract ESG/GRI data

Extract GRI content index disclosures from a sustainability report:

```sh
idxlens extract esg sustainability-report.pdf
```

This outputs GRI disclosures as JSON, including disclosure numbers, titles, page references, and reporting status.

### Extract raw text

Extract text lines from a PDF without financial parsing:

```sh
idxlens extract text report.pdf
```

Extract specific pages:

```sh
idxlens extract text report.pdf --pages "1-3,5"
```

## Next steps

- [CLI Reference](cli-reference.md) -- all commands and flags
- [Architecture](architecture.md) -- how the processing pipeline works
- [Examples](examples/basic-extraction.md) -- detailed usage examples
