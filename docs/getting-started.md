# Getting Started

IDXLens is a CLI tool for fetching and extracting structured financial data from Indonesia Stock Exchange (IDX) reports. Supports XLSX, XBRL, and PDF formats with automated downloading from the IDX portal.

## Installation

### Quick install (recommended)

```sh
curl -fsSL https://raw.githubusercontent.com/lugassawan/idxlens/main/scripts/install.sh | bash
```

This downloads the latest release binary for your OS and architecture, installs it to `~/.local/bin`, and sets up your PATH.

To install a specific version:

```sh
VERSION=v1.1.0 curl -fsSL https://raw.githubusercontent.com/lugassawan/idxlens/main/scripts/install.sh | bash
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
idxlens v1.1.0 (commit: abc1234, built: 2026-01-01T00:00:00Z)
```

## Quick start

### 1. Authenticate with IDX portal

```sh
idxlens auth
```

This launches a headless Chrome session to log in to the IDX portal. Chrome must be installed. The session is stored in `~/.idxlens` (or `IDXLENS_HOME` if set).

### 2. List available reports

```sh
idxlens list BBCA -y 2024
```

This queries the IDX API for available reports. Filter by year and period:

```sh
idxlens list BBCA -y 2024 -p Q3
```

### 3. Fetch reports

```sh
idxlens fetch BBCA -y 2024 -p Q3
```

This downloads reports to the local cache at `~/.idxlens/reports/BBCA/`.

### 4. Extract financial data

```sh
# From a local XLSX file
idxlens extract ~/.idxlens/reports/BBCA/2024-Q3.xlsx --pretty

# From a XBRL ZIP archive
idxlens extract ~/.idxlens/reports/BBCA/2024-Q3.zip

# Presentation KV extraction from PDF
idxlens extract presentation.pdf --mode presentation
```

### Or use the full pipeline

The `analyze` command combines fetch and extract into a single step:

```sh
idxlens analyze BBCA -y 2024 -p Q3 --pretty
```

This fetches reports (if not cached) and extracts from the best available format (XLSX > XBRL > PDF).

Analyze multiple tickers at once:

```sh
idxlens analyze BBCA,BMRI,BBRI -y 2024 -p Q3
```

### Keep IDXLens up to date

```sh
idxlens upgrade
```

This checks for the latest release and updates the binary in place.

## Next steps

- [CLI Reference](cli-reference.md) -- all commands and flags
- [Architecture](architecture.md) -- pipeline design and package details
- [Examples](examples/basic-extraction.md) -- detailed usage examples
