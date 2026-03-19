package xbrl

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Statement holds extracted XBRL financial data.
type Statement struct {
	Ticker string `json:"ticker"`
	Year   int    `json:"year"`
	Period string `json:"period"`
	Facts  []Fact `json:"facts"`
}

// Fact represents a single XBRL fact (a financial data point).
type Fact struct {
	Concept  string  `json:"concept"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
	Period   string  `json:"period"`
	Decimals string  `json:"decimals"`
}

// filenamePattern matches IDX inline XBRL zip filenames.
var filenamePattern = regexp.MustCompile(
	`(?i)FinancialStatement-(\d{4})-([^-]+)-([A-Z]{4})`,
)

// xbrlExtPriority defines the preference order for XBRL file selection.
var xbrlExtPriority = []string{".xbrl", ".xml", ".htm", ".html"}

// ParseZip reads a zip file containing inline XBRL and extracts financial facts.
func ParseZip(zipPath string) (*Statement, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	stmt := &Statement{}
	parseMeta(stmt, filepath.Base(zipPath))

	xbrlFile, err := findXBRLFile(r.File)
	if err != nil {
		return nil, err
	}

	rc, err := xbrlFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open xbrl entry: %w", err)
	}
	defer rc.Close()

	facts, err := parseFacts(rc)
	if err != nil {
		return nil, fmt.Errorf("parse facts: %w", err)
	}

	stmt.Facts = facts

	return stmt, nil
}

func parseMeta(stmt *Statement, filename string) {
	m := filenamePattern.FindStringSubmatch(filename)
	if len(m) != 4 {
		return
	}

	stmt.Year, _ = strconv.Atoi(m[1])
	stmt.Period = m[2]
	stmt.Ticker = m[3]
}

func findXBRLFile(files []*zip.File) (*zip.File, error) {
	for _, wantExt := range xbrlExtPriority {
		for _, f := range files {
			if strings.EqualFold(filepath.Ext(f.Name), wantExt) {
				return f, nil
			}
		}
	}

	return nil, errors.New("no XBRL file found in zip")
}

func parseFacts(r io.Reader) ([]Fact, error) {
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var facts []Fact

	for {
		tok, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("decode token: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		fact, ok := parseFactElement(decoder, se)
		if ok {
			facts = append(facts, fact)
		}
	}

	return facts, nil
}

func parseFactElement(decoder *xml.Decoder, se xml.StartElement) (Fact, bool) {
	local := se.Name.Local

	if !isFactElement(se.Name.Space, local) {
		return Fact{}, false
	}

	var fact Fact

	for _, attr := range se.Attr {
		switch attr.Name.Local {
		case "name":
			fact.Concept = attr.Value
		case "unitRef":
			fact.Unit = attr.Value
		case "contextRef":
			fact.Period = attr.Value
		case "decimals":
			fact.Decimals = attr.Value
		}
	}

	if fact.Concept == "" {
		fact.Concept = se.Name.Local
	}

	var content string

	_ = decoder.DecodeElement(&content, &se)

	cleaned := strings.ReplaceAll(strings.TrimSpace(content), ",", "")
	if v, err := strconv.ParseFloat(cleaned, 64); err == nil {
		fact.Value = v
	}

	return fact, fact.Concept != ""
}

func isFactElement(space, local string) bool {
	if strings.HasSuffix(space, "/inlineXBRL") ||
		strings.Contains(space, "inline") {
		switch local {
		case "nonFraction", "nonNumeric":
			return true
		}
	}

	if strings.Contains(space, "ifrs") {
		return true
	}

	return false
}
