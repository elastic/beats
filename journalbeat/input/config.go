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

package input

import (
	"time"

	"github.com/elastic/beats/journalbeat/config"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

// Config stores the options of an input.
type Config struct {
	// Paths stores the paths to the journal files to be read.
	Paths []string `config:"paths"`
	// Backoff is the current interval to wait before
	// attemting to read again from the journal.
	Backoff time.Duration `config:"backoff" validate:"min=0,nonzero"`
	// MaxBackoff is the limit of the backoff time.
	MaxBackoff time.Duration `config:"max_backoff" validate:"min=0,nonzero"`
	// Seek is the method to read from journals.
	Seek config.SeekMode `config:"seek"`
	// CursorSeekFallback sets where to seek if registry file is not available.
	CursorSeekFallback config.SeekMode `config:"cursor_seek_fallback"`
	// Matches store the key value pairs to match entries.
	Matches []string `config:"include_matches"`
	// SaveRemoteHostname defines if the original source of the entry needs to be saved.
	SaveRemoteHostname bool `config:"save_remote_hostname"`

	// Fields and tags to add to events.
	common.EventMetadata `config:",inline"`
	// Processors to run on events.
	Processors processors.PluginConfig `config:"processors"`
}

var (
	// DefaultConfig is the defaults for an inputs
	DefaultConfig = Config{
		Backoff:            1 * time.Second,
		MaxBackoff:         20 * time.Second,
		Seek:               config.SeekCursor,
		CursorSeekFallback: config.SeekHead,
		SaveRemoteHostname: false,
	}
)
