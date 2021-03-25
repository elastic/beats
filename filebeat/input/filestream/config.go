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
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
)

// Config stores the options of a file stream.
type config struct {
	Reader readerConfig `config:",inline"`

	Paths          []string                `config:"paths"`
	Close          closerConfig            `config:"close"`
	FileWatcher    *common.ConfigNamespace `config:"prospector"`
	FileIdentity   *common.ConfigNamespace `config:"file_identity"`
	CleanInactive  time.Duration           `config:"clean_inactive" validate:"min=0"`
	CleanRemoved   bool                    `config:"clean_removed"`
	HarvesterLimit uint32                  `config:"harvester_limit" validate:"min=0"`
	IgnoreOlder    time.Duration           `config:"ignore_older"`
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

	Parsers []*common.ConfigNamespace `config:"parsers"` // TODO multiline, json, syslog?
}

type backoffConfig struct {
	Init time.Duration `config:"init" validate:"nonzero"`
	Max  time.Duration `config:"max" validate:"nonzero"`
}

func defaultConfig() config {
	return config{
		Reader:         defaultReaderConfig(),
		Paths:          []string{},
		Close:          defaultCloserConfig(),
		CleanInactive:  0,
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
			Inactive:      0 * time.Second,
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
			Init: 1 * time.Second,
			Max:  10 * time.Second,
		},
		BufferSize:     16 * humanize.KiByte,
		LineTerminator: readfile.AutoLineTerminator,
		MaxBytes:       10 * humanize.MiByte,
		Tail:           false,
		Parsers:        nil,
	}
}

func (c *config) Validate() error {
	if len(c.Paths) == 0 {
		return fmt.Errorf("no path is configured")
	}
	// TODO
	//if c.CleanInactive != 0 && c.IgnoreOlder == 0 {
	//	return fmt.Errorf("ignore_older must be enabled when clean_inactive is used")
	//}

	// TODO
	//if c.CleanInactive != 0 && c.CleanInactive <= c.IgnoreOlder+c.ScanFrequency {
	//	return fmt.Errorf("clean_inactive must be > ignore_older + scan_frequency to make sure only files which are not monitored anymore are removed")
	//}

	// TODO
	//if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
	//	c.Multiline != nil {
	//	return fmt.Errorf("When using the JSON decoder and multiline together, you need to specify a message_key value")
	//}

	//if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
	//	(len(c.IncludeLines) > 0 || len(c.ExcludeLines) > 0) {
	//	return fmt.Errorf("When using the JSON decoder and line filtering together, you need to specify a message_key value")
	//}

	return nil
}
