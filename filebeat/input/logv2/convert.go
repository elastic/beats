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

package logv2

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// configType is the type of the configuration value.
type configType int

const (
	ConfTypeBool configType = iota
	// ConfTypeConstant is used for types that are a boolean in the Log input,
	// but a constant string needs to be set for Filestream
	ConfTypeConstant
	ConfTypeInt
	// ConfTypeMap is used for map and arrays, any object.
	ConfTypeMap
	ConfTypeString
)

// configField describes the conversion from the Log input to Filestream input.
type configField struct {
	// fsName is the name of the field in Filestream
	fsName string
	// fsVal for ConfTypeConstant, this value is set when the Log input
	// value is true
	fsVal string
	// kind the type of the configuration field
	kind configType
}

// inputConvTable is a conversion table from the Log input configuration
// to their Filestream equivalent.
var inputConvTable = map[string]configField{
	"backoff":                {fsName: "backoff.init", kind: ConfTypeString},
	"clean_inactive":         {fsName: "clean_inactive", kind: ConfTypeString},
	"clean_removed":          {fsName: "clean_removed", kind: ConfTypeBool},
	"close_eof":              {fsName: "close.reader.on_eof", kind: ConfTypeBool},
	"close_inactive":         {fsName: "close.on_state_change.inactive", kind: ConfTypeString},
	"close_removed":          {fsName: "close.on_state_change.removed", kind: ConfTypeBool},
	"close_renamed":          {fsName: "close.on_state_change.renamed", kind: ConfTypeBool},
	"close_timeout":          {fsName: "close.reader.after_interval", kind: ConfTypeString},
	"encoding":               {fsName: "encoding", kind: ConfTypeString},
	"exclude_files":          {fsName: "prospector.scanner.exclude_files", kind: ConfTypeMap},
	"exclude_lines":          {fsName: "exclude_lines", kind: ConfTypeMap},
	"file_identity":          {fsName: "file_identity", kind: ConfTypeMap},
	"harvester_buffer_size":  {fsName: "buffer_size", kind: ConfTypeInt},
	"harvester_limit":        {fsName: "harvester_limit", kind: ConfTypeInt},
	"ignore_older":           {fsName: "ignore_older", kind: ConfTypeString},
	"include_lines":          {fsName: "include_lines", kind: ConfTypeMap},
	"max_backoff":            {fsName: "backoff.max", kind: ConfTypeString},
	"max_bytes":              {fsName: "message_max_bytes", kind: ConfTypeInt},
	"recursive_glob.enabled": {fsName: "prospector.scanner.recursive_glob", kind: ConfTypeBool},
	"scan_frequency":         {fsName: "prospector.scanner.check_interval", kind: ConfTypeString},
	"symlinks":               {fsName: "prospector.scanner.symlinks", kind: ConfTypeBool},
	"tail_files":             {fsName: "ignore_inactive", fsVal: "since_last_start", kind: ConfTypeConstant},
}

// multilineConvTable conversion table for the multiline.* fields
var multilineConvTable = map[string]configField{
	"count_lines":   {fsName: "count_lines", kind: ConfTypeInt},
	"flush_pattern": {fsName: "flush_pattern", kind: ConfTypeString},
	"match":         {fsName: "match", kind: ConfTypeString},
	"max_lines":     {fsName: "max_lines", kind: ConfTypeInt},
	"negate":        {fsName: "negate", kind: ConfTypeBool},
	"pattern":       {fsName: "pattern", kind: ConfTypeString},
	"skip_newline":  {fsName: "skip_newline", kind: ConfTypeBool},
	"timeout":       {fsName: "timeout", kind: ConfTypeString},
	"type":          {fsName: "type", kind: ConfTypeString},
}

// logInputExclusiveKeys are all the keys we need to remove from the final
// configuration because they only exist in the Log input
var logInputExclusiveKeys = []string{
	"backoff",
	"backoff_factor",
	"close_eof",
	"close_inactive",
	"close_removed",
	"close_renamed",
	"close_timeout",
	"exclude_files",
	"harvester_buffer_size",
	"json",
	"max_backoff",
	"max_bytes",
	"multiline",
	"recursive_glob.enabled",
	"scan",
	"scan_frequency",
	"symlinks",
	"tail_files",
}

