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
)

// configType is the type of the configuration value.
type configType int

const (
	Unknwon configType = iota
	ConfTypeBool
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

// inputConvTable conversion table for the log input configuration
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
	"backoff_factor", // not supported
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
	"scan", // not supported
	"scan_frequency",
	"symlinks",
	"tail_files",
}

// convertConfig convert the Log input configuration to Filestream.
func convertConfig(cfg *config.C) (*config.C, error) {
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
			switch kind.kind {
			case ConfTypeString:
				v, err := cfg.String(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as string: %w", key, err)
				}
				if err := newCfg.SetString(kind.fsName, -1, v); err != nil {
					return nil, fmt.Errorf("cannot set %q: %w", kind.fsName, err)
				}

			case ConfTypeBool:
				v, err := cfg.Bool(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as boolean: %w", key, err)
				}
				if err := newCfg.SetBool(kind.fsName, -1, v); err != nil {
					return nil, fmt.Errorf("cannot set %q: %w", kind.fsName, err)
				}

			case ConfTypeInt:
				v, err := cfg.Int(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as integer: %w", key, err)
				}
				if err := newCfg.SetInt(kind.fsName, -1, v); err != nil {
					return nil, fmt.Errorf("cannot set %q: %w", kind.fsName, err)
				}

			case ConfTypeMap:
				child, err := cfg.Child(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as map/array: %w", key, err)
				}
				if err := newCfg.SetChild(kind.fsName, -1, child); err != nil {
					return nil, fmt.Errorf("cannot set %q: %w", kind.fsName, err)
				}

			case ConfTypeConstant:
				v, err := cfg.Bool(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as boolean: %w", key, err)
				}
				if v {
					if err := newCfg.SetString(kind.fsName, -1, kind.fsVal); err != nil {
						return nil, fmt.Errorf("cannot set %q: %w", kind.fsName, err)
					}
				}
			}
		}
	}

	// Now handle the trick bits, starting with parsers
	// The first parser is Multiline, then JSON

	hasMultiline, err := cfg.Has("multiline", -1)
	if err != nil {
		return nil, fmt.Errorf("cannot access 'json' field: %w", err)
	}

	parsers := []any{}
	if hasMultiline {
		multilineCfg := config.NewConfig()
		multilineChild, err := cfg.Child("multiline", -1)
		if err != nil {
			return nil, fmt.Errorf("cannot access 'multiline': %w", err)
		}

		for key, kind := range multilineConvTable {
			has, err := multilineChild.Has(key, -1)
			if err != nil {
				return nil, fmt.Errorf("cannot read 'multiline.%s': %w", key, err)
			}

			if has {
				switch kind.kind {
				case ConfTypeString:
					v, err := multilineChild.String(key, -1)
					if err != nil {
						return nil, fmt.Errorf("cannot read %q as string: %w", key, err)
					}
					if err := multilineCfg.SetString(kind.fsName, -1, v); err != nil {
						return nil, fmt.Errorf("cannot set %q: %w", key, err)
					}

				case ConfTypeBool:
					v, err := multilineChild.Bool(key, -1)
					if err != nil {
						return nil, fmt.Errorf("cannot read %q as boolean: %w", key, err)
					}
					if err := multilineCfg.SetBool(kind.fsName, -1, v); err != nil {
						return nil, fmt.Errorf("cannot set %q: %w", key, err)
					}

				case ConfTypeInt:
					v, err := multilineChild.Int(key, -1)
					if err != nil {
						return nil, fmt.Errorf("cannot read %q as integer: %w", key, err)
					}
					if err := multilineCfg.SetInt(kind.fsName, -1, v); err != nil {
						return nil, fmt.Errorf("cannot set %q: %w", key, err)
					}
				}
			}
		}

		parsers = append(parsers, map[string]any{
			"multiline": multilineCfg,
		})
	}

	// Handle json now.
	// json has simpler types, so we can use the config directly
	hasJson, err := cfg.Has("json", -1)
	if err != nil {
		return nil, fmt.Errorf("cannot read 'json': %w", err)
	}

	if hasJson {
		jsonCfg, err := cfg.Child("json", -1)
		if err != nil {
			return nil, fmt.Errorf("cannot access 'json': %w", err)
		}

		parsers = append(parsers, map[string]any{"ndjson": jsonCfg})
	}

	// If any parsers were created, set them into the new config.
	// If the original config had a 'parsers' array, it is copied
	// after the multiline and ndjson parsers coming from the log config
	// translation
	if len(parsers) != 0 {
		parsersCfg, err := config.NewConfigFrom(parsers)
		if err != nil {
			return nil, fmt.Errorf("cannot process the converted parsers config: %w", err)
		}

		if cfg.HasField("parsers") {
			logParers, err := cfg.Child("parsers", -1)
			if err != nil {
				return nil, fmt.Errorf("cannot access 'parsers' from config: %w", err)
			}

			lenParsers, err := logParers.CountField("")
			if err != nil {
				return nil, fmt.Errorf("cannot access the length of 'parsers': %w", err)
			}

			for i := range lenParsers {
				el, err := logParers.Child("", i)
				if err != nil {
					return nil, fmt.Errorf("cannot access 'parsers.%d': %w", i, err)
				}

				idx := len(parsers) + i
				if err := parsersCfg.SetChild("", idx, el); err != nil {
					return nil, fmt.Errorf("cannot set 'parsers.%d': %w", idx, err)
				}
			}
		}

		if err := newCfg.SetChild("parsers", -1, parsersCfg); err != nil {
			return nil, fmt.Errorf("cannot set 'parsers': %w", err)
		}
	}

	// Handle file identity
	//  - If no file identity is set, default to 'native'
	//  - If file identity is set, keep it as is
	//  - If file identity is NOT fingerprint, disable fingerprint in the scanner
	disableFingeprint := true
	if !cfg.HasField("file_identity") {
		err := newCfg.SetChild("file_identity.native", -1, config.NewConfig())
		if err != nil {
			return nil, fmt.Errorf("cannot set 'file_identity.native': %w", err)
		}
	} else {
		has, err := cfg.Has("file_identity.fingerprint", -1)
		if err != nil {
			return nil, fmt.Errorf("cannot read 'file_identity.fingerprint': %w", err)
		}
		disableFingeprint = !has
	}

	if disableFingeprint {
		err := newCfg.SetBool("prospector.scanner.fingerprint.enabled", -1, false)
		if err != nil {
			return nil, fmt.Errorf("cannot set 'prospector.scanner.fingerprint.enalbed': %w", err)
		}
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
