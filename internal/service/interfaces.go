package service

import (
	"context"

	"github.com/lugassawan/idxlens/internal/idx"
)

// ReportLister lists financial report attachments from IDX.
type ReportLister interface {
	ListReports(ctx context.Context, ticker string, year int, period string) ([]idx.Attachment, error)
}

// FileDownloader downloads file attachments to a local directory.
type FileDownloader interface {
	Download(ctx context.Context, att idx.Attachment, destDir string) (*idx.DownloadResult, error)
}

// IDXFetcher combines listing and downloading capabilities.
type IDXFetcher interface {
	ReportLister
	FileDownloader
}

// RegistryProvider loads the presentation registry.
type RegistryProvider interface {
	Registry(ctx context.Context) (map[string]idx.CompanyRegistry, error)
}
