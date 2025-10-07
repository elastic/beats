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
	"github.com/elastic/beats/v7/filebeat/input/log"
	loginput "github.com/elastic/beats/v7/filebeat/input/log"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/reader/readjson"
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

func translateCfg(cfg *config.C) (*config.C, error) {
	fsCfg := filestream.DefaultConfig()
	logCfg := log.DefaultConfig()
	if err := cfg.Unpack(&logCfg); err != nil {
		return nil, fmt.Errorf("cannot unpack log input config: %w", err)
	}

	// The config translation follows the order they appear in the Log input
	// [documentation](https://www.elastic.co/docs/reference/beats/filebeat/filebeat-input-log)
	// and the comments are in the format:
	// log-input-config -> filestream-inpur-config

	// paths -> paths
	fsCfg.Paths = logCfg.Paths

	// recursive_glob.enabled -> prospector.scanner.recursive_glob
	fsCfg.FileWatcher.Scanner.RecursiveGlob = logCfg.RecursiveGlob

	// encoding -> encoding
	fsCfg.Reader.Encoding = logCfg.Encoding

	// harvester_buffer_size -> buffer_size
	fsCfg.Reader.BufferSize = logCfg.BufferSize

	// max_bytes -> message_max_bytes
	fsCfg.Reader.MaxBytes = logCfg.MaxBytes

	// ignore_older -> ignore_older
	fsCfg.IgnoreOlder = logCfg.IgnoreOlder

	// close_inactive -> close.on_state_change.inactive
	fsCfg.Close.OnStateChange.Inactive = logCfg.CloseInactive

	// close_renamed -> close.on_state_change.renamed
	fsCfg.Close.OnStateChange.Renamed = logCfg.CloseRenamed

	// close_removed -> close.on_state_change.removed
	fsCfg.Close.OnStateChange.Removed = logCfg.CloseRemoved

	// close_eof -> close.reader.on_eof
	fsCfg.Close.Reader.OnEOF = logCfg.CloseEOF

	// close_timeout -> close.reader.after_interval
	fsCfg.Close.Reader.AfterInterval = logCfg.CloseTimeout

	// clean_inactive -> clean_inactive
	fsCfg.CleanInactive = logCfg.CleanInactive

	// clean_removed -> clean_removed
	fsCfg.CleanRemoved = logCfg.CleanRemoved

	// scan_frequency -> prospector.scanner.check_interval
	fsCfg.FileWatcher.Interval = logCfg.ScanFrequency

	// scan.sort -> NOT SUPPORTED
	// scan.order -> NOT SUPPORTED

	// symlinks -> prospector.scanner.symlinks
	fsCfg.FileWatcher.Scanner.Symlinks = logCfg.Symlinks

	// backoff ->backoff.init
	fsCfg.Reader.Backoff.Init = logCfg.Backoff
	if _, err := cfg.Remove("backoff", -1); err != nil {
		return nil, fmt.Errorf("cannot remove 'backoff' from source config: %s", err)
	}

	// max_backoff -> backoff.max
	fsCfg.Reader.Backoff.Max = logCfg.MaxBackoff

	// backoff_factor -> NOT SUPPORTED

	// harvester_limit -> harvester_limit
	fsCfg.HarvesterLimit = logCfg.HarvesterLimit

	// file_identity -> file_identity
	fsCfg.FileIdentity = logCfg.FileIdentity

	// ==================================================
	// Undocumented options
	// ==================================================
	fsCfg.Reader.LineTerminator = logCfg.LineTerminator

	// ==================================================
	// Options that cannot be directly set
	// ==================================================
	newCfg := config.MustNewConfigFrom(fsCfg)

	// This comes from the default config, remove if not set in the log input
	if _, err := newCfg.Remove("ignore_inactive", -1); err != nil {
		return nil, fmt.Errorf("cannot remove old ignore_inactive value: %s", err)
	}
	// tail_files -> ignore_inactive: "since_last_start"
	if logCfg.TailFiles {
		if err := newCfg.SetString("ignore_inactive", -1, "since_last_start"); err != nil {
			return nil, fmt.Errorf("cannot set ignore_inactive in new config: %s", err)
		}
	}

	// Same config key and type can be kept
	// exclude_lines -> exclude_lines
	// include_lines -> include_lines

	// exclude_files -> prospector.scanner.exclude_files
	if cfg.HasField("exclude_files") {
		child, err := cfg.Child("exclude_files", -1)
		if err != nil {
			return nil, fmt.Errorf("cannot read 'exclude_files': %s", err)
		}
		newCfg.SetChild("prospector.scanner.exclude_files", -1, child)
	}

	// json -> parsers[0]
	parsers := []any{}
	if logCfg.JSON != nil {
		ndjson := readjson.ParserConfig{
			Config: *logCfg.JSON,
		}

		parsers = append(parsers, map[string]any{
			"ndjson": ndjson,
		})
	}

	// multiline -> parsers[1]
	// map of key -> type
	multilineFields := map[string]string{
		"count_lines":   "int",
		"flush_pattern": "obj", // *match.Matcher
		"match":         "string",
		"max_lines":     "int", // pointer
		"negate":        "bool",
		"pattern":       "string", // *match.Matcher
		"skip_newline":  "bool",
		"timeout":       "obj", // time.duration
		"type":          "string",
	}
	mutilineCfg := config.NewConfig()
	for key, kind := range multilineFields {
		mKey := "multiline." + key

		ok, err := cfg.Has(mKey, -1)
		if err != nil {
			return nil, fmt.Errorf("cannot read %q: %s", mKey, err)
		}

		if ok {
			switch kind {
			case "obj":
				child, err := cfg.Child(mKey, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot get %q: %s", mKey, err)
				}
				mutilineCfg.SetChild(mKey, -1, child)
			case "int":
				child, err := cfg.Int(mKey, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot get %q: %s", mKey, err)
				}
				mutilineCfg.SetInt(mKey, -1, child)
			case "bool":
				child, err := cfg.Bool(mKey, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot get %q: %s", mKey, err)
				}
				mutilineCfg.SetBool(mKey, -1, child)
			case "string":
				child, err := cfg.String(mKey, -1)
				if err != nil {
					return nil, fmt.Errorf("cannot get %q: %s", mKey, err)
				}
				mutilineCfg.SetString(mKey, -1, child)
			}
		}
	}

	parsers = append(parsers, mutilineCfg)
	parsersCfg, err := config.NewConfigFrom(parsers)
	if err != nil {
		return nil, fmt.Errorf("cannot convert 'parsers' to config: %s", err)
	}

	newCfg.SetChild("parsers", -1, parsersCfg)

	if err := newCfg.Merge(cfg); err != nil {
		return nil, fmt.Errorf("cannot merge source and translated config: %s", err)
	}

	// Not documented, remove
	// line_terminator -> line_terminator
	newCfg.Remove("line_terminator", -1)

	// Enable take_over
	if err := newCfg.SetBool("take_over.enabled", -1, true); err != nil {
		return nil, fmt.Errorf("cannot set 'take_over.enabled': %w", err)
	}

	// Remove all keys from the log input
	// TODO (Tiaog): Decide if keep or remove this block
	// for _, key := range logInputExclusiveKeys {
	// 	removed, err := newCfg.Remove(key, -1)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cannot remove '%s': %s", key, err)
	// 	}
	// 	if removed {
	// 		fmt.Printf("========== %q REMOVED\n", key)
	// 	}
	// }

	return newCfg, nil
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
