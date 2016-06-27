package harvester

import (
	"fmt"
	"regexp"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/processor"
	"github.com/elastic/beats/libbeat/common"

	"github.com/dustin/go-humanize"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	defaultConfig = harvesterConfig{
		BufferSize:      16 * humanize.KiByte,
		DocumentType:    "log",
		InputType:       cfg.DefaultInputType,
		TailFiles:       false,
		Backoff:         1 * time.Second,
		BackoffFactor:   2,
		MaxBackoff:      10 * time.Second,
		CloseOlder:      1 * time.Hour,
		MaxBytes:        10 * (1 << 20), // 10MB
		CloseRemoved:    false,
		CloseRenamed:    false,
		ForceCloseFiles: false,
	}
)

type harvesterConfig struct {
	common.EventMetadata `config:",inline"`         // Fields and tags to add to events.
	BufferSize           int                        `config:"harvester_buffer_size"`
	DocumentType         string                     `config:"document_type"`
	Encoding             string                     `config:"encoding"`
	InputType            string                     `config:"input_type"`
	TailFiles            bool                       `config:"tail_files"`
	Backoff              time.Duration              `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor        int                        `config:"backoff_factor" validate:"min=1"`
	MaxBackoff           time.Duration              `config:"max_backoff" validate:"min=0,nonzero"`
	CloseOlder           time.Duration              `config:"close_older"`
	CloseRemoved         bool                       `config:"close_removed"`
	CloseRenamed         bool                       `config:"close_renamed"`
	ForceCloseFiles      bool                       `config:"force_close_files"`
	ExcludeLines         []*regexp.Regexp           `config:"exclude_lines"`
	IncludeLines         []*regexp.Regexp           `config:"include_lines"`
	MaxBytes             int                        `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline            *processor.MultilineConfig `config:"multiline"`
	JSON                 *processor.JSONConfig      `config:"json"`
}

func (config *harvesterConfig) Validate() error {

	// TODO: remove in 7.0
	if config.ForceCloseFiles {
		config.CloseRemoved = true
		config.CloseRenamed = true
		logp.Warn("DEPRECATED: force_close_files was set to true. Use close_removed + close_rename")
	}

	// Check input type
	if _, ok := cfg.ValidInputType[config.InputType]; !ok {
		return fmt.Errorf("Invalid input type: %v", config.InputType)
	}

	if config.JSON != nil && len(config.JSON.MessageKey) == 0 &&
		config.Multiline != nil {
		return fmt.Errorf("When using the JSON decoder and multiline together, you need to specify a message_key value")
	}

	if config.JSON != nil && len(config.JSON.MessageKey) == 0 &&
		(len(config.IncludeLines) > 0 || len(config.ExcludeLines) > 0) {
		return fmt.Errorf("When using the JSON decoder and line filtering together, you need to specify a message_key value")
	}

	return nil
}
