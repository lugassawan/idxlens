package idx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Attachment represents a single file attachment from an IDX financial report.
type Attachment struct {
	FileName     string `json:"namaFile"`
	FilePath     string `json:"pathFile"`
	FileType     string `json:"tipeFile"`
	FileSize     int64  `json:"fileSize"`
	EmitenCode   string `json:"kodeEmiten"`
	ReportPeriod string `json:"periLaporan"`
	ReportYear   string `json:"tahunLaporan"`
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
	url := fmt.Sprintf(
		"%s%s?periode=%s&year=%d&kodeEmiten=%s&reportType=rdf&indexFrom=0&pageSize=1000",
		c.baseURL, reportEndpoint, period, year, ticker,
	)

	req, err := c.newRequest(http.MethodGet, url)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}

	req = req.WithContext(ctx)

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
