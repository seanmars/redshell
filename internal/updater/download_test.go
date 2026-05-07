package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestDownloadHappyPath(t *testing.T) {
	body := []byte("hello redshell update payload")
	wantHash := sha256.Sum256(body)
	wantHex := hex.EncodeToString(wantHash[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "redshell.exe")
	got, err := Download(context.Background(), srv.Client(), srv.URL, dest, int64(len(body)), nil)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got != wantHex {
		t.Fatalf("hash mismatch: got %q want %q", got, wantHex)
	}
	gotBytes, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile dest: %v", err)
	}
	if string(gotBytes) != string(body) {
		t.Fatalf("file body mismatch")
	}
	if _, err := os.Stat(dest + ".partial"); !os.IsNotExist(err) {
		t.Fatalf(".partial should be cleaned up: %v", err)
	}
}

func TestDownloadEmitsProgressCallback(t *testing.T) {
	chunk := make([]byte, 1024)
	body := make([]byte, 0, len(chunk)*8)
	for i := 0; i < 8; i++ {
		body = append(body, chunk...)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "8192")
		for i := 0; i < 8; i++ {
			w.Write(chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer srv.Close()

	var calls atomic.Int32
	var lastTotal atomic.Int64
	dest := filepath.Join(t.TempDir(), "redshell.exe")
	_, err := Download(context.Background(), srv.Client(), srv.URL, dest, 0, func(d, total int64) {
		calls.Add(1)
		lastTotal.Store(total)
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if calls.Load() < 1 {
		t.Fatal("expected at least the final progress callback")
	}
	if lastTotal.Load() == 0 {
		t.Fatal("expected total to be populated from Content-Length")
	}
}

func TestDownloadRejectsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "redshell.exe")
	if _, err := Download(context.Background(), srv.Client(), srv.URL, dest, 0, nil); err == nil {
		t.Fatal("expected non-2xx to error")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("dest should not exist on failure")
	}
	if _, err := os.Stat(dest + ".partial"); !os.IsNotExist(err) {
		t.Fatal(".partial should be cleaned up on failure")
	}
}

func TestDownloadCleansPartialOnContextCancel(t *testing.T) {
	body := make([]byte, 1024*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.Write(body)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dest := filepath.Join(t.TempDir(), "redshell.exe")
	if _, err := Download(ctx, srv.Client(), srv.URL, dest, 0, nil); err == nil {
		t.Fatal("expected canceled context to error")
	}
	if _, err := os.Stat(dest + ".partial"); !os.IsNotExist(err) {
		t.Fatal(".partial should not survive a failed download")
	}
}
