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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

// LocalSystemJournalID is the ID of the local system journal.
const localSystemJournalID = "LOCAL_SYSTEM_JOURNAL"

// JournalEntry holds all fields of a journal entry plus cursor and timestamps
type JournalEntry struct {
	Fields             map[string]string
	Cursor             string
	RealtimeTimestamp  uint64
	MonotonicTimestamp uint64
}

type Reader struct {
	cmd      *exec.Cmd
	dataChan chan []byte
	errChan  chan error
	logger   *logp.Logger
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	canceler input.Canceler
	wg       sync.WaitGroup
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
		sinceArg := time.Now().Add(since).Format(time.RFC3339Nano)
		args = append(args, "--since", sinceArg)
	case SeekTail:
		args = append(args, "--since", "now")
	case SeekHead:
		args = append(args, "--no-tail")
	}

	return args
}

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

	logger.Debugf("Journalctl command: journalctl %s", strings.Join(args, " "))
	cmd := exec.Command("journalctl", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &Reader{}, fmt.Errorf("cannot get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &Reader{}, fmt.Errorf("cannot get stderr pipe: %w", err)
	}

	r := Reader{
		cmd:      cmd,
		dataChan: make(chan []byte),
		errChan:  make(chan error),
		logger:   logger,
		stdout:   stdout,
		stderr:   stderr,
		canceler: canceler,
	}

	// Goroutine to read errors from stderr
	r.wg.Add(1)
	go func() {
		defer r.logger.Debug("stderr goroutine done")
		defer r.wg.Done()
		reader := bufio.NewReader(r.stderr)
		msgs := []string{}
		for {
			line, err := reader.ReadString('\n')
			if errors.Is(err, io.EOF) {
				if len(msgs) == 0 {
					return
				}
				errMsg := fmt.Sprintf("Journalctl wrote errors: %s", strings.Join(msgs, "\n"))
				logger.Errorf(errMsg)
				r.errChan <- errors.New(errMsg)
				return
			}
			msgs = append(msgs, line)
		}
	}()

	// Goroutine to read events from stdout
	r.wg.Add(1)
	go func() {
		defer r.logger.Debug("stdout goroutine done")
		defer r.wg.Done()
		reader := bufio.NewReader(r.stdout)
		for {
			data, err := reader.ReadBytes('\n')
			if errors.Is(err, io.EOF) {
				close(r.dataChan)
				return
			}

			select {
			case <-r.canceler.Done():
				return
			case r.dataChan <- data:
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return &Reader{}, fmt.Errorf("cannot start journalctl: %w", err)
	}

	return &r, nil
}

func (r *Reader) Close() error {
	if r.cmd == nil {
		return nil
	}

	if err := r.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("cannot stop journalctl: %w", err)
	}

	r.logger.Debug("waiting for all goroutines to finish")
	r.wg.Wait()
	return nil
}

func (r *Reader) Next(input.Canceler) (JournalEntry, error) {
	d, open := <-r.dataChan
	if !open {
		return JournalEntry{}, errors.New("data chan is closed")
	}
	fields := map[string]string{}
	if err := json.Unmarshal(d, &fields); err != nil {
		return JournalEntry{}, fmt.Errorf("cannot decode Journald JSON: %w", err)
	}

	ts := fields["__REALTIME_TIMESTAMP"]
	unixTS, err := strconv.ParseUint(ts, 10, 64)
	if err != nil {
		return JournalEntry{}, fmt.Errorf("could not convert '__REALTIME_TIMESTAMP' to uint64: %w", err)
	}

	monotomicTs := fields["__MONOTONIC_TIMESTAMP"]
	monotonicTSInt, err := strconv.ParseUint(monotomicTs, 10, 64)
	if err != nil {
		return JournalEntry{}, fmt.Errorf("could not convert '__MONOTONIC_TIMESTAMP' to uint64: %w", err)
	}

	cursor := fields["__CURSOR"]

	return JournalEntry{
		Fields:             fields,
		RealtimeTimestamp:  unixTS,
		Cursor:             cursor,
		MonotonicTimestamp: monotonicTSInt,
	}, nil
}
