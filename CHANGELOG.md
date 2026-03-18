# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-03-18

### Added
- PDF text extraction with pdfcpu (L0)
- Layout analysis and text line assembly (L1)
- Table detection with rule-based line detection and spatial clustering (L2)
- Financial statement mapping with PSAK/IFRS dictionaries (L3)
- Document classification (balance sheet, income statement, cash flow, etc.)
- Indonesian number format parsing
- Bilingual content routing (Indonesian/English)
- Auditor opinion parsing
- ESG/GRI index extraction
- Key-value pair extraction
- JSON and CSV output formats (L4)
- CLI commands: extract (financial, text), classify, batch, version (L5)
- Streaming API with channel-based concurrent processing
- Python bindings via CGo
- GitHub Action for CI/CD integration
- Docker image and Lambda layer
- Benchmark corpus and accuracy framework
- Fuzz testing for parser robustness
- Performance benchmarks for all layers
- Comprehensive documentation site
