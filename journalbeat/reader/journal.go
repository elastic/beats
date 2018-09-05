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
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	CURSOR_FILE = ".journalbeat_position"
)

type Config struct {
	Path          string
	MaxBackoff    time.Duration
	Backoff       time.Duration
	BackoffFactor int
}

type Reader struct {
	j       *sdjournal.Journal
	config  Config
	changes chan int
	done    chan struct{}
}

func New(c Config) (*Reader, error) {
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

	seekToSavedPosition(j)

	return &Reader{
		j:       j,
		changes: make(chan int),
		config:  c,
	}, nil
}

func NewLocal(c Config) (*Reader, error) {
	j, err := sdjournal.NewJournal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open local journal")
	}
	seekToSavedPosition(j)

	return &Reader{
		j:       j,
		changes: make(chan int),
		config:  c,
	}, nil
}

func seekToSavedPosition(j *sdjournal.Journal) {
	if _, err := os.Stat(CURSOR_FILE); os.IsNotExist(err) {
		return
	}

	pos, err := ioutil.ReadFile(CURSOR_FILE)
	if err != nil {
		logp.Info("Cannot open cursor file, starting to tail journal")
		j.SeekTail()
		return
	}
	cursor := string(pos[:])

	j.SeekCursor(cursor)
}

func (r *Reader) Follow() <-chan *beat.Event {
	out := make(chan *beat.Event)
	go r.readEntriesFromJournal(out)

	return out
}

func (r *Reader) readEntriesFromJournal(entries chan<- *beat.Event) {
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
			logp.Debug("reader", "End of journal reached; Backoff now.")
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
					logp.Err("Unexpected change: %d", e)
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
		event := getEvent(entry)
		entries <- event

		n, err = r.j.Next()
		if err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

func getEvent(entry *sdjournal.JournalEntry) *beat.Event {
	fields := common.MapStr{}
	for k, v := range entry.Fields {
		key := strings.TrimLeft(strings.ToLower(k), "_")
		fields[key] = v
	}
	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    fields,
	}
	fmt.Println("%s", event.Fields["message"])
	return &event
}

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

	// TODO move current backoff
	if r.config.Backoff < r.config.MaxBackoff {
		r.config.Backoff = r.config.Backoff * time.Duration(r.config.BackoffFactor)
		if r.config.Backoff > r.config.MaxBackoff {
			r.config.Backoff = r.config.MaxBackoff
		}
		logp.Debug("reader", "Increasing backoff time to: %v factor: %v", r.config.Backoff, r.config.BackoffFactor)
	}
}

func (r *Reader) Close() {
	r.savePosition()
	r.j.Close()
}

func (r *Reader) savePosition() {
	c, err := r.j.GetCursor()
	if err != nil {
		logp.Err("Unable to get cursor from journal: %v", err)
	}

	err = ioutil.WriteFile(CURSOR_FILE, []byte(c), 600)
	if err != nil {
		logp.Err("Unable to write cursor to file: %v", err)
	}

}
