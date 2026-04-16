package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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
	var existingSize int64
	if info, err := os.Stat(dest); err == nil {
		existingSize = info.Size()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect destination file %s: %w", dest, err)
	}

	resp, err := d.startDownload(url, existingSize)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		total := contentRangeTotal(resp.Header.Get("Content-Range"))
		if total > 0 && existingSize == total {
			if onProgress != nil {
				onProgress(total, total)
			}
			return nil
		}
		existingSize = 0
		resp.Body.Close()
		resp, err = d.startDownload(url, 0)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	writeOffset := int64(0)
	total := int64(-1)
	appendMode := false

	switch resp.StatusCode {
	case http.StatusPartialContent:
		appendMode = true
		writeOffset = existingSize
		total = existingSize + maxInt64(resp.ContentLength, 0)
		if start := contentRangeStart(resp.Header.Get("Content-Range")); start >= 0 && start != existingSize {
			appendMode = false
			writeOffset = 0
		}
	case http.StatusOK:
		appendMode = false
		writeOffset = 0
		total = resp.ContentLength
	default:
		return mapDownloadStatusError(resp)
	}

	file, err := openDestinationFile(dest, appendMode)
	if err != nil {
		return err
	}
	defer file.Close()

	if writeOffset > 0 {
		if _, err := file.Seek(writeOffset, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek destination file %s: %w", dest, err)
		}
	}

	progress := &ProgressReader{
		Reader:     resp.Body,
		Total:      total,
		ReadBytes:  writeOffset,
		OnProgress: onProgress,
	}
	if onProgress != nil {
		onProgress(progress.ReadBytes, progress.Total)
	}

	if _, err := io.Copy(file, progress); err != nil {
		return fmt.Errorf("download interrupted: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to flush destination file %s: %w", dest, err)
	}

	actualSize, err := fileSize(dest)
	if err != nil {
		return err
	}
	if progress.Total > 0 && actualSize != progress.Total {
		return fmt.Errorf("download size mismatch: got %d bytes, expected %d bytes", actualSize, progress.Total)
	}

	if onProgress != nil {
		onProgress(actualSize, progress.Total)
	}
	return nil
}

func (d *Downloader) startDownload(url string, offset int64) (*http.Response, error) {
	resp, err := d.executeDownloadRequest(url, offset, true)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusFound:
		redirectURL := strings.TrimSpace(resp.Header.Get("Location"))
		if redirectURL == "" {
			resp.Body.Close()
			return nil, fmt.Errorf("download failed: missing redirect location")
		}
		resp.Body.Close()
		return d.executeDownloadRequest(redirectURL, offset, false)
	default:
		return resp, nil
	}
}

func (d *Downloader) executeDownloadRequest(url string, offset int64, withToken bool) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	if withToken && d.token != "" {
		req.Header.Set("Authorization", "Bearer "+d.token)
	}

	client := d.redirectHTTPClient
	if withToken {
		client = d.httpClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func openDestinationFile(dest string, appendMode bool) (*os.File, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(dest, flags, 0o644)
	if err != nil {
		if appendMode {
			return nil, fmt.Errorf("failed to resume destination file %s: %w", dest, err)
		}
		return nil, fmt.Errorf("failed to create destination file %s: %w", dest, err)
	}
	return file, nil
}

func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect destination file %s: %w", path, err)
	}
	return info.Size(), nil
}

func mapDownloadStatusError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("download unauthorized (401): invalid token")
	case http.StatusForbidden:
		return fmt.Errorf("download forbidden (403): access denied or rate limit exceeded")
	case http.StatusNotFound:
		return fmt.Errorf("download source not found (404)")
	default:
		return fmt.Errorf("download request failed: %s", resp.Status)
	}
}

func contentRangeStart(header string) int64 {
	value := strings.TrimSpace(header)
	if !strings.HasPrefix(strings.ToLower(value), "bytes ") {
		return -1
	}
	value = strings.TrimSpace(value[len("bytes "):])
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return -1
	}
	rangePart := strings.TrimSpace(parts[0])
	startEnd := strings.SplitN(rangePart, "-", 2)
	if len(startEnd) != 2 {
		return -1
	}
	start, err := strconv.ParseInt(strings.TrimSpace(startEnd[0]), 10, 64)
	if err != nil {
		return -1
	}
	return start
}

func contentRangeTotal(header string) int64 {
	value := strings.TrimSpace(header)
	if !strings.HasPrefix(strings.ToLower(value), "bytes ") {
		return -1
	}
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return -1
	}
	totalPart := strings.TrimSpace(parts[1])
	if totalPart == "" || totalPart == "*" {
		return -1
	}
	total, err := strconv.ParseInt(totalPart, 10, 64)
	if err != nil {
		return -1
	}
	return total
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
