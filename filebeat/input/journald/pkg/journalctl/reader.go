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

package journalctl

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
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
var ErrRestarting = errors.New("restarting journalctl")

// JournalEntry holds all fields of a journal entry plus cursor and timestamps
type JournalEntry struct {
	Fields             map[string]any
	Cursor             string
	RealtimeTimestamp  uint64
	MonotonicTimestamp uint64
}

// JctlFactory is a function that returns an instance of journalctl ready to use.
// It exists to allow testing
type JctlFactory func(canceller input.Canceler, logger *logp.Logger, args ...string) (Jctl, error)

// Jctl abstracts the call to journalctl, it exists only for testing purposes
//
//go:generate moq --fmt gofmt -out jctlmock_test.go . Jctl
type Jctl interface {
	// Next returns the next journal entry. If there is no entry available
	// next will block until there is an entry or cancel is cancelled.
	//
	// If cancel is cancelled, Next returns a zero value JournalEntry
	// and ErrCancelled.
	Next(input.Canceler) (data []byte, err error)
	Kill() error
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
	// logger is the logger for the reader
	logger *logp.Logger

	// jctlLogger is the logger for the code controlling
	// the journalctl process
	jctlLogger *logp.Logger

	// args are arguments for journalctl that never change,
	// like the message filters, format, etc
	args []string

	// extraArgs are the arguments used in the first call to
	// journalctl that will be replaced by the cursor argument
	// once data has been ingested
	extraArgs []string

	// cursor is the jornalctl cursor, it is also stored in Filebeat's registry
	cursor string

	canceler input.Canceler

	jctl        Jctl
	jctlFactory JctlFactory

	backoff backoff.Backoff
}

// handleSeekAndCursor returns the correct arguments for seek and cursor.
// If there is a cursor, only the cursor is used, seek is ignored.
// If there is no cursor, then seek is used
// The bool parameter indicates whether there might be messages from
// the previous boots
func handleSeekAndCursor(mode SeekMode, since time.Duration, cursor string) []string {
	if cursor != "" {
		return []string{"--after-cursor", cursor, "--boot", "all"}
	}

	switch mode {
	case SeekSince:
		return []string{
			"--since", time.Now().Add(since).Format(sinceTimeFormat),
			"--boot", "all",
		}
	case SeekTail:
		return []string{"--since", "now"}
	case SeekHead:
		return []string{"--no-tail", "--boot", "all"}
	default:
		// That should never happen
		return []string{}
	}
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
	facilities []int,
	mode SeekMode,
	cursor string,
	since time.Duration,
	file string,
	merge bool,
	newJctl JctlFactory,
) (*Reader, error) {

	logger = logger.Named("reader")

	args := []string{"--utc", "--output=json", "--no-pager", "--all", "--follow"}

	if file != "" && file != localSystemJournalID {

		st, err := os.Stat(file)
		if err != nil {
			logger.Debugf("cannot stat file: %s. Using '--file %s'", err, file)
			args = append(args, "--file", file)
		} else {
			if st.IsDir() {
				args = append(args, "--directory", file)
			} else {
				args = append(args, "--file", file)
			}
		}
	}

	if merge {
		args = append(args, "--merge")
	}

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

	for _, facility := range facilities {
		args = append(args, "--facility", fmt.Sprintf("%d", facility))
	}

	extraArgs := handleSeekAndCursor(mode, since, cursor)

	r := Reader{
		logger:      logger,
		jctlLogger:  logger.Named("journalctl-runner"),
		args:        args,
		extraArgs:   extraArgs,
		cursor:      cursor,
		canceler:    canceler,
		jctlFactory: newJctl,
		backoff:     backoff.NewExpBackoff(canceler.Done(), 100*time.Millisecond, 2*time.Second),
	}

	if err := r.newJctl(extraArgs...); err != nil {
		return &Reader{}, err
	}

	return &r, nil
}

func (r *Reader) newJctl(extraArgs ...string) error {
	args := append(r.args, extraArgs...)

	jctl, err := r.jctlFactory(r.canceler, r.jctlLogger, args...)
	r.jctl = jctl

	return err
}

// Close stops the `journalctl` process and waits for all
// goroutines to return, the canceller passed to `New` should
// be cancelled before `Close` is called
func (r *Reader) Close() error {
	r.logger.Infof("shutting down journalctl, waiting up to: %s", time.Minute)

	if err := r.jctl.Kill(); err != nil {
		return fmt.Errorf("error stopping journalctl: %w", err)
	}

	return nil
}

