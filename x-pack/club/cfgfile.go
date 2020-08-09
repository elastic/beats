package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

type configReader struct {
	paths      pathSettings
	strictPerm bool
}

type configWatcher struct {
	log    *logp.Logger
	reader *configReader
	files  []string
}

func (r *configReader) Read(files []string) (*common.Config, error) {
	return readConfigFiles(r.paths, files, r.strictPerm)
}

func (w *configWatcher) Run(cancel unison.Canceler, handler func(dynamicSettings) error) error {
	lastHash, err := hashFiles(w.files)
	if err != nil {
		w.log.Errorf("Hashing configuration files failed with: %v", err)
	}

	return periodic(cancel, 250*time.Millisecond, func() error {
		hash, err := hashFiles(w.files)
		if err != nil {
			w.log.Errorf("Hashing configuration files failed with: %v", err)
			return nil
		}

		if hash != lastHash {
			lastHash = hash

			if err := w.onChange(handler); err != nil {
				w.log.Errorf("Failed to apply updated configuration: %v", err)
			}
		}

		return nil
	})
}

func (w *configWatcher) onChange(handler func(dynamicSettings) error) error {
	cfg, err := w.reader.Read(w.files)
	if err != nil {
		return fmt.Errorf("reading configuration failed: %w", err)
	}

	var settings dynamicSettings
	if err := cfg.Unpack(&settings); err != nil {
		return fmt.Errorf("failed to parse settings: %v", err)
	}

	if err := handler(settings); err != nil {
		return err
	}

	return nil
}

func readConfigFiles(paths pathSettings, files []string, strictPerm bool) (*common.Config, error) {
	configFilePaths := make([]string, len(files))
	for i, path := range files {
		if !filepath.IsAbs(path) {
			path = filepath.Join(paths.Config, path)
		}
		configFilePaths[i] = path
	}

	if strictPerm {
		for _, path := range configFilePaths {
			if err := common.OwnerHasExclusiveWritePerms(path); err != nil {
				return nil, err
			}
		}
	}

	config := common.NewConfig()
	for _, path := range configFilePaths {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		fileConfig, err := common.NewConfigWithYAML(contents, path)
		if err != nil {
			return nil, err
		}

		if err = config.Merge(fileConfig); err != nil {
			return nil, err
		}
	}

	return config, nil
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
