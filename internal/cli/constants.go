package cli

// Shared flag name constants used across multiple commands.
const (
	flagFormat   = "format"
	flagYear     = "year"
	flagPeriod   = "period"
	flagFileType = "file-type"
	flagWorkers  = "workers"

	defaultFormat  = "json"
	defaultWorkers = 4

	formatXLSX = "xlsx"
	formatXBRL = "xbrl"
	formatPDF  = "pdf"
)
