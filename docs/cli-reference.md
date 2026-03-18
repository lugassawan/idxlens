# CLI Reference

IDXLens provides commands for classifying and extracting data from IDX PDF reports.

## Global

```
idxlens [command]
```

IDXLens is a CLI tool for extracting structured financial data from Indonesia Stock Exchange (IDX) PDF reports. It converts unstructured PDF tables into clean, machine-readable formats.

## Commands

### `version`

Print version information.

```sh
idxlens version
```

Output includes version tag, commit hash, and build timestamp.

---

### `classify`

Classify an IDX PDF report by type. Analyzes the first few pages to determine the report type using heuristic matching.

```sh
idxlens classify [pdf-path]
```

**Arguments:**

| Argument   | Description              |
|-----------|--------------------------|
| `pdf-path` | Path to the PDF file     |

**Flags:**

| Flag              | Short | Default | Description                    |
|-------------------|-------|---------|--------------------------------|
| `--format`        | `-f`  | `text`  | Output format (`text`, `json`) |

**Examples:**

```sh
# Text output (default)
idxlens classify report.pdf

# JSON output
idxlens classify report.pdf --format json
```

**Text output:**

```
Type:       balance-sheet
Confidence: 95%
Language:   id
```

**JSON output:**

```json
{
  "type": "balance-sheet",
  "confidence": 0.95,
  "language": "id"
}
```

**Supported report types:**

| Type                | Description                      |
|---------------------|----------------------------------|
| `balance-sheet`     | Statement of Financial Position  |
| `income-statement`  | Statement of Profit or Loss      |
| `cash-flow`         | Statement of Cash Flows          |
| `equity-changes`    | Statement of Changes in Equity   |

---

### `extract`

Parent command for data extraction subcommands.

```sh
idxlens extract [subcommand]
```

---

### `extract financial`

Extract structured financial data from an IDX PDF report. Runs the full L0-L4 pipeline: PDF parsing, layout analysis, document classification, table detection, financial statement mapping, and output formatting.

```sh
idxlens extract financial [pdf-path]
```

**Arguments:**

| Argument   | Description              |
|-----------|--------------------------|
| `pdf-path` | Path to the PDF file     |

**Flags:**

| Flag              | Short | Default | Description                                              |
|-------------------|-------|---------|----------------------------------------------------------|
| `--type`          | `-t`  | (auto)  | Report type (e.g. `balance-sheet`, `income-statement`)   |
| `--format`        | `-f`  | `json`  | Output format (`json`, `csv`)                            |
| `--output`        | `-o`  | stdout  | Output file path                                         |
| `--pretty`        |       | `false` | Pretty-print output (JSON only)                          |

When `--type` is omitted, IDXLens auto-classifies the document. If classification fails, use `--type` to specify it explicitly.

**Examples:**

```sh
# Auto-detect type, output JSON to stdout
idxlens extract financial report.pdf

# Specify type, pretty JSON
idxlens extract financial report.pdf --type balance-sheet --pretty

# CSV output to file
idxlens extract financial report.pdf --format csv --output data.csv
```

---

### `extract text`

Extract text lines from a PDF by running the L0 (PDF parser) and L1 (layout analyzer) pipeline. Outputs one text line per line, grouped by page.

```sh
idxlens extract text [pdf-path]
```

**Arguments:**

| Argument   | Description              |
|-----------|--------------------------|
| `pdf-path` | Path to the PDF file     |

**Flags:**

| Flag       | Short | Default   | Description                           |
|-----------|-------|-----------|---------------------------------------|
| `--pages` |       | all pages | Page range (e.g. `"1-3,5,7-9"`)      |

**Examples:**

```sh
# Extract all pages
idxlens extract text report.pdf

# Extract specific pages
idxlens extract text report.pdf --pages "1-3,5"
```