// convertConfig converts the Log input configuration to Filestream.
func convertConfig(logger *logp.Logger, cfg *config.C) (*config.C, error) {
	newCfg := config.NewConfig()

	// Merge operations overwrites everything that is in the destination
	// config, so we first merge both configs to ensure shared and common
	// fields are passed to the new one.
	if err := newCfg.Merge(cfg); err != nil {
		return nil, fmt.Errorf("cannot merge configurations: %w", err)
	}

	// Then we remove the log input exclusive fields from the new config,
	// this also removes any field that has a different type, like backoff
	for _, key := range logInputExclusiveKeys {
		if _, err := newCfg.Remove(key, -1); err != nil {
			return nil, fmt.Errorf("cannot remove %q: %w", key, err)
		}
	}

	// Convert all the "static" configuration, those are the fields that
	// can easily be translated by name.
	for key, kind := range inputConvTable {
		has, err := cfg.Has(key, -1)
		if err != nil {
			return nil, fmt.Errorf("cannot read %q: %w", key, err)
		}

		if has {
			if err := translateField(logger, cfg, newCfg, key, kind); err != nil {
				return nil, err
			}
		}
	}

	if err := handleParsers(logger, cfg, newCfg); err != nil {
		return nil, err
	}

	if err := handleFileIdentity(cfg, newCfg); err != nil {
		return nil, err
	}

	// Add final fields
	if err := newCfg.SetString("type", -1, "filestream"); err != nil {
		return nil, fmt.Errorf("cannot set 'type': %w", err)
	}

	if err := newCfg.SetBool("take_over.enabled", -1, true); err != nil {
		return nil, fmt.Errorf("cannot set 'take_over.enabled': %w", err)
	}

	return newCfg, nil
}

// translateField translates a single field from the Log input syntax to Filestream.
// If a config cannot be read as it specified type, it's either an empty key in the
// YAML or it's actually an invalid value, either way, we just ignore it, return
// no error and log at warning level.
func translateField(logger *logp.Logger, cfg, newCfg *config.C, key string, kind configField) error {
	switch kind.kind {
	// If there is any error reading the config as the correct type,
	// either the config is empty or invalid, either way, it
	// is safe to ignore it.
	case ConfTypeString:
		v, err := cfg.String(key, -1)
		if err != nil {
			logger.Warnf("cannot read %q as string: %s, ignoring malformed config entry", key, err)
			return nil
		}
		if v != "null" {
			// empty config keys appear as the `null` string, we also ignore it
			if err := newCfg.SetString(kind.fsName, -1, v); err != nil {
				return fmt.Errorf("cannot set %q: %w", kind.fsName, err)
			}
		}
	case ConfTypeBool:
		v, err := cfg.Bool(key, -1)
		if err != nil {
			logger.Warnf("cannot read %q as bool: %s, ignoring malformed config entry", key, err)
			return nil
		}
		if err := newCfg.SetBool(kind.fsName, -1, v); err != nil {
			return fmt.Errorf("cannot set %q: %w", kind.fsName, err)
		}
	case ConfTypeInt:
		v, err := cfg.Int(key, -1)
		if err != nil {
			logger.Warnf("cannot read %q as int: %s, ignoring malformed config entry", key, err)
			return nil
		}
		if err := newCfg.SetInt(kind.fsName, -1, v); err != nil {
			return fmt.Errorf("cannot set %q: %w", kind.fsName, err)
		}
	case ConfTypeMap:
		child, err := cfg.Child(key, -1)
		if err != nil {
			logger.Warnf("cannot read %q as map: %s, ignoring malformed config entry", key, err)
			return nil
		}
		if err := newCfg.SetChild(kind.fsName, -1, child); err != nil {
			return fmt.Errorf("cannot set %q: %w", kind.fsName, err)
		}
	case ConfTypeConstant:
		v, err := cfg.Bool(key, -1)
		if err != nil {
			logger.Warnf("cannot read %q as bool: %s, ignoring malformed config entry", key, err)
			return nil
		}
		if v {
			if err := newCfg.SetString(kind.fsName, -1, kind.fsVal); err != nil {
				return fmt.Errorf("cannot set %q: %w", kind.fsName, err)
			}
		}
	}

	return nil
}

// handleMultiline converts the multiline configuration. There is at least one
// field in the multiline configuration that cannot be directly copied, so
// we have to call [translateField].
func handleMultiline(logger *logp.Logger, cfg *config.C, parsers *[]any) error {
	hasMultiline, err := cfg.Has("multiline", -1)
	if err != nil {
		return fmt.Errorf("cannot access 'multiline' field: %w", err)
	}

	if !hasMultiline {
		return nil
	}

	newMultilineCfg := config.NewConfig()
	multilineCfg, err := cfg.Child("multiline", -1)
	if err != nil {
		logger.Warnf("cannot read 'multiline' as map: %s, ignoring malformed config entry ", err)
		return nil
	}

	count, err := multilineCfg.CountField("")
	if err != nil {
		logger.Warnf("cannot count elements from 'multiline': %s, ignoring malformed config entry", err)
		return nil
	}

	// Return early if multiline is an empty entry
	if count == 0 {
		return nil
	}

	for key, kind := range multilineConvTable {
		has, err := multilineCfg.Has(key, -1)
		if err != nil {
			return fmt.Errorf("cannot read 'multiline.%s': %w", key, err)
		}

		if !has {
			continue
		}

		if err := translateField(logger, multilineCfg, newMultilineCfg, key, kind); err != nil {
			return err
		}
	}

	*parsers = append(*parsers, map[string]any{
		"multiline": multilineCfg,
	})

	return nil
}

