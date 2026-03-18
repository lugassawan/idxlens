# Dictionaries

IDXLens uses JSON dictionary files to map financial line item labels found in PDFs to standardized keys. Dictionaries support bilingual labels (Indonesian and English) to handle both language variants common in IDX reports.

## Location

Dictionary files are embedded into the binary at build time from:

```
internal/domain/dictionaries/
├── balance_sheet.json
├── cash_flow.json
├── equity_changes.json
└── income_statement.json
```

Each file corresponds to a report type.

## JSON format

A dictionary file has this structure:

```json
{
  "type": "balance-sheet",
  "version": 2,
  "items": [
    {
      "key": "cash_and_equivalents",
      "labels": {
        "id": ["Kas dan Setara Kas", "Kas dan Bank"],
        "en": ["Cash and Cash Equivalents", "Cash and Banks"]
      },
      "section": "assets",
      "level": 2
    }
  ]
}
```

### Top-level fields

| Field     | Type   | Description                                    |
|-----------|--------|------------------------------------------------|
| `type`    | string | Report type identifier (e.g. `"balance-sheet"`)|
| `version` | int    | Schema version for the dictionary              |
| `items`   | array  | List of line item definitions                  |

### Item fields

| Field     | Type              | Description                                           |
|-----------|-------------------|-------------------------------------------------------|
| `key`     | string            | Unique identifier for the line item (snake_case)      |
| `labels`  | map[string][]string | Language-keyed label variants for matching            |
| `section` | string            | Logical section within the statement                  |
| `level`   | int               | Nesting depth (0 = top-level, 1 = section, 2 = item) |

### Label matching

The `labels` field maps language codes to arrays of string variants:

- `"id"` -- Indonesian labels
- `"en"` -- English labels

Multiple variants per language handle differences in casing, abbreviation, or phrasing across different company reports. The matcher compares extracted text against all variants and returns the best match with a confidence score:

| Confidence | Match type                            |
|-----------|---------------------------------------|
| 1.0        | Exact string match                    |
| 0.9        | Case-insensitive match                |
| 0.7        | Label is a substring of the text      |

## Adding custom items

To add a new line item to an existing dictionary:

1. Open the dictionary file in `internal/domain/dictionaries/`.
2. Add a new entry to the `items` array:

```json
{
  "key": "short_term_investments",
  "labels": {
    "id": ["Investasi Jangka Pendek"],
    "en": ["Short-term Investments", "Short Term Investments"]
  },
  "section": "assets",
  "level": 2
}
```

3. Choose a descriptive `key` in snake_case. This becomes the field name in output.
4. Include as many label variants as you find across different company reports.
5. Set `section` to match the statement section (e.g. `"assets"`, `"liabilities"`, `"equity"`).
6. Set `level` to reflect the nesting depth in the statement hierarchy.
7. Rebuild the binary (`make build`) to embed the updated dictionary.

## Report types and sections

### Balance Sheet (`balance_sheet.json`)

| Section       | Description                         |
|--------------|-------------------------------------|
| `assets`      | Current and non-current assets     |
| `liabilities` | Current and non-current liabilities|
| `equity`      | Shareholders' equity               |

### Income Statement (`income_statement.json`)

| Section       | Description                        |
|--------------|------------------------------------|
| `revenue`     | Revenue and sales                  |
| `expenses`    | Cost of goods sold, operating expenses |
| `profit`      | Gross, operating, and net profit   |

### Cash Flow (`cash_flow.json`)

| Section        | Description                       |
|---------------|-----------------------------------|
| `operating`    | Cash flows from operating activities |
| `investing`    | Cash flows from investing activities |
| `financing`    | Cash flows from financing activities |

### Equity Changes (`equity_changes.json`)

| Section        | Description                       |
|---------------|-----------------------------------|
| `equity`       | Changes in equity components      |

## Guidelines for label variants

- Include the exact text as it appears in PDFs (preserving casing).
- Add both formal and abbreviated forms (e.g. "Jumlah Aset Lancar" and "Total Aset Lancar").
- Add all-caps variants if commonly seen (e.g. "TOTAL ASSETS").
- Test new labels against sample PDFs using `idxlens extract text` to see the raw text first.
