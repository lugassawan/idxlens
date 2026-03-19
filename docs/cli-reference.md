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

| Type                       | Description                      |
|----------------------------|----------------------------------|
| `balance-sheet`            | Statement of Financial Position  |
| `income-statement`         | Statement of Profit or Loss      |
| `cash-flow`                | Statement of Cash Flows          |
| `equity-changes`           | Statement of Changes in Equity   |
| `sustainability-report`    | ESG/Sustainability Report        |
| `annual-report`            | Annual Report                    |
| `corporate-presentation`   | Corporate Presentation           |
| `auditor-report`           | Independent Auditor's Report     |
| `notes`                    | Notes to Financial Statements    |

---

### `extract`

Parent command for data extraction subcommands.

```sh
idxlens extract [subcommand]
```

---

### `extract financial`

Extract structured financial data from an IDX PDF report. Runs the full L0-L4 pipeline: PDF parsing, layout analysis, document classification, table detection, financial statement mapping, and output formatting.

Supports banking-specific line items (NIM, NPL, CASA, DPK, etc.), bilingual tab-separated column detection for side-by-side Indonesian/English layouts, presentation-style "Rp tn" unit detection, and abbreviated period parsing (e.g., "Dec-25", "FY-25", "3Q-25").

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

---

### `extract esg`

Extract ESG/GRI content index data from a sustainability report. Scans tables for GRI disclosure numbers and extracts structured data including disclosure number, title, description, page references, and reporting status.

```sh
idxlens extract esg [pdf-path]
```

**Arguments:**

| Argument   | Description              |
|-----------|--------------------------|
| `pdf-path` | Path to the PDF file     |

**Examples:**

```sh
# Extract GRI disclosures from a sustainability report
idxlens extract esg sustainability-report.pdf
```

**Output (JSON):**

```json
{
  "disclosures": [
    {
      "number": "201-1",
      "title": "Direct economic value generated and distributed",
      "description": "",
      "page_ref": "45",
      "status": "reported"
    }
  ],
  "framework": "GRI"
}
```

---

### `batch`

Process multiple PDF files matching a glob pattern using bounded concurrency. Each file runs through the full extraction pipeline and results are written to the specified output directory.

```sh
idxlens batch [glob-pattern]
```

**Arguments:**

| Argument       | Description                                |
|---------------|--------------------------------------------|
| `glob-pattern` | File glob pattern (e.g. `"reports/*.pdf"`) |

**Flags:**

| Flag            | Short | Default | Description                                              |
|-----------------|-------|---------|----------------------------------------------------------|
| `--workers`     | `-w`  | `4`     | Number of concurrent workers (capped at CPU count)       |
| `--output-dir`  | `-d`  |         | Output directory for results                             |
| `--format`      | `-f`  | `json`  | Output format (`json`, `csv`)                            |
| `--type`        | `-t`  | (auto)  | Report type (e.g. `balance-sheet`, `income-statement`)   |

**Examples:**

```sh
# Process all PDFs in reports/ with default settings
idxlens batch "reports/*.pdf"

# Use 8 workers and save results to output/
idxlens batch "reports/*.pdf" --workers 8 --output-dir output/

# Process as CSV with explicit type
idxlens batch "data/*.pdf" --format csv --type balance-sheet
```

The command outputs a JSON summary with the count of successful and failed files.
