# Architecture

IDXLens processes PDF files through a six-layer pipeline. Each layer has a single responsibility and communicates with adjacent layers through interfaces.

## Layer overview

```
┌─────────────────────────────────────────────┐
│  Framework: cmd/idxlens/main.go             │  Calls cli.Execute()
├─────────────────────────────────────────────┤
│  L5  CLI             internal/cli/          │  Cobra commands, flag parsing
├─────────────────────────────────────────────┤
│  L4  Output          internal/output/       │  JSON/CSV formatting
├─────────────────────────────────────────────┤
│  L3  Domain          internal/domain/       │  Classification, mapping, dictionaries
├─────────────────────────────────────────────┤
│  L2  Table           internal/table/        │  Table detection from layout
├─────────────────────────────────────────────┤
│  L1  Layout          internal/layout/       │  Text grouping, line detection
├─────────────────────────────────────────────┤
│  L0  PDF             internal/pdf/          │  Raw PDF parsing (pdfcpu)
└─────────────────────────────────────────────┘
```

## Dependency rule

Dependencies flow strictly downward. Each layer may only import from layers below it:

```
cli -> output -> domain -> table -> layout -> pdf
```

No layer imports from a layer above it. This keeps the architecture testable and maintainable -- lower layers can be tested in isolation without any knowledge of how they are consumed.

## Layer details

### L0: PDF Parser (`internal/pdf/`)

Wraps the [pdfcpu](https://github.com/pdfcpu/pdfcpu) library to extract raw text content from PDF pages. Provides a `Reader` interface for opening PDFs and iterating over pages.

**Interface:**

```go
type Reader interface {
    Open(r io.ReadSeeker) error
    Close() error
    PageCount() int
    ReadPage(pageNum int) (Page, error)
}
```

**Responsibilities:**
- Open and validate PDF files
- Extract raw text elements with position coordinates
- Provide page-level access to content

### L1: Text and Layout Engine (`internal/layout/`)

Transforms raw PDF text elements into structured layout pages with text lines, blocks, and spatial relationships.

**Interface:**

```go
type Analyzer interface {
    Analyze(page pdf.Page) (LayoutPage, error)
}
```

**Responsibilities:**
- Group text elements into lines based on vertical proximity
- Sort elements within lines by horizontal position
- Build a spatial model of the page content

### L2: Table Detector (`internal/table/`)

Identifies tabular structures in layout pages by detecting aligned columns, row boundaries, and header regions.

**Responsibilities:**
- Detect table boundaries within layout pages
- Extract headers and data rows
- Handle multi-column layouts common in financial reports

### L3: IDX Domain Engine (`internal/domain/`)

Contains all IDX-specific business logic: document classification, financial statement mapping, ESG extraction, noise filtering, number parsing, and dictionary-based label matching.

**Key components:**

| Component          | Purpose                                                  |
|-------------------|----------------------------------------------------------|
| Classifier         | Heuristic-based report type detection (9 document types) |
| Mapper             | Maps table rows to financial line items using dictionaries|
| ESG Extractor      | Extracts GRI content index disclosures from sustainability reports |
| Noise Filter       | Removes garbled text, governance tables, page references, and non-financial content |
| Dictionary         | Bilingual label matching (Indonesian/English), including banking-specific items |
| Number parser      | Indonesian number format parsing (dot thousands, comma decimal) |
| Bilingual detector | Tab-separated column detection for side-by-side ID/EN layouts |

**Responsibilities:**
- Classify documents by type (balance sheet, income statement, sustainability report, etc.)
- Map raw table data to structured financial statements
- Extract GRI disclosures from ESG content index tables
- Filter noise: garbled PDF text, spaced-out characters, governance/compliance tables, page references
- Match labels to dictionary items with confidence scores
- Parse Indonesian-format numbers and abbreviated periods (Dec-25, FY-25, 3Q-25)
- Detect "Rp tn" unit formats in corporate presentations

### L4: Output Formatter (`internal/output/`)

Formats financial statements into output formats (JSON, CSV).

**Interface:**

```go
type Formatter interface {
    Format(w io.Writer, stmt *domain.FinancialStatement) error
}
```

**Responsibilities:**
- Serialize financial statements to JSON (with optional pretty-printing)
- Serialize financial statements to CSV with sorted period columns

### L5: CLI (`internal/cli/`)

Cobra-based command definitions. Wires the pipeline together, handles flag parsing, and manages I/O.

**Responsibilities:**
- Define commands (`classify`, `extract financial`, `extract text`, `extract esg`, `batch`, `version`)
- Parse and validate flags
- Orchestrate the pipeline: open PDF, analyze, classify, detect tables, map, format
- Handle output destination (stdout or file)

## Interface boundaries

Each layer defines its interfaces at its own boundary. Implementations live in the layer below. This follows the Dependency Inversion Principle -- upper layers depend on abstractions, not concrete implementations.

```
L5 cli/        uses    output.Formatter (interface defined in L4)
L4 output/     uses    domain.FinancialStatement (types defined in L3)
L3 domain/     uses    table.Table (types defined in L2)
L2 table/      uses    layout.LayoutPage (types defined in L1)
L1 layout/     uses    pdf.Page (types defined in L0)
```

## Data flow

A typical `extract financial` command flows through all layers:

```
PDF file
  │
  ▼
L0 pdf.Reader.ReadPage()        →  pdf.Page (raw text + positions)
  │
  ▼
L1 layout.Analyzer.Analyze()    →  layout.LayoutPage (structured lines)
  │
  ▼
L3 domain.Classifier.Classify() →  domain.Classification (report type)
  │
  ▼
L2 table.Detector.Detect()      →  []table.Table (headers + rows)
  │
  ▼
L3 domain.Mapper.Map()          →  domain.FinancialStatement (structured data)
  │
  ▼
L4 output.Formatter.Format()    →  JSON or CSV output
```

## Design principles

- **Pure Go, no CGO**: Single static binary with no external dependencies at runtime
- **No network calls**: All processing is local. PDF in, data out.
- **Internal only**: All packages live under `internal/` -- no public API surface
- **Interface-driven**: Cross-layer boundaries use interfaces for testability and decoupling
