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
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
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

// Preset names, the keys accepted under the 'preset' option.
const (
	// presetAuto selects high_cardinality when the input has more files than
	// the configured threshold, otherwise default.
	presetAuto = "auto"
	// presetDefault keeps the configured values and the regular behaviour.
	presetDefault = "default"
	// presetHighCardinality tunes the input for a large number of files.
	presetHighCardinality = "high_cardinality"
)

// defaultFilesAbove is the number of files above which the auto
// preset selects high_cardinality.
const defaultFilesAbove = 5000

// config stores the options of a file stream.
type config struct {
	Reader readerConfig `config:",inline"`

	ID           string                    `config:"id"`
	Paths        []string                  `config:"paths"`
	Close        closerConfig              `config:"close"`
	ReadUntilEOF loginp.ReadUntilEOFConfig `config:"read_until_eof"`

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
	// IncludeFileFingerprint controls whether log.file.fingerprint is added
	// to published events. Enabled by default.
	IncludeFileFingerprint bool `config:"include_file_fingerprint"`

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

	// Preset selects how the input tunes itself. It holds exactly one of
	// the preset names (auto, default, high_cardinality) with its settings
	// underneath. Nil means it is not set, which behaves like auto.
	Preset *conf.Namespace `config:"preset"`

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

// presetAutoConfig holds the settings of 'preset.auto'.
type presetAutoConfig struct {
	HighCardinality highCardinalityConfig `config:"high_cardinality"`
}

// highCardinalityConfig holds the settings of 'preset.auto.high_cardinality'.
type highCardinalityConfig struct {
	// FilesAbove is the file count that has to be exceeded, strictly, for
	// the auto preset to select high_cardinality.
	FilesAbove int `config:"files_above" validate:"min=0"`
}

// presetSelection is the preset chosen by a configuration.
type presetSelection struct {
	// name is the selected preset: presetAuto, presetDefault or
	// presetHighCardinality.
	name string
	// explicit is true when the user wrote the preset.
	explicit bool
	// threshold is the file count above which presetAuto selects
	// high_cardinality.
	threshold int
	// forcedDefaultReason explains why a preset that is not set resolved to
	// presetDefault instead of presetAuto. Empty otherwise.
	forcedDefaultReason string
}

// unpackPresetAuto reads the settings of the 'preset.auto' namespace and
// applies their defaults.
func unpackPresetAuto(ns *conf.Namespace) (presetAutoConfig, error) {
	autoCfg := presetAutoConfig{
		HighCardinality: highCardinalityConfig{
			FilesAbove: defaultFilesAbove,
		},
	}
	if err := ns.Config().Unpack(&autoCfg); err != nil {
		return presetAutoConfig{}, fmt.Errorf("cannot read 'preset.%s' config: %w", presetAuto, err)
	}

	return autoCfg, nil
}

// cleanInactiveEnabled reports whether clean_inactive is in effect. Zero only
// counts with legacy_clean_inactive, the same answer before and after
// configure rewrites 'clean_inactive: 0' to -1.
func (c *config) cleanInactiveEnabled() bool {
	return c.CleanInactive > 0 || (c.CleanInactive == 0 && c.LegacyCleanInactive)
}

// presetIncompatibleOptions returns the enabled options that cannot be used
// with the auto and high_cardinality presets, named as the user writes them.
func (c *config) presetIncompatibleOptions() []string {
	var opts []string
	if c.Delete.Enabled {
		opts = append(opts, "delete.enabled")
	}
	if c.cleanInactiveEnabled() {
		opts = append(opts, "clean_inactive")
	}

	return opts
}

// quoteOptions formats option names for messages: each one in single quotes,
// joined with " and ".
func quoteOptions(opts []string) string {
	quoted := make([]string, 0, len(opts))
	for _, o := range opts {
		quoted = append(quoted, "'"+o+"'")
	}

	return strings.Join(quoted, " and ")
}

// presetSelection returns the preset selected by the configuration. It does
// not depend on Validate having run. A preset that is not set selects
// presetAuto with the default threshold, or presetDefault when an option
// incompatible with the other presets is enabled.
func (c *config) presetSelection() (presetSelection, error) {
	sel := presetSelection{
		name:      presetAuto,
		threshold: defaultFilesAbove,
	}
	if c.Preset == nil || c.Preset.Name() == "" {
		if opts := c.presetIncompatibleOptions(); len(opts) > 0 {
			sel.name = presetDefault
			sel.forcedDefaultReason = fmt.Sprintf(
				"'preset' is not set and %s cannot be used with 'preset.auto' or 'preset.high_cardinality'",
				quoteOptions(opts))
		}
		return sel, nil
	}

	sel.name = c.Preset.Name()
	sel.explicit = true
	switch sel.name {
	case presetAuto:
		autoCfg, err := unpackPresetAuto(c.Preset)
		if err != nil {
			return presetSelection{}, err
		}
		sel.threshold = autoCfg.HighCardinality.FilesAbove
	case presetDefault, presetHighCardinality:
		// no settings
	default:
		return presetSelection{}, fmt.Errorf("unknown preset %q, must be one of: %q, %q, %q",
			sel.name, presetAuto, presetDefault, presetHighCardinality)
	}

	return sel, nil
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
	Backoff        loginp.BackoffConfig    `config:"backoff"`
	BufferSize     int                     `config:"buffer_size" validate:"min=0,nonzero"`
	Encoding       string                  `config:"encoding"`
	ExcludeLines   []match.Matcher         `config:"exclude_lines"`
	IncludeLines   []match.Matcher         `config:"include_lines"`
	LineTerminator readfile.LineTerminator `config:"line_terminator"`
	MaxBytes       int                     `config:"message_max_bytes" validate:"min=0,nonzero"`
	Tail           bool                    `config:"seek_to_tail"`

	Parsers parser.Config `config:",inline"`
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
		ReadUntilEOF:              loginp.DefaultReadUntilEOFConfig(),
		IncludeFileOwnerName:      false,
		IncludeFileOwnerGroupName: false,
		IncludeFileFingerprint:    true,
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
			CheckInterval: loginp.DefaultStateCheckInterval,
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
		Backoff:        loginp.DefaultBackoffConfig(),
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

	sel, err := c.presetSelection()
	if err != nil {
		return fmt.Errorf("invalid preset: %w", err)
	}
	if sel.explicit && sel.name != presetDefault {
		if opts := c.presetIncompatibleOptions(); len(opts) > 0 {
			return fmt.Errorf("'preset.%s' cannot be used together with %s, use 'preset.default' instead",
				sel.name, quoteOptions(opts))
		}
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

func collectOffendingInputs(duplicates []string, ids map[string][]*conf.C) []map[string]any {
	var cfgs []map[string]any

	for _, id := range duplicates {
		for _, dupcfgs := range ids[id] {
			toJson := map[string]any{}
			err := dupcfgs.Unpack(&toJson)
			if err != nil {
				toJson[id] = fmt.Sprintf("failed to unpack config: %v", err)
			}
			cfgs = append(cfgs, toJson)
		}
	}

	return cfgs
}
