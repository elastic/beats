package cfgload

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/timed"
)

// Watcher monitors the paths given in Files for changes.
type Watcher struct {
	Log    *logp.Logger
	Files  []string
	Reader Reader
}

type Reader interface {
	ReadFiles(files []string) (*common.Config, error)
}

// Run executes the watchers main loop. It blocks until the watcher is shut down.
// The handler function is called with the merged configuration if the watcher detects any file changes.
func (w *Watcher) Run(cancel context.Context, handler func(*common.Config) error) error {
	lastHash, err := hashFiles(w.Files)
	if err != nil {
		w.Log.Errorf("Hashing configuration files failed with: %v", err)
	}

	return timed.Periodic(cancel, 250*time.Millisecond, func() error {
		hash, err := hashFiles(w.Files)
		if err != nil {
			w.Log.Errorf("Hashing configuration files failed with: %v", err)
			return nil
		}

		if hash != lastHash {
			lastHash = hash

			if err := w.onChange(handler); err != nil {
				w.Log.Errorf("Failed to apply updated configuration: %v", err)
			}
		}

		return nil
	})
}

func (w *Watcher) onChange(handler func(*common.Config) error) error {
	cfg, err := w.Reader.ReadFiles(w.Files)
	if err != nil {
		return fmt.Errorf("reading configuration failed: %w", err)
	}

	if err := handler(cfg); err != nil {
		return err
	}

	return nil
}

func hashFiles(paths []string) (string, error) {
	h := sha256.New()
	for _, path := range paths {
		if err := streamFileTo(h, path); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func streamFileTo(w io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}
