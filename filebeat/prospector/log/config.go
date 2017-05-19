package log

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	defaultConfig = config{
		// Common
		ForwarderConfig: harvester.ForwarderConfig{
			Type: cfg.DefaultType,
		},
		CleanInactive: 0,

		// Prospector
		Enabled:        true,
		IgnoreOlder:    0,
		ScanFrequency:  10 * time.Second,
		CleanRemoved:   true,
		HarvesterLimit: 0,
		Symlinks:       false,
		TailFiles:      false,

		// Harvester
		BufferSize: 16 * humanize.KiByte,
		MaxBytes:   10 * humanize.MiByte,
		LogConfig: LogConfig{
			Backoff:       1 * time.Second,
			BackoffFactor: 2,
			MaxBackoff:    10 * time.Second,
			CloseInactive: 5 * time.Minute,
			CloseRemoved:  true,
			CloseRenamed:  false,
			CloseEOF:      false,
			CloseTimeout:  0,
		},
	}
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	LogConfig                 `config:",inline"`

	// Common
	InputType     string        `config:"input_type"`
	CleanInactive time.Duration `config:"clean_inactive" validate:"min=0"`

	// Prospector
	Enabled        bool            `config:"enabled"`
	ExcludeFiles   []match.Matcher `config:"exclude_files"`
	IgnoreOlder    time.Duration   `config:"ignore_older"`
	Paths          []string        `config:"paths"`
	ScanFrequency  time.Duration   `config:"scan_frequency" validate:"min=0,nonzero"`
	CleanRemoved   bool            `config:"clean_removed"`
	HarvesterLimit uint64          `config:"harvester_limit" validate:"min=0"`
	Symlinks       bool            `config:"symlinks"`
	TailFiles      bool            `config:"tail_files"`
	RecursiveGlob  bool            `config:"recursive_glob.enabled"`

	// Harvester
	BufferSize int    `config:"harvester_buffer_size"`
	Encoding   string `config:"encoding"`

	ExcludeLines []match.Matcher         `config:"exclude_lines"`
	IncludeLines []match.Matcher         `config:"include_lines"`
	MaxBytes     int                     `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline    *reader.MultilineConfig `config:"multiline"`
	JSON         *reader.JSONConfig      `config:"json"`
}

type LogConfig struct {
	Backoff       time.Duration `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor int           `config:"backoff_factor" validate:"min=1"`
	MaxBackoff    time.Duration `config:"max_backoff" validate:"min=0,nonzero"`
	CloseInactive time.Duration `config:"close_inactive"`
	CloseRemoved  bool          `config:"close_removed"`
	CloseRenamed  bool          `config:"close_renamed"`
	CloseEOF      bool          `config:"close_eof"`
	CloseTimeout  time.Duration `config:"close_timeout" validate:"min=0"`
}

func (c *config) Validate() error {

	// DEPRECATED 6.0.0: warning is already outputted on propsector level
	if c.InputType != "" {
		c.Type = c.InputType
	}

	// Prospector
	if c.Type == harvester.LogType && len(c.Paths) == 0 {
		return fmt.Errorf("No paths were defined for prospector")
	}

	if c.CleanInactive != 0 && c.IgnoreOlder == 0 {
		return fmt.Errorf("ignore_older must be enabled when clean_inactive is used")
	}

	if c.CleanInactive != 0 && c.CleanInactive <= c.IgnoreOlder+c.ScanFrequency {
		return fmt.Errorf("clean_inactive must be > ignore_older + scan_frequency to make sure only files which are not monitored anymore are removed")
	}

	// Harvester
	// Check input type
	if _, ok := harvester.ValidType[c.Type]; !ok {
		return fmt.Errorf("Invalid input type: %v", c.Type)
	}

	if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
		c.Multiline != nil {
		return fmt.Errorf("When using the JSON decoder and multiline together, you need to specify a message_key value")
	}

	if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
		(len(c.IncludeLines) > 0 || len(c.ExcludeLines) > 0) {
		return fmt.Errorf("When using the JSON decoder and line filtering together, you need to specify a message_key value")
	}

	return nil
}

func (c *config) resolvePaths() error {
	var paths []string
	if !c.RecursiveGlob {
		logp.Debug("prospector", "recursive glob disabled")
		paths = c.Paths
	} else {
		logp.Debug("prospector", "recursive glob enabled")
	}
	for _, path := range c.Paths {
		patterns, err := file.GlobPatterns(path, recursiveGlobDepth)
		if err != nil {
			return err
		}
		if len(patterns) > 1 {
			logp.Debug("prospector", "%q expanded to %#v", path, patterns)
		}
		paths = append(paths, patterns...)
	}
	c.Paths = paths
	return nil
}
