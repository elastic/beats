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

package log

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"

	cfg "github.com/elastic/beats/v8/filebeat/config"
	"github.com/elastic/beats/v8/filebeat/harvester"
	"github.com/elastic/beats/v8/filebeat/input/file"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v8/libbeat/common/match"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/reader/multiline"
	"github.com/elastic/beats/v8/libbeat/reader/readfile"
	"github.com/elastic/beats/v8/libbeat/reader/readjson"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	LogConfig                 `config:",inline"`

	// Common
	InputType     string        `config:"input_type"`
	CleanInactive time.Duration `config:"clean_inactive" validate:"min=0"`

	// Input
	Enabled        bool                    `config:"enabled"`
	ExcludeFiles   []match.Matcher         `config:"exclude_files"`
	IgnoreOlder    time.Duration           `config:"ignore_older"`
	Paths          []string                `config:"paths"`
	ScanFrequency  time.Duration           `config:"scan_frequency" validate:"min=0,nonzero"`
	CleanRemoved   bool                    `config:"clean_removed"`
	HarvesterLimit uint32                  `config:"harvester_limit" validate:"min=0"`
	Symlinks       bool                    `config:"symlinks"`
	TailFiles      bool                    `config:"tail_files"`
	RecursiveGlob  bool                    `config:"recursive_glob.enabled"`
	FileIdentity   *common.ConfigNamespace `config:"file_identity"`

	// Harvester
	BufferSize int    `config:"harvester_buffer_size"`
	Encoding   string `config:"encoding"`
	ScanOrder  string `config:"scan.order"`
	ScanSort   string `config:"scan.sort"`

	LineTerminator readfile.LineTerminator `config:"line_terminator"`
	ExcludeLines   []match.Matcher         `config:"exclude_lines"`
	IncludeLines   []match.Matcher         `config:"include_lines"`
	MaxBytes       int                     `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline      *multiline.Config       `config:"multiline"`
	JSON           *readjson.Config        `config:"json"`

	// Hidden on purpose, used by the docker input:
	DockerJSON *struct {
		Stream   string `config:"stream"`
		Partial  bool   `config:"partial"`
		Format   string `config:"format"`
		CRIFlags bool   `config:"cri_flags"`
	} `config:"docker-json"`
}

type LogConfig struct {
	Backoff       time.Duration `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor int           `config:"backoff_factor" validate:"min=1"`
	MaxBackoff    time.Duration `config:"max_backoff" validate:"min=0,nonzero"`
	CloseInactive time.Duration `config:"close_inactive"`
	CloseRemoved  bool          `config:"close_removed"`
	CloseRenamed  bool          `config:"close_renamed"`
	CloseEOF      bool          `config:"close_eof"`
	CloseTimeout  time.Duration `config:"close_timeout" validate:"min=0"`
}

// Contains available scan options
const (
	ScanOrderAsc     = "asc"
	ScanOrderDesc    = "desc"
	ScanSortNone     = ""
	ScanSortModtime  = "modtime"
	ScanSortFilename = "filename"
)

// ValidScanOrder of valid scan orders
var ValidScanOrder = map[string]struct{}{
	ScanOrderAsc:  {},
	ScanOrderDesc: {},
}

// ValidScanOrder of valid scan orders
var ValidScanSort = map[string]struct{}{
	ScanSortNone:     {},
	ScanSortModtime:  {},
	ScanSortFilename: {},
}

func defaultConfig() config {
	return config{
		// Common
		ForwarderConfig: harvester.ForwarderConfig{
			Type: cfg.DefaultType,
		},
		CleanInactive: 0,

		// Input
		Enabled:        true,
		IgnoreOlder:    0,
		ScanFrequency:  10 * time.Second,
		CleanRemoved:   true,
		HarvesterLimit: 0,
		Symlinks:       false,
		TailFiles:      false,
		ScanSort:       "",
		ScanOrder:      "asc",
		RecursiveGlob:  true,
		FileIdentity:   nil,

		// Harvester
		BufferSize:     16 * humanize.KiByte,
		MaxBytes:       10 * humanize.MiByte,
		LineTerminator: readfile.AutoLineTerminator,
		LogConfig: LogConfig{
			Backoff:       1 * time.Second,
			BackoffFactor: 2,
			MaxBackoff:    10 * time.Second,
			CloseInactive: 5 * time.Minute,
			CloseRemoved:  true,
			CloseRenamed:  false,
			CloseEOF:      false,
			CloseTimeout:  0,
		},
	}
}

func (c *config) Validate() error {
	// DEPRECATED 6.0.0: warning is already outputted on input level
	if c.InputType != "" {
		c.Type = c.InputType
	}

	// Input
	if c.Type == harvester.LogType && len(c.Paths) == 0 {
		return fmt.Errorf("No paths were defined for input")
	}

	if c.CleanInactive != 0 && c.IgnoreOlder == 0 {
		return fmt.Errorf("ignore_older must be enabled when clean_inactive is used")
	}

	if c.CleanInactive != 0 && c.CleanInactive <= c.IgnoreOlder+c.ScanFrequency {
		return fmt.Errorf("clean_inactive must be > ignore_older + scan_frequency to make sure only files which are not monitored anymore are removed")
	}

	// Harvester
	if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
		c.Multiline != nil {
		return fmt.Errorf("When using the JSON decoder and multiline together, you need to specify a message_key value")
	}

	if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
		(len(c.IncludeLines) > 0 || len(c.ExcludeLines) > 0) {
		return fmt.Errorf("When using the JSON decoder and line filtering together, you need to specify a message_key value")
	}

	if c.ScanSort != "" {
		cfgwarn.Experimental("scan_sort is used.")

		// Check input type
		if _, ok := ValidScanSort[c.ScanSort]; !ok {
			return fmt.Errorf("Invalid scan sort: %v", c.ScanSort)
		}

		// Check input type
		if _, ok := ValidScanOrder[c.ScanOrder]; !ok {
			return fmt.Errorf("Invalid scan order: %v", c.ScanOrder)
		}
	}

	return nil
}

// resolveRecursiveGlobs expands `**` from the globs in multiple patterns
func (c *config) resolveRecursiveGlobs() error {
	if !c.RecursiveGlob {
		logp.Debug("input", "recursive glob disabled")
		return nil
	}

	logp.Debug("input", "recursive glob enabled")
	var paths []string
	for _, path := range c.Paths {
		patterns, err := file.GlobPatterns(path, recursiveGlobDepth)
		if err != nil {
			return err
		}
		if len(patterns) > 1 {
			logp.Debug("input", "%q expanded to %#v", path, patterns)
		}
		paths = append(paths, patterns...)
	}
	c.Paths = paths
	return nil
}

// normalizeGlobPatterns calls `filepath.Abs` on all the globs from config
func (c *config) normalizeGlobPatterns() error {
	var paths []string
	for _, path := range c.Paths {
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("Failed to get the absolute path for %s: %v", path, err)
		}
		paths = append(paths, pathAbs)
	}
	c.Paths = paths
	return nil
}
