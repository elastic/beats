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

//go:build linux && cgo
// +build linux,cgo

package journalread

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/urso/sderr"

	"github.com/elastic/beats/v8/libbeat/common/backoff"
	"github.com/elastic/beats/v8/libbeat/common/cleanup"
	"github.com/elastic/beats/v8/libbeat/logp"
)

// Reader implements a Journald base reader with backoff support. The reader
// will block until a new entry can be read from the journal.
type Reader struct {
	log     *logp.Logger
	backoff backoff.Backoff
	journal journal
}

type canceler interface {
	Done() <-chan struct{}
	Err() error
}

type journal interface {
	Close() error
	Next() (uint64, error)
	Wait(time.Duration) int
	GetEntry() (*sdjournal.JournalEntry, error)
	SeekHead() error
	SeekTail() error
	SeekCursor(string) error
}

// LocalSystemJournalID is the ID of the local system journal.
const localSystemJournalID = "LOCAL_SYSTEM_JOURNAL"

// NewReader creates a new Reader for an already opened journal. The reader assumed to take
// ownership of the journal, and needs to be closed.
func NewReader(log *logp.Logger, journal journal, backoff backoff.Backoff) *Reader {
	return &Reader{log: log, journal: journal, backoff: backoff}
}

// Open opens a journal and creates a reader for it.
// Additonal settings can be applied to the journal by passing functions to with.
// Open returns an error if the journal can not be opened, or if one with-function failed.
//
// Open will opend the systems journal if the path is empty or matches LOCAL_SYSTEM_JOURNAL.
// The path can optionally point to a file or a directory.
func Open(log *logp.Logger, path string, backoff backoff.Backoff, with ...func(j *sdjournal.Journal) error) (*Reader, error) {
	j, err := openJournal(path)
	if err != nil {
		return nil, err
	}

	ok := false
	defer cleanup.IfNot(&ok, func() { j.Close() })

	for _, w := range with {
		if err := w(j); err != nil {
			return nil, err
		}
	}

	ok = true
	return NewReader(log, j, backoff), nil
}

func openJournal(path string) (*sdjournal.Journal, error) {
	if path == localSystemJournalID || path == "" {
		j, err := sdjournal.NewJournal()
		if err != nil {
			err = sderr.Wrap(err, "failed to open local journal")
		}
		return j, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, sderr.Wrap(err, "failed to read meta data for %{path}", path)
	}

	if stat.IsDir() {
		j, err := sdjournal.NewJournalFromDir(path)
		if err != nil {
			err = sderr.Wrap(err, "failed to open journal directory %{path}", path)
		}
		return j, err
	}

	j, err := sdjournal.NewJournalFromFiles(path)
	if err != nil {
		err = sderr.Wrap(err, "failed to open journal file %{path}", path)
	}
	return j, err
}

// Close closes the journal.
func (r *Reader) Close() error {
	return r.journal.Close()
}

// Seek moves the read pointer to a new position.
// If a cursor or SeekTail is given, Seek tries to ignore the entry at the
// given position, jumping right to the next entry.
func (r *Reader) Seek(mode SeekMode, cursor string) (err error) {
	switch mode {
	case SeekHead:
		err = r.journal.SeekHead()
	case SeekTail:
		if err = r.journal.SeekTail(); err == nil {
			_, err = r.journal.Next()
		}
	case SeekCursor:
		if err = r.journal.SeekCursor(cursor); err == nil {
			_, err = r.journal.Next()
		}
	default:
		return fmt.Errorf("invalid seek mode '%v'", mode)
	}
	return err
}

// Next reads a new journald entry from the journal. It blocks if there is
// currently no entry available in the journal, or until an error has occured.
func (r *Reader) Next(cancel canceler) (*sdjournal.JournalEntry, error) {
	for cancel.Err() == nil {
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
				// TODO: backoff support is currently not cancellable :(
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
		r.backoff.Reset()

		return entry, nil
	}
	return nil, cancel.Err()
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

	r.log.Errorf("Unknown return code from Wait: %d\n", c)
	return false, nil
}
