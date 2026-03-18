# Basic Extraction

This guide walks through common extraction workflows with IDXLens.

## Classify a report first

Before extracting data, you can check what type of report a PDF contains:

```sh
idxlens classify quarterly-report.pdf
```

```
Type:       balance-sheet
Confidence: 95%
Language:   id
```

For machine-readable output:

```sh
idxlens classify quarterly-report.pdf --format json
```

```json
{
  "type": "balance-sheet",
  "confidence": 0.95,
  "language": "id"
}
```

## Extract financial data

### Auto-detect report type

```sh
idxlens extract financial quarterly-report.pdf
```

IDXLens classifies the document automatically and extracts structured data. The output is JSON by default.

### Specify report type explicitly

If auto-classification does not produce the expected result, specify the type:

```sh
idxlens extract financial quarterly-report.pdf --type income-statement
```

Available types: `balance-sheet`, `income-statement`, `cash-flow`, `equity-changes`.

### Pretty-print JSON

```sh
idxlens extract financial quarterly-report.pdf --pretty
```

Sample output:

```json
{
  "type": "balance-sheet",
  "company": "PT Example Tbk",
  "periods": ["2024-12-31", "2023-12-31"],
  "currency": "IDR",
  "unit": "millions",
  "language": "id",
  "items": [
    {
      "key": "total_assets",
      "label": "Jumlah Aset",
      "section": "assets",
      "level": 0,
      "confidence": 1.0,
      "values": {
        "2024-12-31": 50000000,
        "2023-12-31": 45000000
      }
    }
  ]
}
```

### Export to CSV

```sh
idxlens extract financial quarterly-report.pdf --format csv --output data.csv
```

The CSV includes columns for label, key, section, and one column per period.

## Extract raw text

Use `extract text` to inspect what text IDXLens sees in the PDF. This is useful for debugging or understanding why a label is not matching a dictionary entry.

### All pages

```sh
idxlens extract text quarterly-report.pdf
```

### Specific pages

```sh
idxlens extract text quarterly-report.pdf --pages "2-4"
```

### Single page

```sh
idxlens extract text quarterly-report.pdf --pages "1"
```

## Combine with other tools

### Pipe JSON to jq

```sh
idxlens extract financial report.pdf | jq '.items[] | select(.section == "assets")'
```

### Extract a single field

```sh
idxlens extract financial report.pdf | jq -r '.company'
```

### Count line items

```sh
idxlens extract financial report.pdf | jq '.items | length'
```

### Filter by confidence

```sh
idxlens extract financial report.pdf | jq '.items[] | select(.confidence >= 0.9)'
```
