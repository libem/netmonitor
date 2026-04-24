package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotatesBySize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)
	writer, err := NewRotatingWriter(dir, "net-monitor", 10, 5*24*time.Hour, time.Hour)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}
	writer.now = func() time.Time { return now }
	defer writer.Close()

	if _, err := writer.Write([]byte("12345")); err != nil {
		t.Fatalf("first Write() error = %v", err)
	}
	if _, err := writer.Write([]byte("67890")); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}
	if _, err := writer.Write([]byte("abc")); err != nil {
		t.Fatalf("third Write() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "net-monitor-20260424.log")); err != nil {
		t.Fatalf("stat base log: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "net-monitor-20260424.1.log")); err != nil {
		t.Fatalf("stat rotated log: %v", err)
	}
}

func TestRotatesByDate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 4, 24, 23, 59, 0, 0, time.UTC)
	writer, err := NewRotatingWriter(dir, "net-monitor", 1024, 5*24*time.Hour, time.Hour)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}
	writer.now = func() time.Time { return now }
	defer writer.Close()

	if _, err := writer.Write([]byte("day1")); err != nil {
		t.Fatalf("day1 Write() error = %v", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := writer.Write([]byte("day2")); err != nil {
		t.Fatalf("day2 Write() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "net-monitor-20260424.log")); err != nil {
		t.Fatalf("stat day1 log: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "net-monitor-20260425.log")); err != nil {
		t.Fatalf("stat day2 log: %v", err)
	}
}

func TestCleanupRemovesExpiredLogs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	writer, err := NewRotatingWriter(dir, "net-monitor", 1024, 5*24*time.Hour, time.Hour)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}
	writer.now = func() time.Time { return now }
	defer writer.Close()

	oldPath := filepath.Join(dir, "net-monitor-20260410.log")
	newPath := filepath.Join(dir, "net-monitor-20260424.log")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile oldPath error = %v", err)
	}
	if err := os.Chtimes(oldPath, now.Add(-6*24*time.Hour), now.Add(-6*24*time.Hour)); err != nil {
		t.Fatalf("Chtimes oldPath error = %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile newPath error = %v", err)
	}
	if err := os.Chtimes(newPath, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("Chtimes newPath error = %v", err)
	}

	writer.mu.Lock()
	err = writer.cleanupLocked(now)
	writer.mu.Unlock()
	if err != nil {
		t.Fatalf("cleanupLocked() error = %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("oldPath still exists, err = %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("newPath missing, err = %v", err)
	}
}
