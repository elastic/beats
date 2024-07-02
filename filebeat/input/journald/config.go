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

//go:build linux

package journald

import (
	"sync"
	"time"

	"github.com/elastic/go-ucfg"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalctl"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
)

// includeMatchesWarnOnce allow for a config deprecation warning to be
// logged only once if an old config format is detected.
var includeMatchesWarnOnce sync.Once

// Config stores the options of a journald input.
type config struct {
	// Paths stores the paths to the journal files to be read.
	Paths []string `config:"paths"`

	// Since is the relative time offset from now to provide journal
	// entries from.
	Since time.Duration `config:"since"`

	// Seek is the method to read from journals.
	Seek journalctl.SeekMode `config:"seek"`

	// Matches store the key value pairs to match entries.
	Matches bwcIncludeMatches `config:"include_matches"`

	// Units stores the units to monitor.
	Units []string `config:"units"`

	// Transports stores the list of transports to include in the messages.
	Transports []string `config:"transports"`

	// Identifiers stores the syslog identifiers to watch.
	Identifiers []string `config:"syslog_identifiers"`

	// SaveRemoteHostname defines if the original source of the entry needs to be saved.
	SaveRemoteHostname bool `config:"save_remote_hostname"`

	// Parsers configuration
	Parsers parser.Config `config:",inline"`
}

// bwcIncludeMatches is a wrapper that accepts include_matches configuration
// from 7.x to allow old config to remain compatible.
type bwcIncludeMatches journalfield.IncludeMatches

func (im *bwcIncludeMatches) Unpack(c *ucfg.Config) error {
	// Handle 7.x config format in a backwards compatible manner. Old format:
	// include_matches: [_SYSTEMD_UNIT=foo.service, _SYSTEMD_UNIT=bar.service]
	if c.IsArray() {
		var matches []journalfield.Matcher
		if err := c.Unpack(&matches); err != nil {
			return err
		}
		im.Matches = append(im.Matches, matches...)

		includeMatchesWarnOnce.Do(func() {
			cfgwarn.Deprecate("", "Please migrate your journald input's "+
				"include_matches config to the new more expressive format.")
		})
		return nil
	}

	return c.Unpack((*journalfield.IncludeMatches)(im))
}

func defaultConfig() config {
	return config{
		Seek:               journalctl.SeekHead,
		SaveRemoteHostname: false,
	}
}
