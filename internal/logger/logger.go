package logger

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/brainproxy/brainproxy/internal/models"
)

// Logger writes completed request events to JSON files on disk.
type Logger struct {
	dir      string
	maxFiles int
	mu       sync.Mutex // serializes cleanup to avoid race conditions
}

// New creates a Logger that writes JSON files to dir.
// It creates the directory if it doesn't exist.
// If dir is empty, logging is disabled (Save becomes a no-op).
// maxFiles limits the number of log files kept on disk (0 = unlimited).
func New(dir string, maxFiles int) (*Logger, error) {
	if dir == "" {
		return &Logger{}, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Logger{dir: dir, maxFiles: maxFiles}, nil
}

// Save writes the request event to a JSON file asynchronously.
// File name format: {timestamp}_{short-id}.json
// Example: 20260617-143052_a1b2c3d4.json
func (l *Logger) Save(event *models.RequestEvent) {
	if l.dir == "" || event == nil {
		return
	}

	go func() {
		ts := event.Timestamp.Format("20060102-150405")
		shortID := event.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		filename := ts + "_" + shortID + ".json"
		path := filepath.Join(l.dir, filename)

		data, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			log.Printf("logger: marshal error for %s: %v", filename, err)
			return
		}

		if err := os.WriteFile(path, data, 0o644); err != nil {
			log.Printf("logger: write error for %s: %v", filename, err)
			return
		}

		log.Printf("logger: saved %s", path)

		// Evict old files if over limit
		if l.maxFiles > 0 {
			l.cleanup()
		}
	}()
}

// cleanup removes the oldest log files when the count exceeds maxFiles.
func (l *Logger) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return
	}

	// Filter only .json files
	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			files = append(files, e)
		}
	}

	if len(files) <= l.maxFiles {
		return
	}

	// Sort by name (timestamp prefix ensures chronological order)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Delete oldest files
	toDelete := len(files) - l.maxFiles
	for i := 0; i < toDelete; i++ {
		path := filepath.Join(l.dir, files[i].Name())
		if err := os.Remove(path); err != nil {
			log.Printf("logger: failed to remove old file %s: %v", files[i].Name(), err)
		} else {
			log.Printf("logger: evicted old file %s", files[i].Name())
		}
	}
}
