package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ProgressFunc func(bytesDownloaded, totalBytes int64)

const progressEmitInterval = 250 * time.Millisecond

func Download(ctx context.Context, httpClient *http.Client, urlStr, destPath string, expectedSize int64, progress ProgressFunc) (sha256Hex string, err error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("download: build request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("download: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	partialPath := destPath + ".partial"
	f, err := os.Create(partialPath)
	if err != nil {
		return "", fmt.Errorf("download: create partial: %w", err)
	}
	cleanupPartial := true
	defer func() {
		if cleanupPartial {
			_ = f.Close()
			_ = os.Remove(partialPath)
		}
	}()

	total := expectedSize
	if total <= 0 {
		total = resp.ContentLength
	}

	hasher := sha256.New()
	pw := &progressWriter{
		total:    total,
		progress: progress,
	}

	if _, err = io.Copy(io.MultiWriter(f, hasher, pw), resp.Body); err != nil {
		return "", fmt.Errorf("download: copy: %w", err)
	}
	if err = f.Close(); err != nil {
		return "", fmt.Errorf("download: close partial: %w", err)
	}
	if err = os.Rename(partialPath, destPath); err != nil {
		return "", fmt.Errorf("download: rename partial: %w", err)
	}
	cleanupPartial = false

	if progress != nil {
		progress(pw.written, total)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

type progressWriter struct {
	written  int64
	total    int64
	progress ProgressFunc
	lastEmit time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.written += int64(n)
	if p.progress != nil && time.Since(p.lastEmit) >= progressEmitInterval {
		p.progress(p.written, p.total)
		p.lastEmit = time.Now()
	}
	return n, nil
}
