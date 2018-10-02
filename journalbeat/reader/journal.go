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
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/pkg/errors"

	"github.com/elastic/beats/journalbeat/checkpoint"
	"github.com/elastic/beats/journalbeat/cmd/instance"
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
	// Matches store the key value pairs to match entries.
	Matches []string
}

// Reader reads entries from journal(s).
type Reader struct {
	journal *sdjournal.Journal
	config  Config
	changes chan int
	done    chan struct{}
	logger  *logp.Logger
}

// New creates a new journal reader and moves the FP to the configured position.
func New(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	f, err := os.Stat(c.Path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}

	logger = logger.With("path", c.Path)

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

	err = setupMatches(j, c.Matches)
	if err != nil {
		return nil, err
	}

	r := &Reader{
		journal: j,
		changes: make(chan int),
		config:  c,
		done:    done,
		logger:  logger,
	}
	r.seek(state.Cursor)

	instance.AddJournalToMonitor(c.Path, j)

	r.logger.Debug("New journal is opened for reading")

	return r, nil
}

// NewLocal creates a reader to read form the local journal and moves the FP
// to the configured position.
func NewLocal(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	j, err := sdjournal.NewJournal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open local journal")
	}

	c.Path = LocalSystemJournalID
	logger = logger.With("path", "local")
	logger.Debug("New local journal is opened for reading")

	err = setupMatches(j, c.Matches)
	if err != nil {
		return nil, err
	}

	r := &Reader{
		journal: j,
		changes: make(chan int),
		config:  c,
		done:    done,
		logger:  logger,
	}
	r.seek(state.Cursor)

	instance.AddJournalToMonitor(c.Path, j)

	return r, nil
}

func setupMatches(j *sdjournal.Journal, matches []string) error {
	for _, m := range matches {
		elems := strings.Split(m, "=")
		if len(elems) != 2 {
			return fmt.Errorf("invalid match format: %s", m)
		}

		var p string
		for journalKey, eventKey := range journaldEventFields {
			if elems[0] == eventKey {
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
	if r.config.Seek == "cursor" {
		if cursor == "" {
			r.journal.SeekHead()
			r.logger.Debug("Seeking method set to cursor, but no state is saved for reader. Starting to read from the beginning")
			return
		}
		r.journal.SeekCursor(cursor)
		_, err := r.journal.Next()
		if err != nil {
			r.logger.Error("Error while seeking to cursor")
		}
		r.logger.Debug("Seeked to position defined in cursor")
	} else if r.config.Seek == "tail" {
		r.journal.SeekTail()
		r.logger.Debug("Tailing the journal file")
	} else if r.config.Seek == "head" {
		r.journal.SeekHead()
		r.logger.Debug("Reading from the beginning of the journal file")
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
				r.logger.Error("Unexpected error while reading from journal: %v", err)
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
						//r.logger.Error("Unexpected error: %v", syscall.Errno(-e))
					}
					r.wait()
				}
			}
		}
	}
}

func (r *Reader) readUntilNotNull(entries chan<- *beat.Event) error {
	n, err := r.journal.Next()
	if err != nil && err != io.EOF {
		return err
	}

	for n != 0 {
		entry, err := r.journal.GetEntry()
		if err != nil {
			return err
		}
		event := r.toEvent(entry)
		entries <- event

		n, err = r.journal.Next()
		if err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

// toEvent creates a beat.Event from journal entries.
func (r *Reader) toEvent(entry *sdjournal.JournalEntry) *beat.Event {
	fields := common.MapStr{}
	custom := common.MapStr{}

	for k, v := range entry.Fields {
		if kk, ok := journaldEventFields[k]; !ok {
			normalized := strings.ToLower(strings.TrimLeft(k, "_"))
			custom.Put(normalized, v)
		} else {
			if isKept(kk) {
				fields.Put(kk, v)
			}
		}
	}

	if len(custom) != 0 {
		fields["custom"] = custom
	}

	state := checkpoint.JournalState{
		Path:               r.config.Path,
		Cursor:             entry.Cursor,
		RealtimeTimestamp:  entry.RealtimeTimestamp,
		MonotonicTimestamp: entry.MonotonicTimestamp,
	}

	fields["read_timestamp"] = time.Now()
	receivedByJournal := time.Unix(0, int64(entry.RealtimeTimestamp)*1000)

	event := beat.Event{
		Timestamp: receivedByJournal,
		Fields:    fields,
		Private:   state,
	}
	return &event
}

func isKept(key string) bool {
	return key != ""
}

// stopOrWait waits for a journal event.
func (r *Reader) stopOrWait() {
	select {
	case <-r.done:
	case r.changes <- r.journal.Wait(100 * time.Millisecond):
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
		r.logger.Debugf("Increasing backoff time to: %v factor: %v", r.config.Backoff, r.config.BackoffFactor)
	}
}

// Close closes the underlying journal reader.
func (r *Reader) Close() {
	instance.StopMonitoringJournal(r.config.Path)
	r.journal.Close()
}
