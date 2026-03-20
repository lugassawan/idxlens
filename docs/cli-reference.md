# CLI Reference

IDXLens provides commands for fetching and extracting structured financial data from IDX reports.

## Global

```
idxlens [command]
```

IDXLens is a CLI tool that fetches and extracts structured financial data from Indonesia Stock Exchange (IDX) reports. Supports XLSX, XBRL, and PDF formats.

## Global Flags

| Flag        | Default | Description                                      |
|------------|---------|--------------------------------------------------|
| `--verbose` | `false` | Enable verbose output (structured logging to stderr) |

## Environment

| Variable              | Default       | Description                              |
|----------------------|---------------|------------------------------------------|
| `IDXLENS_HOME`        | `~/.idxlens` | Local cache and session directory        |
| `IDXLENS_AUTH_TIMEOUT` | `30s`        | Authentication timeout (e.g. `60s`, `2m`) |
| `NO_COLOR`            |               | Disable colored output when set          |

## Commands

### `auth`

Authenticate with the IDX portal via headless Chrome. Stores the session locally for use by `list`, `fetch`, and `analyze` commands.

```sh
idxlens auth
```

Chrome must be installed. The command launches a headless browser session, navigates to the IDX portal, and stores authentication cookies (with expiry timestamps) in `IDXLENS_HOME`. The timeout can be configured via `IDXLENS_AUTH_TIMEOUT` (default: `30s`).

If cookies have expired, commands that require authentication will return a clear error message prompting you to re-authenticate.

---

### `list`

List available financial reports for one or more tickers.

```sh
idxlens list TICKER[,TICKER...]
```

**Arguments:**

| Argument  | Description                              |
|----------|------------------------------------------|
| `TICKER`  | One or more ticker symbols (comma-separated) |

**Flags:**

| Flag       | Short | Default | Description                              |
|-----------|-------|---------|------------------------------------------|
| `--year`   | `-y`  |         | Filter by reporting year                 |
| `--period` | `-p`  |         | Filter by period (`Q1`, `Q2`, `Q3`, `FY`) |

**Examples:**

```sh
# List all reports for BBCA
idxlens list BBCA

# Filter by year
idxlens list BBCA -y 2024

# Filter by year and period
idxlens list BBCA,BMRI -y 2024 -p Q3
```

---

### `fetch`

Download financial reports to the local cache.

```sh
idxlens fetch TICKER[,TICKER...]
```

**Arguments:**

| Argument  | Description                              |
|----------|------------------------------------------|
| `TICKER`  | One or more ticker symbols (comma-separated) |

**Flags:**

| Flag          | Short | Default | Description                                        |
|--------------|-------|---------|----------------------------------------------------|
| `--year`      | `-y`  |         | Filter by reporting year                           |
| `--period`    | `-p`  |         | Filter by period (`Q1`, `Q2`, `Q3`, `FY`)          |
| `--file-type` |       |         | Filter by file type (`xlsx`, `xbrl`, `pdf`)        |
| `--workers`   | `-w`  | `4`     | Number of concurrent download workers              |
| `--dry-run`   |       | `false` | List files that would be downloaded without downloading |

**Examples:**

```sh
# Fetch all reports for BBCA
idxlens fetch BBCA -y 2025

# Fetch specific year and period
idxlens fetch BBCA -y 2024 -p Q3

# Preview files without downloading
idxlens fetch BBCA -y 2025 --dry-run

# Preview with file type filter
idxlens fetch BBCA -y 2025 --dry-run --file-type xlsx

# Fetch only XLSX files with 8 workers
idxlens fetch BBCA,BMRI -y 2024 --file-type xlsx -w 8
```

Reports are saved to `IDXLENS_HOME/data/<ticker>/<year>/<period>/`.

---

### `extract`

Extract structured financial data from a local file or fetched report.

```sh
idxlens extract [TICKER|FILE]
```

**Arguments:**

| Argument       | Description                                    |
|---------------|------------------------------------------------|
| `TICKER|FILE`  | Ticker symbol (uses cached report) or file path |

**Flags:**

| Flag       | Short | Default | Description                                          |
|-----------|-------|---------|------------------------------------------------------|
| `--mode`   | `-m`  |         | Extraction mode (`presentation` for PDF KV extraction) |
| `--year`   | `-y`  |         | Reporting year (when using ticker)                   |
| `--period` | `-p`  |         | Reporting period (when using ticker)                 |
| `--format` | `-f`  | `json`  | Output format (`json`)                               |
| `--output` | `-o`  | stdout  | Output file path                                     |
| `--pretty` |       | `false` | Pretty-print JSON output                             |

**Examples:**

```sh
# Extract from a local XLSX file
idxlens extract path/to/report.xlsx --pretty

# Extract from a XBRL ZIP archive
idxlens extract path/to/report.zip

# Extract presentation KV pairs from a PDF
idxlens extract path/to/presentation.pdf --mode presentation

# Save output to a file
idxlens extract report.xlsx --output result.json
```

---

### `analyze`

Full pipeline: fetch reports if not cached, then extract from the best available format (XBRL > XLSX > PDF).

```sh
idxlens analyze TICKER[,TICKER...]
```

**Arguments:**

| Argument  | Description                              |
|----------|------------------------------------------|
| `TICKER`  | One or more ticker symbols (comma-separated) |

**Flags:**

| Flag       | Short | Default | Description                     |
|-----------|-------|---------|---------------------------------|
| `--year`   | `-y`  |         | Reporting year                  |
| `--period` | `-p`  |         | Reporting period                |
| `--format` | `-f`  | `json`  | Output format (`json`)          |
| `--output` | `-o`  | stdout  | Output file path                |
| `--pretty` |       | `false` | Pretty-print JSON output        |

**Examples:**

```sh
# Analyze BBCA Q3 2024
idxlens analyze BBCA -y 2024 -p Q3

# Pretty-print output
idxlens analyze BBCA -y 2024 -p Q3 --pretty

# Analyze multiple tickers
idxlens analyze BBCA,BMRI,BBRI -y 2024 -p Q3

# Save to file
idxlens analyze BBCA -y 2024 --output bbca.json
```

---

### `upgrade`

Self-update IDXLens to the latest version from GitHub Releases.

```sh
idxlens upgrade
```

Checks for the latest release, downloads the platform-specific binary, and atomically replaces the current binary.

---

### `version`

Print version information.

```sh
idxlens version
```

Output includes version tag, commit hash, and build timestamp.
