package idx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Attachment represents a single file attachment from an IDX financial report.
type Attachment struct {
	FileName     string `json:"File_Name"`
	FilePath     string `json:"File_Path"`
	FileType     string `json:"File_Type"`
	FileSize     int64  `json:"File_Size"`
	EmitenCode   string `json:"Emiten_Code"`
	ReportPeriod string `json:"Report_Period"`
	ReportYear   string `json:"Report_Year"`
}

type reportResponse struct {
	Results []reportResult `json:"Results"`
}

type reportResult struct {
	Attachments []Attachment `json:"Attachments"`
}

const reportEndpoint = "/primary/ListedCompany/GetFinancialReport"

// ListReports fetches financial report attachments for the given ticker, year, and period.
func (c *Client) ListReports(ctx context.Context, ticker string, year int, period string) ([]Attachment, error) {
	endpoint := c.baseURL + reportEndpoint
	params := url.Values{
		"periode":    {period},
		"year":       {strconv.Itoa(year)},
		"kodeEmiten": {ticker},
		"reportType": {"rdf"},
		"indexFrom":  {"0"},
		"pageSize":   {"1000"},
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint+"?"+params.Encode())
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}

	//nolint:gosec // URL built from trusted baseURL set at client construction
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list reports request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list reports: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("list reports: read body: %w", err)
	}

	var response reportResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("list reports: unmarshal response: %w", err)
	}

	var attachments []Attachment
	for _, result := range response.Results {
		attachments = append(attachments, result.Attachments...)
	}

	return attachments, nil
}
