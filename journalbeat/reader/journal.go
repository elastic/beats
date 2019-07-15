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
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/pkg/errors"

	"github.com/elastic/beats/journalbeat/checkpoint"
	"github.com/elastic/beats/journalbeat/cmd/instance"
	"github.com/elastic/beats/journalbeat/config"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/backoff"
	"github.com/elastic/beats/libbeat/logp"
)

// Reader reads entries from journal(s).
type Reader struct {
	journal *sdjournal.Journal
	config  Config
	done    chan struct{}
	logger  *logp.Logger
	backoff backoff.Backoff
}

// New creates a new journal reader and moves the FP to the configured position.
func New(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	f, err := os.Stat(c.Path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}

	var j *sdjournal.Journal
	if f.IsDir() {
		j, err = sdjournal.NewJournalFromDir(c.Path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open journal directory")
		}
	} else {
		j, err = sdjournal.NewJournalFromFiles(c.Path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open journal file")
		}
	}

	l := logger.With("path", c.Path)
	l.Debug("New journal is opened for reading")

	return newReader(l, done, c, j, state)
}

// NewLocal creates a reader to read form the local journal and moves the FP
// to the configured position.
func NewLocal(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	j, err := sdjournal.NewJournal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open local journal")
	}

	l := logger.With("path", "local")
	l.Debug("New local journal is opened for reading")

	return newReader(l, done, c, j, state)
}

func newReader(logger *logp.Logger, done chan struct{}, c Config, journal *sdjournal.Journal, state checkpoint.JournalState) (*Reader, error) {
	err := setupMatches(journal, c.Matches)
	if err != nil {
		return nil, err
	}

	r := &Reader{
		journal: journal,
		config:  c,
		done:    done,
		logger:  logger,
		backoff: backoff.NewExpBackoff(done, c.Backoff, c.MaxBackoff),
	}
	r.seek(state.Cursor)

	instance.AddJournalToMonitor(c.Path, journal)

	return r, nil
}

func setupMatches(j *sdjournal.Journal, matches []string) error {
	for _, m := range matches {
		elems := strings.Split(m, "=")
		if len(elems) != 2 {
			return fmt.Errorf("invalid match format: %s", m)
		}

		var p string
		for journalKey, eventField := range journaldEventFields {
			if elems[0] == eventField.name {
				p = journalKey + "=" + elems[1]
			}
		}

		// pass custom fields as is
		if p == "" {
			p = m
		}

		logp.Debug("journal", "Added matcher expression: %s", p)

		err := j.AddMatch(p)
		if err != nil {
			return fmt.Errorf("error adding match to journal %v", err)
		}

		err = j.AddDisjunction()
		if err != nil {
			return fmt.Errorf("error adding disjunction to journal: %v", err)
		}
	}
	return nil
}

// seek seeks to the position determined by the coniguration and cursor state.
func (r *Reader) seek(cursor string) {
	switch r.config.Seek {
	case config.SeekCursor:
		if cursor == "" {
			switch r.config.CursorSeekFallback {
			case config.SeekHead:
				r.journal.SeekHead()
				r.logger.Debug("Seeking method set to cursor, but no state is saved for reader. Starting to read from the beginning")
			case config.SeekTail:
				r.journal.SeekTail()
				r.journal.Next()
				r.logger.Debug("Seeking method set to cursor, but no state is saved for reader. Starting to read from the end")
			default:
				r.logger.Error("Invalid option for cursor_seek_fallback")
			}
			return
		}
		r.journal.SeekCursor(cursor)
		_, err := r.journal.Next()
		if err != nil {
			r.logger.Error("Error while seeking to cursor")
		}
		r.logger.Debug("Seeked to position defined in cursor")
	case config.SeekTail:
		r.journal.SeekTail()
		r.journal.Next()
		r.logger.Debug("Tailing the journal file")
	case config.SeekHead:
		r.journal.SeekHead()
		r.logger.Debug("Reading from the beginning of the journal file")
	default:
		r.logger.Error("Invalid seeking mode")
	}
}

// Next waits until a new event shows up and returns it.
// It blocks until an event is returned or an error occurs.
func (r *Reader) Next() (*beat.Event, error) {
	for {
		select {
		case <-r.done:
			return nil, nil
		default:
		}

		c, err := r.journal.Next()
		if err != nil && err != io.EOF {
			return nil, err
		}

		switch {
		// error while reading next entry
		case c < 0:
			return nil, fmt.Errorf("error while reading next entry %+v", syscall.Errno(-c))
		// no new entry, so wait
		case c == 0:
			hasNewEntry, err := r.checkForNewEvents()
			if err != nil {
				return nil, err
			}
			if !hasNewEntry {
				r.backoff.Wait()
			}
			continue
		// new entries are available
		default:
		}

		entry, err := r.journal.GetEntry()
		if err != nil {
			return nil, err
		}
		event := r.toEvent(entry)
		r.backoff.Reset()

		return event, nil
	}
}

func (r *Reader) checkForNewEvents() (bool, error) {
	c := r.journal.Wait(100 * time.Millisecond)
	switch c {
	case sdjournal.SD_JOURNAL_NOP:
		return false, nil
	// new entries are added or the journal has changed (e.g. vacuum, rotate)
	case sdjournal.SD_JOURNAL_APPEND, sdjournal.SD_JOURNAL_INVALIDATE:
		return true, nil
	default:
	}

	r.logger.Errorf("Unknown return code from Wait: %d\n", c)
	return false, nil
}

// toEvent creates a beat.Event from journal entries.
func (r *Reader) toEvent(entry *sdjournal.JournalEntry) *beat.Event {
	fields := common.MapStr{}
	custom := common.MapStr{}

	for entryKey, v := range entry.Fields {
		if fieldConversionInfo, ok := journaldEventFields[entryKey]; !ok {
			normalized := strings.ToLower(strings.TrimLeft(entryKey, "_"))
			custom.Put(normalized, v)
		} else if !fieldConversionInfo.dropped {
			value := r.convertNamedField(fieldConversionInfo, v)
			fields.Put(fieldConversionInfo.name, value)
		}
	}

	if len(custom) != 0 {
		fields.Put("journald.custom", custom)
	}

	// if entry is coming from a remote journal, add_host_metadata overwrites the source hostname, so it
	// has to be copied to a different field
	if r.config.SaveRemoteHostname {
		remoteHostname, err := fields.GetValue("host.hostname")
		if err == nil {
			fields.Put("log.source.address", remoteHostname)
		}
	}

	state := checkpoint.JournalState{
		Path:               r.config.Path,
		Cursor:             entry.Cursor,
		RealtimeTimestamp:  entry.RealtimeTimestamp,
		MonotonicTimestamp: entry.MonotonicTimestamp,
	}

	fields.Put("event.created", time.Now())
	receivedByJournal := time.Unix(0, int64(entry.RealtimeTimestamp)*1000)

	event := beat.Event{
		Timestamp: receivedByJournal,
		Fields:    fields,
		Private:   state,
	}
	return &event
}

func (r *Reader) convertNamedField(fc fieldConversion, value string) interface{} {
	if fc.isInteger {
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			r.logger.Debugf("Failed to convert field: %s \"%v\" to int: %v", fc.name, value, err)
			return value
		}
		return v
	}
	return value
}

// Close closes the underlying journal reader.
func (r *Reader) Close() {
	instance.StopMonitoringJournal(r.config.Path)
	r.journal.Close()
}
