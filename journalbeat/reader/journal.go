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
	"io"
	"os"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/pkg/errors"

	"github.com/elastic/beats/journalbeat/checkpoint"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	// LocalSystemJournalID is the ID of the local system journal.
	LocalSystemJournalID = "LOCAL_SYSTEM_JOURNAL"
)

// Config stores the options of a reder.
type Config struct {
	// Path is the path to the journal file.
	Path string
	// Seek specifies the seeking stategy.
	// Possible values: head, tail, cursor.
	Seek string
	// MaxBackoff is the limit of the backoff time.
	MaxBackoff time.Duration
	// Backoff is the current interval to wait before
	// attemting to read again from the journal.
	Backoff time.Duration
	// BackoffFactor is the multiplier of Backoff.
	BackoffFactor int
}

// Reader reads entries from journal(s).
type Reader struct {
	j       *sdjournal.Journal
	config  Config
	changes chan int
	done    chan struct{}
}

// New creates a new journal reader and moves the FP to the configured position.
func New(c Config, done chan struct{}, state checkpoint.JournalState) (*Reader, error) {
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

	r := &Reader{
		j:       j,
		changes: make(chan int),
		config:  c,
		done:    done,
	}
	r.seek(state.Cursor)

	logp.Debug("reader", "New journal is opened for reading")

	return r, nil
}

// NewLocal creates a reader to read form the local journal and moves the FP
// to the configured position.
func NewLocal(c Config, done chan struct{}, state checkpoint.JournalState) (*Reader, error) {
	j, err := sdjournal.NewJournal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open local journal")
	}

	logp.Debug("reader", "New local journal is opened for reading")

	r := &Reader{
		j:       j,
		changes: make(chan int),
		config:  c,
		done:    done,
	}
	r.seek(state.Cursor)
	return r, nil
}

// seek seeks to the position determined by the coniguration and cursor state.
func (r *Reader) seek(cursor string) {
	if r.config.Seek == "cursor" {
		if cursor == "" {
			r.j.SeekHead()
			logp.Debug("journal", "Seeking method set to cursor, but no state is saved for reader. Starting to read from the beginning")
			return
		}
		r.j.SeekCursor(cursor)
		_, err := r.j.Next()
		if err != nil {
			logp.Err("Error while seeking to cursor")
		}
		logp.Debug("journal", "Seeked to position defined in cursor")
	} else if r.config.Seek == "tail" {
		r.j.SeekTail()
		logp.Debug("journal", "Tailing the journal file")
	} else if r.config.Seek == "head" {
		r.j.SeekHead()
		logp.Debug("journal", "Reading from the beginning of the journal file")
	}
}

// Follow reads entries from journals.
func (r *Reader) Follow() chan *beat.Event {
	out := make(chan *beat.Event)
	go r.readEntriesFromJournal(out)

	return out
}

func (r *Reader) readEntriesFromJournal(entries chan *beat.Event) {
	defer close(entries)

process:
	for {
		select {
		case <-r.done:
			return
		default:
			err := r.readUntilNotNull(entries)
			if err != nil {
				logp.Err("Unexpected error while reading from journal: %v", err)
			}
		}

		for {
			go r.stopOrWait()

			select {
			case <-r.done:
				return
			case e := <-r.changes:
				switch e {
				case sdjournal.SD_JOURNAL_NOP:
					r.wait()
				case sdjournal.SD_JOURNAL_APPEND, sdjournal.SD_JOURNAL_INVALIDATE:
					continue process
				default:
					if e < 0 {
						//logp.Err("Unexpected error: %v", syscall.Errno(-e))
					}
					r.wait()
				}
			}
		}
	}
}

func (r *Reader) readUntilNotNull(entries chan<- *beat.Event) error {
	n, err := r.j.Next()
	if err != nil && err != io.EOF {
		return err
	}

	for n != 0 {
		entry, err := r.j.GetEntry()
		if err != nil {
			return err
		}
		event := r.toEvent(entry)
		entries <- event

		n, err = r.j.Next()
		if err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

// toEvent creates a beat.Event from journal entries.
func (r *Reader) toEvent(entry *sdjournal.JournalEntry) *beat.Event {
	fields := common.MapStr{}
	for journalKey, eventKey := range journaldEventFields {
		if entry.Fields[journalKey] != "" {
			fields.Put(eventKey, entry.Fields[journalKey])
		}
	}

	state := checkpoint.JournalState{
		Path:               r.config.Path,
		Cursor:             entry.Cursor,
		RealtimeTimestamp:  entry.RealtimeTimestamp,
		MonotonicTimestamp: entry.MonotonicTimestamp,
	}

	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    fields,
		Private:   state,
	}
	return &event
}

// stopOrWait waits for a journal event.
func (r *Reader) stopOrWait() {
	select {
	case <-r.done:
	case r.changes <- r.j.Wait(100 * time.Millisecond):
	}
}

func (r *Reader) wait() {
	select {
	case <-r.done:
		return
	case <-time.After(r.config.Backoff):
	}

	if r.config.Backoff < r.config.MaxBackoff {
		r.config.Backoff = r.config.Backoff * time.Duration(r.config.BackoffFactor)
		if r.config.Backoff > r.config.MaxBackoff {
			r.config.Backoff = r.config.MaxBackoff
		}
		logp.Debug("reader", "Increasing backoff time to: %v factor: %v", r.config.Backoff, r.config.BackoffFactor)
	}
}

// Close closes the underlying journal reader.
func (r *Reader) Close() {
	r.j.Close()
}
