package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadFile(fpath string, url string) error {
	dir := filepath.Dir(fpath)
	if dir != "." && dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", fpath, err)
		}
	}

	out, err := os.Create(fpath)
	if err != nil {
		log.Printf("DownloadFile: failed to create file at path %s: %v", fpath, err)
		return fmt.Errorf("failed to create file %s: %w", fpath, err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK HTTP status for %s: %s", url, resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write data to file %s: %w", fpath, err)
	}

	return nil
}
