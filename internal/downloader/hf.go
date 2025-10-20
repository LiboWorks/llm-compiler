package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadToModels(url, targetFile string) error {
	if err := os.MkdirAll("models", 0755); err != nil {
		return err
	}
	outPath := filepath.Join("models", targetFile)
	if _, err := os.Stat(outPath); err == nil {
		// already exists
		return nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	// Use HF token if provided
	if tk := os.Getenv("HUGGINGFACE_TOKEN"); tk != "" {
		req.Header.Set("Authorization", "Bearer "+tk)
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
