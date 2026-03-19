package cli

// Shared flag name constants used across multiple commands.
const (
	flagFormat   = "format"
	flagOutput   = "output"
	flagPretty   = "pretty"
	flagYear     = "year"
	flagPeriod   = "period"
	flagFileType = "file-type"
	flagWorkers  = "workers"
	flagMode     = "mode"

	defaultFormat  = "json"
	defaultWorkers = 4

	formatXLSX = "xlsx"
	formatXBRL = "xbrl"
	formatPDF  = "pdf"

	modeFinancial    = "financial"
	modePresentation = "presentation"
)