// next reads the next entry from journalctl. It handles any errors from
// journalctl restarting it as necessary with a backoff strategy. It either
// returns a valid journald entry or ErrCancelled when the input is cancelled.
func (r *Reader) next(cancel input.Canceler) ([]byte, error) {
	msg, err := r.jctl.Next(cancel)

	// Check if the input has been cancelled
	if cancel.Err() != nil {
		// The caller is responsible for calling Reader.Close to terminate
		// journalctl. Cancelling this canceller only means this Next call was
		// cancelled. Because the input has been cancelled, we ignore the message
		// and any error it might have returned.
		return nil, ErrCancelled
	}

	// Two options:
	//   - No error, return the message
	//   - Error, journalctl exited with an error, restart with
	//     backoff if necessary.
	if err == nil {
		return msg, nil
	}
	r.logger.Warnf("reader error: '%s', restarting...", err)

	// Handle backoff
	//
	// If the last restart (if any) was more than 5s ago,
	// recreate the backoff and do not wait.
	// We recreate the backoff so r.backoff.Last().IsZero()
	// will return true next time it's called making us to
	// wait in case jouranlctl crashes in less than 5s.
	if !r.backoff.Last().IsZero() && time.Since(r.backoff.Last()) > 5*time.Second {
		r.backoff = backoff.NewExpBackoff(cancel.Done(), 100*time.Millisecond, 2*time.Second)
	} else {
		r.backoff.Wait()
	}

	var extraArgs []string
	// Corner case: journalctl exited with an error before reading the
	// 1st message. This means we don't have a cursor and need to restart
	// it with the initial arguments.
	if r.cursor == "" {
		extraArgs = r.extraArgs
	} else {
		// We have a cursor, set it instead of the other options that select
		// where in the journal to start reading because they are incompatible
		// with setting the cursor.
		extraArgs = []string{"--after-cursor", r.cursor}
	}

	if err := r.newJctl(extraArgs...); err != nil {
		// If we cannot restart journalct, there is nothing we can do.
		return nil, fmt.Errorf("cannot restart journalctl: %w", err)
	}

	// Return an empty message and wait for the caller to call us again
	return nil, ErrRestarting
}

// Next returns the next journal entry. If there is no entry available
// next will block until there is an entry or cancel is cancelled.
//
// If cancel is cancelled, Next returns a zero value JournalEntry
// and ErrCancelled.
func (r *Reader) Next(cancel input.Canceler) (JournalEntry, error) {
	// r.next returns ErrRestarting when journalctl is restarting,
	// this happens in two situations:
	//  - When the reader first starts, it runs journalctl without the follow
	//    flat to read messages from all previous boots, journalctl exits once
	//    all messages are read.
	//  - journalctl exited unexpectedly and was restarted.
	// On both cases Readr.Next must block until we have a valid journal entry
	// or the input is cancelled.
	for {
		msg, err := r.next(cancel)
		if err != nil {
			if errors.Is(err, ErrRestarting) {
				continue
			}
			return JournalEntry{}, err
		}

		return r.handleMessage(msg)
	}
}

func (r *Reader) handleMessage(msg []byte) (JournalEntry, error) {
	fields := map[string]any{}
	if err := json.Unmarshal(msg, &fields); err != nil {
		r.logger.Error("journal event cannot be parsed as map[string]any, " +
			"look at the events log file for the raw journal event")

		// Log raw data to events log file
		msg := fmt.Sprintf("data cannot be parsed as map[string]any. Data: '%s'",
			string(msg))
		r.logger.Errorw(
			msg,
			"error.message", err.Error(),
			logp.TypeKey, logp.EventType)

		return JournalEntry{}, fmt.Errorf("cannot decode Journald JSON: %w", err)
	}

	ts, isString := fields["__REALTIME_TIMESTAMP"].(string)
	if !isString {
		return JournalEntry{},
			fmt.Errorf("'__REALTIME_TIMESTAMP': '%[1]v', type %[1]T is not a string",
				fields["__REALTIME_TIMESTAMP"])
	}
	unixTS, err := strconv.ParseUint(ts, 10, 64)
	if err != nil {
		return JournalEntry{},
			fmt.Errorf("could not convert '__REALTIME_TIMESTAMP' to uint64: %w",
				err)
	}

	monotomicTs, isString := fields["__MONOTONIC_TIMESTAMP"].(string)
	if !isString {
		return JournalEntry{},
			fmt.Errorf("'__MONOTONIC_TIMESTAMP': '%[1]v', type %[1]T is not a string",
				fields["__MONOTONIC_TIMESTAMP"])
	}
	monotonicTSInt, err := strconv.ParseUint(monotomicTs, 10, 64)
	if err != nil {
		return JournalEntry{},
			fmt.Errorf("could not convert '__MONOTONIC_TIMESTAMP' to uint64: %w",
				err)
	}

	cursor, isString := fields["__CURSOR"].(string)
	if !isString {
		return JournalEntry{},
			fmt.Errorf("'_CURSOR': '%[1]v', type %[1]T is not a string",
				fields["_CURSOR"])
	}

	// Update our cursor so we can restart journalctl if needed
	r.cursor = cursor

	return JournalEntry{
		Fields:             fields,
		RealtimeTimestamp:  unixTS,
		Cursor:             cursor,
		MonotonicTimestamp: monotonicTSInt,
	}, nil
}
