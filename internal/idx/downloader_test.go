package idx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownload(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		content := "hello world"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(content))
		}))
		defer srv.Close()

		destDir := t.TempDir()
		c := New(WithBaseURL(srv.URL))
		att := Attachment{FileName: "test.pdf", FilePath: "/files/test.pdf"}

		result, err := c.Download(context.Background(), att, destDir)
		if err != nil {
			t.Fatalf("Download() error: %v", err)
		}

		if result.LocalPath != filepath.Join(destDir, "test.pdf") {
			t.Errorf("LocalPath = %q, want %q", result.LocalPath, filepath.Join(destDir, "test.pdf"))
		}

		got, err := os.ReadFile(result.LocalPath)
		if err != nil {
			t.Fatalf("read downloaded file: %v", err)
		}

		if string(got) != content {
			t.Errorf("file content = %q, want %q", string(got), content)
		}
	})

	t.Run("server error returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		destDir := t.TempDir()
		c := New(WithBaseURL(srv.URL))
		att := Attachment{FileName: "missing.pdf", FilePath: "/files/missing.pdf"}

		_, err := c.Download(context.Background(), att, destDir)
		if err == nil {
			t.Fatal("Download() expected error for 404")
		}
	})

	t.Run("atomic write cleans up tmp file", func(t *testing.T) {
		content := "atomic content"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(content))
		}))
		defer srv.Close()

		destDir := t.TempDir()
		c := New(WithBaseURL(srv.URL))
		att := Attachment{FileName: "atomic.pdf", FilePath: "/files/atomic.pdf"}

		_, err := c.Download(context.Background(), att, destDir)
		if err != nil {
			t.Fatalf("Download() error: %v", err)
		}

		tmpPath := filepath.Join(destDir, "atomic.pdf.tmp")
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			t.Errorf("tmp file should not exist after successful download")
		}

		finalPath := filepath.Join(destDir, "atomic.pdf")
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			t.Error("final file should exist after successful download")
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("new content"))
		}))
		defer srv.Close()

		destDir := t.TempDir()
		existing := filepath.Join(destDir, "overwrite.pdf")
		if err := os.WriteFile(existing, []byte("old content"), 0o644); err != nil {
			t.Fatalf("create existing file: %v", err)
		}

		c := New(WithBaseURL(srv.URL))
		att := Attachment{FileName: "overwrite.pdf", FilePath: "/files/overwrite.pdf"}

		_, err := c.Download(context.Background(), att, destDir)
		if err != nil {
			t.Fatalf("Download() error: %v", err)
		}

		got, err := os.ReadFile(existing)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		if string(got) != "new content" {
			t.Errorf("file content = %q, want %q", string(got), "new content")
		}
	})
}

func TestDownloadAll(t *testing.T) {
	t.Run("multiple files with bounded concurrency", func(t *testing.T) {
		files := map[string]string{
			"/files/a.pdf": "content a",
			"/files/b.pdf": "content b",
			"/files/c.pdf": "content c",
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			content, ok := files[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write([]byte(content))
		}))
		defer srv.Close()

		destDir := t.TempDir()
		c := New(WithBaseURL(srv.URL))

		atts := []Attachment{
			{FileName: "a.pdf", FilePath: "/files/a.pdf"},
			{FileName: "b.pdf", FilePath: "/files/b.pdf"},
			{FileName: "c.pdf", FilePath: "/files/c.pdf"},
		}

		results := c.DownloadAll(context.Background(), atts, destDir, 2)
		if len(results) != 3 {
			t.Fatalf("got %d results, want 3", len(results))
		}

		for i, r := range results {
			if r.Err != nil {
				t.Errorf("result[%d] error: %v", i, r.Err)
				continue
			}

			got, err := os.ReadFile(r.LocalPath)
			if err != nil {
				t.Errorf("result[%d] read file: %v", i, err)
				continue
			}

			want := files[atts[i].FilePath]
			if string(got) != want {
				t.Errorf("result[%d] content = %q, want %q", i, string(got), want)
			}
		}
	})

	t.Run("partial failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/files/good.pdf" {
				w.Write([]byte("good"))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		destDir := t.TempDir()
		c := New(WithBaseURL(srv.URL))

		atts := []Attachment{
			{FileName: "good.pdf", FilePath: "/files/good.pdf"},
			{FileName: "bad.pdf", FilePath: "/files/bad.pdf"},
		}

		results := c.DownloadAll(context.Background(), atts, destDir, 2)
		if results[0].Err != nil {
			t.Errorf("result[0] unexpected error: %v", results[0].Err)
		}

		if results[1].Err == nil {
			t.Error("result[1] expected error for missing file")
		}
	})
}
