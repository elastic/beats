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

package reader

import (
	"time"

	"github.com/elastic/beats/journalbeat/config"
)

// Config stores the options of a reder.
type Config struct {
	// Path is the path to the journal file.
	Path string
	// Seek specifies the seeking stategy.
	// Possible values: head, tail, cursor.
	Seek config.SeekMode
	// CursorSeekFallback sets where to seek if registry file is not available.
	CursorSeekFallback config.SeekMode
	// MaxBackoff is the limit of the backoff time.
	MaxBackoff time.Duration
	// Backoff is the current interval to wait before
	// attemting to read again from the journal.
	Backoff time.Duration
	// Matches store the key value pairs to match entries.
	Matches []string
	// SaveRemoteHostname defines if the original source of the entry needs to be saved.
	SaveRemoteHostname bool
}

const (
	// LocalSystemJournalID is the ID of the local system journal.
	LocalSystemJournalID = "LOCAL_SYSTEM_JOURNAL"
)
