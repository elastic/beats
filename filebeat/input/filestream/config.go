// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package filestream

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Compression mode constants
const (
	// CompressionNone disables compression handling; all files are treated as
	// plain text.
	CompressionNone = ""
	// CompressionGZIP treats all files as gzip compressed.
	CompressionGZIP = "gzip"
	// CompressionAuto auto-detects gzip files and decompresses them.
	CompressionAuto = "auto"
)

// config stores the options of a file stream.
type config struct {
	Reader readerConfig `config:",inline"`

	ID           string            `config:"id"`
	Paths        []string          `config:"paths"`
	Close        closerConfig      `config:"close"`
	FileWatcher  fileWatcherConfig `config:"prospector.scanner"`
	FileIdentity *conf.Namespace   `config:"file_identity"`

	// Compression specifies how file compression is handled.
	// Valid values: "" (none), "gzip" (all files are gzip), "auto"
	// (auto-detect).
	Compression string `config:"compression"`

	// GZIPExperimental is deprecated and is ignored. Use Compression instead.
	// Deprecated.
	GZIPExperimental *bool `config:"gzip_experimental"`

	// Whether to add the file owner name and group to the event metadata.
	// Disabled by default.
	IncludeFileOwnerName      bool `config:"include_file_owner_name"`
	IncludeFileOwnerGroupName bool `config:"include_file_owner_group_name"`

	// -1 means that registry will never be cleaned, disabling clean_inactive.
	// Setting it to 0 also disables clean_inactive
	// "clean_inactive" is parsed, again, and used by internal/input-logfile/manager.go
	CleanInactive  time.Duration      `config:"clean_inactive" validate:"min=-1"`
	CleanRemoved   bool               `config:"clean_removed"`
	HarvesterLimit uint32             `config:"harvester_limit" validate:"min=0"`
	IgnoreOlder    time.Duration      `config:"ignore_older"`
	IgnoreInactive ignoreInactiveType `config:"ignore_inactive"`
	Rotation       *conf.Namespace    `config:"rotation"`
	Delete         deleterConfig      `config:"delete"`

	// TakeOver is also independently parsed by InputManager.Create
	// (see internal/input-logfile/manager.go).
	TakeOver loginp.TakeOverConfig `config:"take_over"`
	// AllowIDDuplication is used by InputManager.Create
	// (see internal/input-logfile/manager.go).
	AllowIDDuplication bool `config:"allow_deprecated_id_duplication"`

	// LegacyCleanInactive disables the clean_inactive validation,
	// and allows it to be set to 0, re-ingesting all files on restart,
	// effectively preserving the old, buggy, behaviour.
	// (see internal/input-logfile/manager.go)
	LegacyCleanInactive bool `config:"legacy_clean_inactive"`
}

type deleterConfig struct {
	Enabled     bool          `config:"enabled"`
	GracePeriod time.Duration `config:"grace_period"`

	// configurable for testing
	retries      int           `config:"-"`
	retryBackoff time.Duration `config:"-"`
}

type closerConfig struct {
	OnStateChange stateChangeCloserConfig `config:"on_state_change"`
	Reader        readerCloserConfig      `config:"reader"`
}

type readerCloserConfig struct {
	AfterInterval time.Duration `config:"after_interval"`
	OnEOF         bool          `config:"on_eof"`
}

type stateChangeCloserConfig struct {
	CheckInterval time.Duration `config:"check_interval" validate:"nonzero"`
	Inactive      time.Duration `config:"inactive"`
	Removed       bool          `config:"removed"`
	Renamed       bool          `config:"renamed"`
}

type readerConfig struct {
	Backoff        backoffConfig           `config:"backoff"`
	BufferSize     int                     `config:"buffer_size"`
	Encoding       string                  `config:"encoding"`
	ExcludeLines   []match.Matcher         `config:"exclude_lines"`
	IncludeLines   []match.Matcher         `config:"include_lines"`
	LineTerminator readfile.LineTerminator `config:"line_terminator"`
	MaxBytes       int                     `config:"message_max_bytes" validate:"min=0,nonzero"`
	Tail           bool                    `config:"seek_to_tail"`

	Parsers parser.Config `config:",inline"`
}

type backoffConfig struct {
	Init time.Duration `config:"init" validate:"nonzero"`
	Max  time.Duration `config:"max" validate:"nonzero"`
}

type rotationConfig struct {
	Strategy *conf.Namespace `config:"strategy" validate:"required"`
}

type commonRotationConfig struct {
	SuffixRegex string `config:"suffix_regex" validate:"required"`
	DateFormat  string `config:"dateformat"`
}

type copyTruncateConfig commonRotationConfig

func defaultConfig() config {
	return config{
		Reader:                    defaultReaderConfig(),
		Paths:                     []string{},
		Close:                     defaultCloserConfig(),
		IncludeFileOwnerName:      false,
		IncludeFileOwnerGroupName: false,
		CleanInactive:             -1,
		CleanRemoved:              true,
		HarvesterLimit:            0,
		IgnoreOlder:               0,
		Delete:                    defaultDeleterConfig(),
		FileWatcher:               defaultFileWatcherConfig(), // Config key: prospector.scanner
	}
}

func defaultCloserConfig() closerConfig {
	return closerConfig{
		OnStateChange: stateChangeCloserConfig{
			CheckInterval: 5 * time.Second,
			Removed:       defaultCloserOnStateChangeRemoved(),
			Inactive:      5 * time.Minute,
			Renamed:       false,
		},
		Reader: readerCloserConfig{
			OnEOF:         false,
			AfterInterval: 0 * time.Second,
		},
	}
}

