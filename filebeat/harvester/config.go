package harvester

import (
	"fmt"
	"regexp"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
)

var (
	defaultConfig = harvesterConfig{
		BufferSize:      cfg.DefaultHarvesterBufferSize,
		DocumentType:    cfg.DefaultDocumentType,
		InputType:       cfg.DefaultInputType,
		TailFiles:       cfg.DefaultTailFiles,
		Backoff:         cfg.DefaultBackoff,
		BackoffFactor:   cfg.DefaultBackoffFactor,
		MaxBackoff:      cfg.DefaultMaxBackoff,
		CloseOlder:      cfg.DefaultCloseOlder,
		ForceCloseFiles: cfg.DefaultForceCloseFiles,
		MaxBytes:        cfg.DefaultMaxBytes,
	}
)

type harvesterConfig struct {
	common.EventMetadata `config:",inline"`     // Fields and tags to add to events.
	BufferSize           int                    `config:"harvester_buffer_size"`
	DocumentType         string                 `config:"document_type"`
	Encoding             string                 `config:"encoding"`
	InputType            string                 `config:"input_type"`
	TailFiles            bool                   `config:"tail_files"`
	Backoff              time.Duration          `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor        int                    `config:"backoff_factor" validate:"min=1"`
	MaxBackoff           time.Duration          `config:"max_backoff" validate:"min=0,nonzero"`
	CloseOlder           time.Duration          `config:"close_older"`
	ForceCloseFiles      bool                   `config:"force_close_files"`
	ExcludeLines         []*regexp.Regexp       `config:"exclude_lines"`
	IncludeLines         []*regexp.Regexp       `config:"include_lines"`
	MaxBytes             int                    `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline            *input.MultilineConfig `config:"multiline"`
	JSON                 *input.JSONConfig      `config:"json"`
}

func (config *harvesterConfig) Validate() error {

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
