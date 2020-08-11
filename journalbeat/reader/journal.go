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

//+build linux,cgo

package reader

import (
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"

	"github.com/elastic/beats/v7/journalbeat/checkpoint"
	"github.com/elastic/beats/v7/journalbeat/cmd/instance"
	"github.com/elastic/beats/v7/journalbeat/pkg/journalfield"
	"github.com/elastic/beats/v7/journalbeat/pkg/journalread"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/ctxtool"
)

// Reader reads entries from journal(s).
type Reader struct {
	r       *journalread.Reader
	journal *sdjournal.Journal
	config  Config
	done    chan struct{}
	logger  *logp.Logger
	backoff backoff.Backoff
}

// New creates a new journal reader and moves the FP to the configured position.
func New(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	return newReader(c.Path, c, done, state, logger)
}

// NewLocal creates a reader to read form the local journal and moves the FP
// to the configured position.
func NewLocal(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	return newReader(LocalSystemJournalID, c, done, state, logger)
}

func newReader(path string, c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	logger = logger.With("path", path)
	backoff := backoff.NewExpBackoff(done, c.Backoff, c.MaxBackoff)

	var journal *sdjournal.Journal
	r, err := journalread.Open(logger, c.Path, backoff, func(j *sdjournal.Journal) error {
		journal = j
		return journalfield.ApplyMatchersOr(j, c.Matches)
	})
	if err != nil {
		return nil, err
	}

	if err := r.Seek(seekBy(logger, c, state)); err != nil {
		logger.Error("Continue from current position. Seek failed with: %v", err)
	}

	logger.Debug("New journal is opened for reading")
	instance.AddJournalToMonitor(c.Path, journal)

	return &Reader{
		r:       r,
		journal: journal,
		config:  c,
		done:    done,
		logger:  logger,
		backoff: backoff,
	}, nil
}

func seekBy(log *logp.Logger, c Config, state checkpoint.JournalState) (journalread.SeekMode, string) {
	mode := c.Seek
	if mode == journalread.SeekCursor && state.Cursor == "" {
		mode = c.CursorSeekFallback
		if mode != journalread.SeekHead && mode != journalread.SeekTail {
			log.Error("Invalid option for cursor_seek_fallback")
			mode = journalread.SeekHead
		}
	}
	return mode, state.Cursor
}

// Close closes the underlying journal reader.
func (r *Reader) Close() {
	instance.StopMonitoringJournal(r.config.Path)
	r.r.Close()
}

// Next waits until a new event shows up and returns it.
// It blocks until an event is returned or an error occurs.
func (r *Reader) Next() (*beat.Event, error) {
	entry, err := r.r.Next(ctxtool.FromChannel(r.done))
	if err != nil {
		return nil, err
	}

	event := toEvent(r.logger, r.config.CheckpointID, entry, r.config.SaveRemoteHostname)
	return event, nil
}

// toEvent creates a beat.Event from journal entries.
func toEvent(logger *logp.Logger, id string, entry *sdjournal.JournalEntry, saveRemoteHostname bool) *beat.Event {
	created := time.Now()
	fields := journalfield.NewConverter(logger, nil).Convert(entry.Fields)
	fields.Put("event.kind", "event")

	// if entry is coming from a remote journal, add_host_metadata overwrites the source hostname, so it
	// has to be copied to a different field
	if saveRemoteHostname {
		remoteHostname, err := fields.GetValue("host.hostname")
		if err == nil {
			fields.Put("log.source.address", remoteHostname)
		}
	}

	state := checkpoint.JournalState{
		Path:               id,
		Cursor:             entry.Cursor,
		RealtimeTimestamp:  entry.RealtimeTimestamp,
		MonotonicTimestamp: entry.MonotonicTimestamp,
	}

	fields.Put("event.created", created)
	receivedByJournal := time.Unix(0, int64(entry.RealtimeTimestamp)*1000)

	event := beat.Event{
		Timestamp: receivedByJournal,
		Fields:    fields,
		Private:   state,
	}
	return &event
}
