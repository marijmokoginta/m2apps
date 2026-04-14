package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Downloader struct {
	httpClient         *http.Client
	redirectHTTPClient *http.Client
	token              string
}

func New(token string) *Downloader {
	return &Downloader{
		httpClient: &http.Client{
			Timeout: 0,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		redirectHTTPClient: &http.Client{Timeout: 0},
		token:              strings.TrimSpace(token),
	}
}

func (d *Downloader) Download(url string, dest string, onProgress func(read, total int64)) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")
	if d.token != "" {
		req.Header.Set("Authorization", "Bearer "+d.token)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	var downloadBody io.ReadCloser
	var total int64

	switch resp.StatusCode {
	case http.StatusOK:
		downloadBody = resp.Body
		total = resp.ContentLength
	case http.StatusFound:
		redirectURL := strings.TrimSpace(resp.Header.Get("Location"))
		if redirectURL == "" {
			return fmt.Errorf("download failed: missing redirect location")
		}

		req2, err := http.NewRequest(http.MethodGet, redirectURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create redirected download request: %w", err)
		}

		resp2, err := d.redirectHTTPClient.Do(req2)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: status %d", resp2.StatusCode)
		}

		downloadBody = resp2.Body
		total = resp2.ContentLength
	case http.StatusUnauthorized:
		return fmt.Errorf("download unauthorized (401): invalid token")
	case http.StatusForbidden:
		return fmt.Errorf("download forbidden (403): access denied or rate limit exceeded")
	case http.StatusNotFound:
		return fmt.Errorf("download source not found (404)")
	default:
		return fmt.Errorf("download request failed: %s", resp.Status)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dest, err)
	}
	defer file.Close()

	progress := &ProgressReader{
		Reader:     downloadBody,
		Total:      total,
		OnProgress: onProgress,
	}

	if _, err := io.Copy(file, progress); err != nil {
		return fmt.Errorf("download interrupted: %w", err)
	}

	if onProgress != nil {
		onProgress(progress.ReadBytes, progress.Total)
	}

	return nil
}
