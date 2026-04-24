package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	defaultLogDir          = "logs"
	defaultLogPrefix       = "net-monitor"
	defaultMaxFileSize     = 10 * 1024 * 1024
	defaultRetention       = 5 * 24 * time.Hour
	defaultCleanupInterval = time.Hour
)

var logFilePattern = regexp.MustCompile(`^net-monitor-(\d{8})(?:\.(\d+))?\.log$`)

type RotatingWriter struct {
	mu              sync.Mutex
	dir             string
	prefix          string
	maxSize         int64
	retention       time.Duration
	cleanupInterval time.Duration
	now             func() time.Time

	file        *os.File
	currentDate string
	currentIdx  int
	size        int64
	lastCleanup time.Time
}

func Setup() (io.Closer, error) {
	writer, err := NewRotatingWriter(defaultLogDir, defaultLogPrefix, defaultMaxFileSize, defaultRetention, defaultCleanupInterval)
	if err != nil {
		return nil, err
	}

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(io.MultiWriter(os.Stdout, writer))
	return writer, nil
}

func NewRotatingWriter(dir, prefix string, maxSize int64, retention, cleanupInterval time.Duration) (*RotatingWriter, error) {
	if maxSize <= 0 {
		return nil, fmt.Errorf("maxSize must be greater than 0")
	}
	if retention <= 0 {
		return nil, fmt.Errorf("retention must be greater than 0")
	}
	if cleanupInterval <= 0 {
		return nil, fmt.Errorf("cleanupInterval must be greater than 0")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	w := &RotatingWriter{
		dir:             dir,
		prefix:          prefix,
		maxSize:         maxSize,
		retention:       retention,
		cleanupInterval: cleanupInterval,
		now:             time.Now,
	}
	if err := w.ensureFileForWrite(0, w.now()); err != nil {
		return nil, err
	}
	if err := w.cleanupLocked(w.now()); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := w.now()
	if err := w.ensureFileForWrite(int64(len(p)), now); err != nil {
		return 0, err
	}
	if now.Sub(w.lastCleanup) >= w.cleanupInterval {
		if err := w.cleanupLocked(now); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closeCurrentLocked()
}

func (w *RotatingWriter) ensureFileForWrite(incoming int64, now time.Time) error {
	date := now.Format("20060102")
	if w.file == nil || w.currentDate != date {
		if err := w.closeCurrentLocked(); err != nil {
			return err
		}
		return w.openLatestForDateLocked(date)
	}

	if w.size > 0 && w.size+incoming > w.maxSize {
		if err := w.rotateLocked(date); err != nil {
			return err
		}
	}
	return nil
}

func (w *RotatingWriter) openLatestForDateLocked(date string) error {
	idx, size, err := w.discoverWritableFileLocked(date)
	if err != nil {
		return err
	}
	path := w.filePath(date, idx)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", path, err)
	}
	w.file = file
	w.currentDate = date
	w.currentIdx = idx
	w.size = size
	return nil
}

func (w *RotatingWriter) rotateLocked(date string) error {
	if err := w.closeCurrentLocked(); err != nil {
		return err
	}
	idx := w.currentIdx + 1
	path := w.filePath(date, idx)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create rotated log file %s: %w", path, err)
	}
	w.file = file
	w.currentDate = date
	w.currentIdx = idx
	w.size = 0
	return nil
}

func (w *RotatingWriter) closeCurrentLocked() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.size = 0
	return err
}

func (w *RotatingWriter) discoverWritableFileLocked(date string) (int, int64, error) {
	matches, err := w.listLogFilesLocked(date)
	if err != nil {
		return 0, 0, err
	}
	if len(matches) == 0 {
		return 0, 0, nil
	}

	last := matches[len(matches)-1]
	if last.size >= w.maxSize {
		return last.idx + 1, 0, nil
	}
	return last.idx, last.size, nil
}

type logFileInfo struct {
	name string
	idx  int
	size int64
}

func (w *RotatingWriter) listLogFilesLocked(date string) ([]logFileInfo, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("read log dir: %w", err)
	}

	prefix := fmt.Sprintf("%s-%s", w.prefix, date)
	matches := make([]logFileInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !matchesDatePrefix(name, prefix) {
			continue
		}
		idx, ok := parseLogIndex(name, prefix)
		if !ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat log file %s: %w", name, err)
		}
		matches = append(matches, logFileInfo{name: name, idx: idx, size: info.Size()})
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].idx < matches[j].idx })
	return matches, nil
}

func matchesDatePrefix(name, prefix string) bool {
	return name == prefix+".log" || (len(name) > len(prefix)+5 && name[:len(prefix)] == prefix)
}

func parseLogIndex(name, prefix string) (int, bool) {
	base := prefix + ".log"
	if name == base {
		return 0, true
	}
	if len(name) <= len(prefix)+6 || name[:len(prefix)+1] != prefix+"." || filepath.Ext(name) != ".log" {
		return 0, false
	}
	idx, err := strconv.Atoi(name[len(prefix)+1 : len(name)-4])
	if err != nil {
		return 0, false
	}
	return idx, true
}

func (w *RotatingWriter) cleanupLocked(now time.Time) error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return fmt.Errorf("read log dir for cleanup: %w", err)
	}

	cutoff := now.Add(-w.retention)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !logFilePattern.MatchString(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat log file %s: %w", name, err)
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(w.dir, name)); err != nil {
				return fmt.Errorf("remove expired log %s: %w", name, err)
			}
		}
	}
	w.lastCleanup = now
	return nil
}

func (w *RotatingWriter) filePath(date string, idx int) string {
	if idx == 0 {
		return filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, date))
	}
	return filepath.Join(w.dir, fmt.Sprintf("%s-%s.%d.log", w.prefix, date, idx))
}