func defaultReaderConfig() readerConfig {
	return readerConfig{
		Backoff: backoffConfig{
			Init: 2 * time.Second,
			Max:  10 * time.Second,
		},
		BufferSize:     16 * humanize.KiByte,
		LineTerminator: readfile.AutoLineTerminator,
		MaxBytes:       10 * humanize.MiByte,
		Tail:           false,
	}
}

func defaultDeleterConfig() deleterConfig {
	return deleterConfig{
		GracePeriod:  30 * time.Minute,
		retries:      5,
		retryBackoff: 2 * time.Second,
	}
}

func (c *config) Validate() error {
	if len(c.Paths) == 0 {
		return fmt.Errorf("no path is configured")
	}

	if !c.LegacyCleanInactive {
		// clean_inactive can only be used if ignore_older is enabled
		if c.IgnoreOlder == 0 && c.CleanInactive > 0 {
			return errors.New("clean_inactive can only be enabled if ignore_older is also enabled")
		}

		// clean_inactive is only enabled if clean_inactive > 0
		if c.CleanInactive > 0 {
			if c.CleanInactive <= c.IgnoreOlder+c.FileWatcher.Interval {
				return fmt.Errorf("clean_inactive must be greater than ignore_older + "+
					"prospector.scanner.check_interval, however %s <= %s + %s",
					c.CleanInactive.String(),
					c.IgnoreOlder.String(),
					c.FileWatcher.Interval.String())
			}
		}

		if c.AllowIDDuplication && c.TakeOver.Enabled {
			return errors.New("allow_deprecated_id_duplication and take_over " +
				"cannot be enabled at the same time")
		}
	}

	switch c.Compression {
	case CompressionNone:
		// no validation needed
	case CompressionGZIP, CompressionAuto:
		if c.FileIdentity != nil && c.FileIdentity.Name() != fingerprintName {
			return fmt.Errorf(
				"compression='%s' requires 'file_identity' to be 'fingerprint'. Current file_identity is '%s'",
				c.Compression, c.FileIdentity.Name())
		}
	default:
		return fmt.Errorf("invalid compression value %q, must be one of: %q, %q, %q",
			c.Compression, CompressionNone, CompressionGZIP, CompressionAuto)
	}

	if c.ID == "" && c.TakeOver.Enabled {
		return errors.New("'take_over' mode is only allowed if an input ID is set")
	}

	return nil
}

// checkUnsupportedParams checks if unsupported/deprecated/discouraged parameters are set and logs a warning
func (c config) checkUnsupportedParams(logger *logp.Logger) {
	if c.AllowIDDuplication {
		logger.Named("filestream").Warn(
			"setting `allow_deprecated_id_duplication` will lead to data " +
				"duplication and incomplete input metrics, it's use is " +
				"highly discouraged.")
	}
	if c.GZIPExperimental != nil {
		if c.Compression != CompressionNone {
			logger.Named("filestream").Warn(cfgwarn.Deprecate(
				"",
				"'gzip_experimental' is deprecated and ignored. 'compression' is set, using it instead"))
		} else {
			logger.Named("filestream").Warn(cfgwarn.Deprecate(
				"",
				"'gzip_experimental' is deprecated and ignored, set 'compression' instead"))
		}
	}
}

// ValidateInputIDs checks all filestream inputs to ensure all input IDs are
// unique. If there is a duplicated ID, it logs an error containing the offending
// input configurations and returns an error containing the duplicated IDs.
// A single empty ID is a valid ID as it's unique, however multiple empty IDs
// are not unique and are therefore are treated as any other duplicated ID.
func ValidateInputIDs(inputs []*conf.C, logger *logp.Logger) error {
	duplicatedConfigs := make(map[string][]*conf.C)
	var duplicates []string
	for _, input := range inputs {
		fsInput := struct {
			ID   string `config:"id"`
			Type string `config:"type"`
		}{}
		err := input.Unpack(&fsInput)
		if err != nil {
			return fmt.Errorf("failed to unpack filestream input configuration: %w", err)
		}
		if fsInput.Type == "filestream" {
			duplicatedConfigs[fsInput.ID] = append(duplicatedConfigs[fsInput.ID], input)
			// we just need to collect the duplicated IDs once, therefore collect
			// it only the first time we see a duplicated ID.
			if len(duplicatedConfigs[fsInput.ID]) == 2 {
				duplicates = append(duplicates, fsInput.ID)
			}
		}
	}

	if len(duplicates) != 0 {
		jsonDupCfg := collectOffendingInputs(duplicates, duplicatedConfigs)
		logger.Errorw("filestream inputs with duplicated IDs", "inputs", jsonDupCfg)
		var quotedDuplicates []string
		for _, dup := range duplicates {
			quotedDuplicates = append(quotedDuplicates, fmt.Sprintf("%q", dup))
		}
		return fmt.Errorf("filestream inputs validation error: filestream inputs with duplicated IDs: %v", strings.Join(quotedDuplicates, ","))
	}

	return nil
}

func collectOffendingInputs(duplicates []string, ids map[string][]*conf.C) []map[string]interface{} {
	var cfgs []map[string]interface{}

	for _, id := range duplicates {
		for _, dupcfgs := range ids[id] {
			toJson := map[string]interface{}{}
			err := dupcfgs.Unpack(&toJson)
			if err != nil {
				toJson[id] = fmt.Sprintf("failed to unpack config: %v", err)
			}
			cfgs = append(cfgs, toJson)
		}
	}

	return cfgs
}
