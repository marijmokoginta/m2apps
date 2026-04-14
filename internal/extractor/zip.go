package extractor

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractZip(src string, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	cleanDest := filepath.Clean(dest)
	prefix := cleanDest + string(os.PathSeparator)

	for _, file := range reader.File {
		destPath := filepath.Join(cleanDest, file.Name)
		cleanPath := filepath.Clean(destPath)

		// Prevent Zip Slip by ensuring extracted path stays under destination.
		if cleanPath != cleanDest && !strings.HasPrefix(cleanPath, prefix) {
			return fmt.Errorf("invalid file path")
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanPath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", cleanPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", cleanPath, err)
		}

		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry %s: %w", file.Name, err)
		}

		dstFile, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("failed to create extracted file %s: %w", cleanPath, err)
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			dstFile.Close()
			srcFile.Close()
			return fmt.Errorf("failed to extract file %s: %w", cleanPath, err)
		}

		if err := dstFile.Close(); err != nil {
			srcFile.Close()
			return fmt.Errorf("failed to close extracted file %s: %w", cleanPath, err)
		}

		if err := srcFile.Close(); err != nil {
			return fmt.Errorf("failed to close zip entry %s: %w", file.Name, err)
		}
	}

	return nil
}
