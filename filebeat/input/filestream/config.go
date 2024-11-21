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
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Config stores the options of a file stream.
type config struct {
	Reader readerConfig `config:",inline"`

	ID           string          `config:"id"`
	Paths        []string        `config:"paths"`
	Close        closerConfig    `config:"close"`
	FileWatcher  *conf.Namespace `config:"prospector"`
	FileIdentity *conf.Namespace `config:"file_identity"`

	// -1 means that registry will never be cleaned
	CleanInactive  time.Duration      `config:"clean_inactive" validate:"min=-1"`
	CleanRemoved   bool               `config:"clean_removed"`
	HarvesterLimit uint32             `config:"harvester_limit" validate:"min=0"`
	IgnoreOlder    time.Duration      `config:"ignore_older"`
	IgnoreInactive ignoreInactiveType `config:"ignore_inactive"`
	Rotation       *conf.Namespace    `config:"rotation"`
	TakeOver       bool               `config:"take_over"`
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
		Reader:         defaultReaderConfig(),
		Paths:          []string{},
		Close:          defaultCloserConfig(),
		CleanInactive:  -1,
		CleanRemoved:   true,
		HarvesterLimit: 0,
		IgnoreOlder:    0,
	}
}

func defaultCloserConfig() closerConfig {
	return closerConfig{
		OnStateChange: stateChangeCloserConfig{
			CheckInterval: 5 * time.Second,
			Removed:       true, // TODO check clean_removed option
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

func (c *config) Validate() error {
	if len(c.Paths) == 0 {
		return fmt.Errorf("no path is configured")
	}

	return nil
}

// ValidateInputIDs checks all filestream inputs to ensure all have an ID and
// the ID is unique. If there is any empty or duplicated ID, it logs an error
// containing the offending input configurations and returns an error containing
// the duplicated IDs.
func ValidateInputIDs(inputs []*conf.C, logger *logp.Logger) error {
	ids := make(map[string][]*conf.C)
	var duplicates []string
	var empty []*conf.C
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
			if fsInput.ID == "" {
				empty = append(empty, input)
				continue
			}
			ids[fsInput.ID] = append(ids[fsInput.ID], input)
			if len(ids[fsInput.ID]) > 1 {
				duplicates = append(duplicates, fsInput.ID)
			}
		}
	}

	var errs []string
	if empty != nil {
		errs = append(errs, "input without ID")
	}
	if len(duplicates) != 0 {
		errs = append(errs, fmt.Sprintf("filestream inputs with duplicated IDs: %v", strings.Join(duplicates, ",")))
	}

	if len(errs) != 0 {
		jsonDupCfg := collectOffendingInputs(duplicates, empty, ids)
		logger.Errorw("filestream inputs with invalid IDs", "inputs", jsonDupCfg)

		return fmt.Errorf("filestream inputs validation error: %s", strings.Join(errs, ", "))
	}

	return nil
}

func collectOffendingInputs(duplicates []string, empty []*conf.C, ids map[string][]*conf.C) []map[string]interface{} {
	var cfgs []map[string]interface{}

	if len(empty) > 0 {
		for _, cfg := range empty {
			toJson := map[string]interface{}{}
			err := cfg.Unpack(&toJson)
			if err != nil {
				toJson["emptyID"] = fmt.Sprintf("failes to umpack config: %v", err)
			}
			cfgs = append(cfgs, toJson)
		}
	}

	for _, id := range duplicates {
		for _, dupcfgs := range ids[id] {
			toJson := map[string]interface{}{}
			err := dupcfgs.Unpack(&toJson)
			if err != nil {
				toJson[id] = fmt.Sprintf("failes to umpack config: %v", err)
			}
			cfgs = append(cfgs, toJson)
		}
	}

	return cfgs
}
