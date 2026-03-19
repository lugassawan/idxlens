# Basic Extraction

This guide walks through common extraction workflows with IDXLens.

## Extract from XLSX

XLSX is the preferred format for financial data extraction. It produces the most accurate results.

```sh
idxlens extract report.xlsx
```

Pretty-print the output:

```sh
idxlens extract report.xlsx --pretty
```

Save to a file:

```sh
idxlens extract report.xlsx --output result.json
```

## Extract from XBRL ZIP

XBRL archives contain taxonomy-based financial data:

```sh
idxlens extract report.zip
```

```sh
idxlens extract report.zip --pretty
```

## Extract presentation from PDF

Corporate presentations can be extracted as key-value pairs using the `--mode presentation` flag:

```sh
idxlens extract presentation.pdf --mode presentation
```

```sh
idxlens extract presentation.pdf --mode presentation --pretty
```

## Full pipeline with analyze

The `analyze` command fetches reports (if not cached) and extracts from the best available format:

```sh
# Single ticker
idxlens analyze BBCA -y 2024 -p Q3 --pretty

# Multiple tickers
idxlens analyze BBCA,BMRI,BBRI -y 2024 -p Q3

# Save output
idxlens analyze BBCA -y 2024 --output bbca.json
```

The format priority is XLSX > XBRL > PDF. If XLSX is available, it will be used first.

## Pipe to jq

IDXLens outputs JSON by default, making it easy to pipe to `jq` for filtering and transformation.

### Filter items by section

```sh
idxlens extract report.xlsx | jq '.items[] | select(.section == "assets")'
```

### Extract a single field

```sh
idxlens extract report.xlsx | jq -r '.company'
```

### Count line items

```sh
idxlens extract report.xlsx | jq '.items | length'
```

### Get values for a specific period

```sh
idxlens analyze BBCA -y 2024 -p Q3 | jq '.items[] | {label: .label, value: .values["2024-09-30"]}'
```
