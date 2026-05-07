// Package history persists executed-request entries as append-only JSONL.
package history

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Entry is a single recorded request execution.
type Entry struct {
	Timestamp       time.Time         `json:"timestamp"`
	Collection      string            `json:"collection"`
	RequestPath     string            `json:"request_path"`
	RequestName     string            `json:"request_name,omitempty"`
	Environment     string            `json:"environment,omitempty"`
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Status          string            `json:"status,omitempty"`
	StatusCode      int               `json:"status_code,omitempty"`
	ElapsedMS       int64             `json:"elapsed_ms"`
	ResponsePreview string            `json:"response_preview,omitempty"`
	Error           string            `json:"error,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
}

// MaxPreviewBytes caps the inline response preview.
const MaxPreviewBytes = 4096

// MaxRead caps how many entries Load returns (most recent first).
const MaxRead = 500

// Store reads and writes the history file at Path.
type Store struct {
	Path string
}

// DefaultPath returns the default history file path, honouring XDG_DATA_HOME.
func DefaultPath() (string, error) {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "brio", "history.jsonl"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "brio", "history.jsonl"), nil
}

// New returns a Store at path (creates parent directory lazily on Append).
func New(path string) *Store { return &Store{Path: path} }

// NewDefault returns a Store at DefaultPath().
func NewDefault() (*Store, error) {
	p, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return New(p), nil
}

// Append writes one entry. Creates parent dirs if missing.
func (s *Store) Append(e Entry) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if len(e.ResponsePreview) > MaxPreviewBytes {
		e.ResponsePreview = e.ResponsePreview[:MaxPreviewBytes]
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(e)
}

// Load returns up to MaxRead most recent entries (newest first).
func (s *Store) Load() ([]Entry, error) {
	f, err := os.Open(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var all []Entry
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			var e Entry
			if jerr := json.Unmarshal(line, &e); jerr == nil {
				all = append(all, e)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	// Reverse → newest first.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	if len(all) > MaxRead {
		all = all[:MaxRead]
	}
	return all, nil
}
