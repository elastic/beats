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

	"github.com/elastic/beats/v7/filebeat/channel"
	v1 "github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/filestream"
	loginput "github.com/elastic/beats/v7/filebeat/input/log"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const pluginName = "log"

func init() {
	// Register an input V1, that's used by the log input
	if err := v1.Register(pluginName, newV1Input); err != nil {
		panic(err)
	}
}

// runAsFilestream checks whether the configuration should be run as
// Filestream input, on any error the boolean value must be ignore and
// no input started. runAsFilestream also sets the input type accordingly.
func runAsFilestream(cfg *config.C) (bool, error) {
	if features.LogInputRunFilestream() {
		return true, nil
	}

	if !management.UnderAgent() {
		return false, nil
	}

	// ID is required to run as Filestream input
	if !cfg.HasField("id") {
		return false, nil
	}

	if ok := cfg.HasField("run_as_filestream"); ok {
		runAsFilestream, err := cfg.Bool("run_as_filestream", -1)
		if err != nil {
			return false, fmt.Errorf("newV1Input: cannot parse 'run_as_filestream': %w", err)
		}

		if runAsFilestream {
			if err := cfg.SetString("type", -1, "filestream"); err != nil {
				return false, fmt.Errorf("cannot set 'type': %w", err)
			}

			return true, nil
		}
	}

	return false, nil
}

// newV1Input creates a new log input
func newV1Input(
	cfg *config.C,
	outlet channel.Connector,
	context v1.Context,
	logger *logp.Logger,
) (v1.Input, error) {
	// Inputs V2 should be tried last, so if this function is run we are
	// supposed to be running as the Log input. However not to rely on the
	// factory implementation, also check whether to run as Log or Filestream
	// inputs.
	asFilestream, err := runAsFilestream(cfg)
	if err != nil {
		return nil, err
	}

	if asFilestream {
		return nil, v2.ErrUnknownInput
	}

	inp, err := loginput.NewInput(cfg, outlet, context, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create log input: %w", err)
	}

	logger.Debug("Log input running as Log input")
	return inp, err
}

// PluginV2 proxies the call to filestream's Plugin function
func PluginV2(logger *logp.Logger, store statestore.States) v2.Plugin {
	// The InputManager for Filestream input is from an internal package, so we
	// cannot instantiate it directly here. To circumvent that, we instantiate
	// the whole Filestream Plugin
	filestreamPlugin := filestream.Plugin(logger, store)

	m := manager{
		next:   filestreamPlugin.Manager,
		logger: logger,
	}
	filestreamPlugin.Manager = m

	p := v2.Plugin{
		Name:      pluginName,
		Stability: feature.Stable,
		Info:      "log input running filestream",
		Doc:       "Log input running Filestream input",
		Manager:   m,
	}
	return p
}

type manager struct {
	next   v2.InputManager
	logger *logp.Logger
}

func (m manager) Init(grp unison.Group) error {
	return m.next.Init(grp)
}

func (m manager) Create(cfg *config.C) (v2.Input, error) {
	// When inputs are created, inputs V2 are tried first, so if we
	// are supposed to run as the Log input, return v2.ErrUnknownInput
	asFilestream, err := runAsFilestream(cfg)
	if err != nil {
		return nil, err
	}

	if asFilestream {
		newCfg, err := translateCfg(cfg)
		if err != nil {
			return nil, fmt.Errorf("cannot translate log config to filestream: %s", err)
		}

		m.logger.Debug("Log input running as Filestream input")
		return m.next.Create(newCfg)
	}

	return nil, v2.ErrUnknownInput
}

var logInputExclusiveKeys = []string{
	"recursive_glob.enabled",
	"harvester_buffer_size",
	"max_bytes",
	"close_inactive",
	"close_renamed",
	"close_removed",
	"close_eof",
	"close_timeout",
	"scan_frequency",
	"scan", // not supported
	"symlinks",
	"max_backoff",
	"backoff_factor", // not supported
	"exclude_files",
	"json",
	"multiline",
	"tail_files",
}

type configType int

const (
	Unknwon configType = iota
	NotSupported
	ConfTypeBool
	ConfTypeInt
	ConfTypeFloat
	ConfTypeStringArray // Could this be just array?
	ConfTypeString
	ConfTypeMap
	ConfTypeConstant
	ConfTypeConvString
)

type configField struct {
	fsName string
	fsVal  string
	kind   configType
}

