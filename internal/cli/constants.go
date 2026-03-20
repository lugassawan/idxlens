package cli

// Shared flag name constants used across multiple commands.
const (
	flagFormat   = "format"
	flagOutput   = "output"
	flagPretty   = "pretty"
	flagYear     = "year"
	flagPeriod   = "period"
	flagFileType = "file-type"
	flagVerbose  = "verbose"
	flagWorkers  = "workers"
	flagMode     = "mode"
	flagDryRun   = "dry-run"

	defaultFormat  = "json"
	defaultWorkers = 4

	descYearRequired = "Report year (required)"
	descPeriod       = "Report period (e.g. Q1, Q2, Q3, FY)"

	formatXLSX = "xlsx"
	formatXBRL = "xbrl"
	formatPDF  = "pdf"

	modeFinancial    = "financial"
	modePresentation = "presentation"

	envNoColor = "NO_COLOR"
)