// handleJSON copies the JSON configuration from the Log input into the
// parsers array from Filestream
func handleJSON(logger *logp.Logger, cfg *config.C, parsers *[]any) error {
	hasJson, err := cfg.Has("json", -1)
	if err != nil {
		return fmt.Errorf("cannot read 'json': %w", err)
	}

	if !hasJson {
		return nil
	}

	jsonCfg, err := cfg.Child("json", -1)
	if err != nil {
		logger.Warnf("cannot read 'json' as map: %s, ignoring malformed config entry ", err)
		return nil
	}

	// If jsonCfg has no elements (it's empty) or if there is any error
	// trying to read the number of element, return.
	// Any error here means an malformed config, therefore it does not
	// need to be converted to Filestream configuration format
	count, err := jsonCfg.CountField("")
	if err != nil || count == 0 {
		return nil //nolint:nilerr // On invalid config nothing and return.
	}

	keysUnderRoot, err := jsonCfg.Bool("keys_under_root", -1)
	if err != nil {
		logger.Warnf(
			"cannot read 'json.keys_under_root' as boolean: %s, ignoring malformed config entry ",
			err,
		)
	}

	if !keysUnderRoot {
		if err := jsonCfg.SetString("target", -1, "json"); err != nil {
			return fmt.Errorf("cannot set 'target' in the ndjson parser: %w", err)
		}
	}

	*parsers = append(*parsers, map[string]any{"ndjson": jsonCfg})

	return nil
}

// copyParsers copies any existing 'parsers' from cfg into parsersCfg.
// offset is the offset in parsersCfg to start adding the new ones.
func copyParsers(cfg, parsersCfg *config.C, offset int) error {
	logParers, err := cfg.Child("parsers", -1)
	if err != nil {
		return fmt.Errorf("cannot access 'parsers' from config: %w", err)
	}

	lenParsers, err := logParers.CountField("")
	if err != nil {
		return fmt.Errorf("cannot access the length of 'parsers': %w", err)
	}

	for i := range lenParsers {
		el, err := logParers.Child("", i)
		if err != nil {
			return fmt.Errorf("cannot access 'parsers.%d': %w", i, err)
		}

		idx := offset + i
		if err := parsersCfg.SetChild("", idx, el); err != nil {
			return fmt.Errorf("cannot set 'parsers.%d': %w", idx, err)
		}
	}

	return nil
}

// handleParsers converts the multiline and JSON configuration by calling
// [handleMultiline] and [handleJSON], adds them all into a parsers array
// and finally copies any other parsers, if any, at the end of the array.
func handleParsers(logger *logp.Logger, cfg, newCfg *config.C) error {
	parsers := []any{}
	if err := handleMultiline(logger, cfg, &parsers); err != nil {
		return err
	}

	if err := handleJSON(logger, cfg, &parsers); err != nil {
		return err
	}

	// If no parsers were created, return. Any pre-existing parsers
	// entry will be maintained and passed as is to Filestream.
	if len(parsers) <= 0 {
		return nil
	}

	parsersCfg, err := config.NewConfigFrom(parsers)
	if err != nil {
		return fmt.Errorf("cannot process the converted parsers config: %w", err)
	}

	// If 'parsers' was also set in the Log input configuration, add them at the
	// end of the new 'parsers' array.
	if cfg.HasField("parsers") {
		if err := copyParsers(cfg, parsersCfg, len(parsers)); err != nil {
			return err
		}
	}

	if err := newCfg.SetChild("parsers", -1, parsersCfg); err != nil {
		return fmt.Errorf("cannot set 'parsers': %w", err)
	}

	return nil
}

// handleFileIdentity sets the file identity using the following rules:
//   - If no file identity is set, default to 'native' (same as Log input)
//   - If file identity is set, keep it as is
//   - If file identity is NOT fingerprint, disable fingerprint in the scanner
func handleFileIdentity(cfg, newCfg *config.C) error {
	if cfg.HasField("file_identity") {
		isFingerprint, err := cfg.Has("file_identity.fingerprint", -1)
		if err != nil {
			return fmt.Errorf("cannot read 'file_identity.fingerprint': %w", err)
		}

		if isFingerprint {
			return nil
		}

		// If the file identity is not fingerprint,
		// disable fingerprint in the scanner
		if err := newCfg.SetBool("prospector.scanner.fingerprint.enabled", -1, false); err != nil {
			return fmt.Errorf("cannot set 'prospector.scanner.fingerprint.enalbed': %w", err)
		}

		return nil
	}

	if err := newCfg.SetChild("file_identity.native", -1, config.NewConfig()); err != nil {
		return fmt.Errorf("cannot set 'file_identity.native': %w", err)
	}

	if err := newCfg.SetBool("prospector.scanner.fingerprint.enabled", -1, false); err != nil {
		return fmt.Errorf("cannot set 'prospector.scanner.fingerprint.enalbed': %w", err)
	}

	return nil
}