var convTable = map[string]configField{
	"paths":                  {fsName: "paths", kind: ConfTypeStringArray},
	"recursive_glob.enabled": {fsName: "prospector.scanner.recursive_glob", kind: ConfTypeBool},
	"encoding":               {fsName: "encoding", kind: ConfTypeString},
	"harvester_buffer_size":  {fsName: "buffer_size", kind: ConfTypeInt},
	"max_bytes":              {fsName: "message_max_bytes", kind: ConfTypeInt},
	"ignore_older":           {fsName: "ignore_older", kind: ConfTypeString},
	"close_inactive":         {fsName: "close.on_state_change.inactive", kind: ConfTypeString},
	"close_renamed":          {fsName: "close.on_state_change.renamed", kind: ConfTypeBool},
	"close_removed":          {fsName: "close.on_state_change.removed", kind: ConfTypeBool},
	"close_eof":              {fsName: "close.reader.on_eof", kind: ConfTypeBool},
	"close_timeout":          {fsName: "close.reader.after_interval", kind: ConfTypeString},
	"clean_inactive":         {fsName: "clean_inactive", kind: ConfTypeString},
	"clean_removed":          {fsName: "clean_removed", kind: ConfTypeBool},
	"scan_frequency":         {fsName: "prospector.scanner.check_interval", kind: ConfTypeString},
	"symlinks":               {fsName: "prospector.scanner.symlinks", kind: ConfTypeBool},
	"backoff":                {fsName: "backoff.init", kind: ConfTypeString},
	"max_backoff":            {fsName: "backoff.max", kind: ConfTypeString},
	"harvester_limit":        {fsName: "harvester_limit", kind: ConfTypeInt},
	"file_identity":          {fsName: "file_identity", kind: ConfTypeMap},
	"exclude_lines":          {fsName: "exclude_lines", kind: ConfTypeStringArray},
	"include_lines":          {fsName: "include_lines", kind: ConfTypeStringArray},
	"exclude_files":          {fsName: "prospector.scanner.exclude_files", kind: ConfTypeStringArray},
	"tail_files":             {fsName: "ignore_inactive", fsVal: "since_last_start", kind: ConfTypeConstant},
	// "scan.sort":              "NOT SUPPORTED",
	// "scan.order":             "NOT SUPPORTED",
	// "backoff_factor": {fsName: "NOT SUPPORTED", kind: NotSupported},
}

func translateCfg(cfg *config.C) (*config.C, error) {
	newCfg := config.NewConfig()

	// Convert all the "static" configuration, those are the fields that
	// can easily be translated by name
	for key, kind := range convTable {
		has, err := cfg.Has(key, -1)
		if err != nil {
			return nil, fmt.Errorf("cannot read %q: %w", key, err)
		}

		if has {
			switch kind.kind {
			case ConfTypeString:
				v, err := cfg.String(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as string: %s", key, err)
				}
				newCfg.SetString(kind.fsName, -1, v)

			case ConfTypeBool:
				v, err := cfg.Bool(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as boolean: %w", key, err)
				}
				newCfg.SetBool(key, -1, v)

			case ConfTypeInt:
				v, err := cfg.Int(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as integer: %w", key, err)
				}
				newCfg.SetInt(kind.fsName, -1, v)

			case ConfTypeFloat:
				v, err := cfg.Float(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as float: %w", key, err)
				}
				newCfg.SetFloat(kind.fsName, -1, v)

			case ConfTypeMap:
				child, err := cfg.Child(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as map: %w", key, err)
				}
				newCfg.SetChild(kind.fsName, -1, child)

			case ConfTypeStringArray:
				child, err := cfg.Child(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as map: %w", key, err)
				}
				newCfg.SetChild(kind.fsName, -1, child)

			case ConfTypeConstant:
				v, err := cfg.Bool(key, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot read %q as boolean: %w", key, err)
				}
				if v {
					newCfg.SetString(kind.fsName, -1, kind.fsVal)
				}
			}
		}
	}

	// Now handle the trick bits, starting with parsers
	// The first parser is JSON, then Multiline

	hasMultiline, err := cfg.Has("json", -1)
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
				case ConfTypeString, ConfTypeConvString:
					v, err := multilineChild.String(key, -1)
					if err != nil {
						return nil, fmt.Errorf("cannot read %q as string: %s", key, err)
					}
					multilineCfg.SetString(kind.fsName, -1, v)

				case ConfTypeBool:
					v, err := multilineChild.Bool(key, -1)
					if err != nil {
						return nil, fmt.Errorf("cannot read %q as boolean: %w", key, err)
					}
					multilineCfg.SetBool(key, -1, v)

				case ConfTypeInt:
					v, err := multilineChild.Int(key, -1)
					if err != nil {
						return nil, fmt.Errorf("cannot read %q as integer: %w", key, err)
					}
					multilineCfg.SetInt(kind.fsName, -1, v)
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
			return nil, fmt.Errorf("cannot get 'json': %w", err)
		}

		parsers = append(parsers, map[string]any{"ndjson": jsonCfg})
	}

	// If any parsers was created, set it into the new config
	if len(parsers) != 0 {
		parsersCfg, err := config.NewConfigFrom(parsers)
		if err != nil {
			return nil, fmt.Errorf("cannot convert 'json' config to parser: %w", err)
		}
		if err := newCfg.SetChild("parsers", -1, parsersCfg); err != nil {
			return nil, fmt.Errorf("cannot set parsers: %w", err)
		}

		// TODO: handle existing parsers
	}

	return newCfg, nil
}

var multilineConvTable = map[string]configField{
	"type":          {fsName: "type", kind: ConfTypeConvString},
	"negate":        {fsName: "negate", kind: ConfTypeBool},
	"match":         {fsName: "match", kind: ConfTypeString},
	"max_lines":     {fsName: "max_lines", kind: ConfTypeInt},
	"pattern":       {fsName: "pattern", kind: ConfTypeString},
	"timeout":       {fsName: "timeout", kind: ConfTypeString},
	"flush_pattern": {fsName: "flush_pattern", kind: ConfTypeString},
	"count_lines":   {fsName: "count_lines", kind: ConfTypeInt},
	"skip_newline":  {fsName: "skip_newline", kind: ConfTypeBool},
}
