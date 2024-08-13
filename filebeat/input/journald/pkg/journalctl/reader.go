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

package journalctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

// LocalSystemJournalID is the ID of the local system journal.
const localSystemJournalID = "LOCAL_SYSTEM_JOURNAL"

// sinceTimeFormat is a time formatting string for the --since flag passed
// to journalctl, it follows a pattern accepted by multiple versions of
// Systemd/Journald.
const sinceTimeFormat = "2006-01-02 15:04:05.999999999"

// ErrCancelled indicates the read was cancelled
var ErrCancelled = errors.New("cancelled")

// JournalEntry holds all fields of a journal entry plus cursor and timestamps
type JournalEntry struct {
	Fields             map[string]any
	Cursor             string
	RealtimeTimestamp  uint64
	MonotonicTimestamp uint64
}

// Reader reads entries from journald by calling `jouranlctl`
// and reading its output.
//
// We call `journalctl` because it proved to be the most resilient way of
// reading journal entries. We have tried to use
// `github.com/coreos/go-systemd/v22/sdjournal`, however due to a bug in
// libsystemd (https://github.com/systemd/systemd/pull/29456) Filebeat
// would crash during journal rotation on high throughput systems.
//
// More details can be found in the PR introducing this feature and related
// issues. PR: https://github.com/elastic/beats/pull/40061.
type Reader struct {
	logger   *logp.Logger
	canceler input.Canceler
	wg       sync.WaitGroup

	jctl *journalctl
}

// handleSeekAndCursor adds the correct arguments for seek and cursor.
// If there is a cursor, only the cursor is used, seek is ignored.
// If there is no cursor, then seek is used
func handleSeekAndCursor(args []string, mode SeekMode, since time.Duration, cursor string) []string {
	if cursor != "" {
		args = append(args, "--after-cursor", cursor)
		return args
	}

	switch mode {
	case SeekSince:
		args = append(args, "--since", time.Now().Add(since).Format(sinceTimeFormat))
	case SeekTail:
		args = append(args, "--since", "now")
	case SeekHead:
		args = append(args, "--no-tail")
	}

	return args
}

// New instantiates and starts a reader for journald logs.
//
// The Reader starts a `journalctl` process with JSON output to read the journal
// entries. Units and syslog identifiers are passed using the corresponding CLI
// flags, matchers are passed directly to `journalctl` then transports are added
// as matchers using `_TRANSPORTS` key.
//
// `mode` defines the 'seek mode'. It indicates whether the journal should be
// read from the tail, head or starting from the cursor. If a cursor is passed,
// then the seek mode is ignored.
//
// To start reading from a relative time, use mode: SeekSince and since should
// be a time.Duration relative to the current time to start reading the
// journald.
//
// File is the journal file to be read, for the system journal use the string
// `LOCAL_SYSTEM_JOURNAL`.
//
// It's the caller's responsibility to call `Close` on the reader to stop
// the `journalctl` process.
//
// If `canceler` is cancelled, the reading goroutine is stopped and subsequent
// calls to `Next` will return an error.
func New(
	logger *logp.Logger,
	canceler input.Canceler,
	units []string,
	syslogIdentifiers []string,
	transports []string,
	matchers journalfield.IncludeMatches,
	mode SeekMode,
	cursor string,
	since time.Duration,
	file string) (*Reader, error) {

	args := []string{"--utc", "--output=json", "--follow"}
	if file != "" && file != localSystemJournalID {
		args = append(args, "--file", file)
	}

	args = handleSeekAndCursor(args, mode, since, cursor)

	for _, u := range units {
		args = append(args, "--unit", u)
	}

	for _, i := range syslogIdentifiers {
		args = append(args, "--identifier", i)
	}

	for _, m := range matchers.Matches {
		args = append(args, m.String())
	}

	for _, m := range transports {
		args = append(args, fmt.Sprintf("_TRANSPORT=%s", m))
	}

	jctl, err := newJournalctl(canceler, logger.Named("journalctl-runner"), "journalctl", args...)
	if err != nil {
		return &Reader{}, err
	}

	r := Reader{
		jctl:     jctl,
		logger:   logger,
		canceler: canceler,
	}

	return &r, nil
}

// Close stops the `journalctl` process and waits for all
// goroutines to return, the canceller passed to `New` should
// be cancelled before `Close` is called
func (r *Reader) Close() error {
	// Try reading stderr until EOF. If there is too much data to read,
	// timeout after 1 minute and proceed to kill the journalctl process.
	readStderrTimeout, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	r.logger.Infof("shutting down journalctl, waiting up to: %s", time.Minute)
ReadErrForLoop:
	for {
		select {
		case <-readStderrTimeout.Done():
			r.logger.Error("timedout while reading stderr from journalctl, the process will be killed")
			break ReadErrForLoop
		case stderrLine, isOpen := <-r.jctl.Error():
			r.logger.Error(stderrLine)
			if !isOpen {
				break ReadErrForLoop
			}
		}
	}

	if err := r.jctl.Kill(); err != nil {
		return fmt.Errorf("cannot stop journalctl: %w", err)
	}

	r.logger.Debug("waiting for all goroutines to finish")
	r.wg.Wait()
	return nil
}

// Next returns the next journal entry. If there is no entry available
// next will block until there is an entry or cancel is cancelled.
//
// If cancel is cancelled, Next returns a zero value JournalEntry
// and ErrCancelled.
func (r *Reader) Next(cancel input.Canceler) (JournalEntry, error) {
	d, err := r.jctl.Next(cancel)
	if err != nil {
		return JournalEntry{}, err
	}

	fields := map[string]any{}
	if err := json.Unmarshal(d, &fields); err != nil {
		r.logger.Error("journal event cannot be parsed as map[string]any, look at the events log file for the raw journal event")
		// Log raw data to events log file
		msg := fmt.Sprintf("data cannot be parsed as map[string]any JSON: '%s'", string(d))
		r.logger.Errorw(msg, logp.TypeKey, logp.EventType, "error.message", err.Error())
		return JournalEntry{}, fmt.Errorf("cannot decode Journald JSON: %w", err)
	}

	ts, isString := fields["__REALTIME_TIMESTAMP"].(string)
	if !isString {
		return JournalEntry{}, fmt.Errorf("'__REALTIME_TIMESTAMP': '%[1]v', type %[1]T is not a string", fields["__REALTIME_TIMESTAMP"])
	}
	unixTS, err := strconv.ParseUint(ts, 10, 64)
	if err != nil {
		return JournalEntry{}, fmt.Errorf("could not convert '__REALTIME_TIMESTAMP' to uint64: %w", err)
	}

	monotomicTs, isString := fields["__MONOTONIC_TIMESTAMP"].(string)
	if !isString {
		return JournalEntry{}, fmt.Errorf("'__MONOTONIC_TIMESTAMP': '%[1]v', type %[1]T is not a string", fields["__MONOTONIC_TIMESTAMP"])
	}
	monotonicTSInt, err := strconv.ParseUint(monotomicTs, 10, 64)
	if err != nil {
		return JournalEntry{}, fmt.Errorf("could not convert '__MONOTONIC_TIMESTAMP' to uint64: %w", err)
	}

	cursor, isString := fields["__CURSOR"].(string)
	if !isString {
		return JournalEntry{}, fmt.Errorf("'_CURSOR': '%[1]v', type %[1]T is not a string", fields["_CURSOR"])
	}

	return JournalEntry{
		Fields:             fields,
		RealtimeTimestamp:  unixTS,
		Cursor:             cursor,
		MonotonicTimestamp: monotonicTSInt,
	}, nil
}
