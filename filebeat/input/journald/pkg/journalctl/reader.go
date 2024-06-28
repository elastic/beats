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

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalread"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

// LocalSystemJournalID is the ID of the local system journal.
const localSystemJournalID = "LOCAL_SYSTEM_JOURNAL"

type Reader struct {
	cmd      *exec.Cmd
	dataChan chan []byte
	errChan  chan error
	logger   *logp.Logger
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	canceler input.Canceler
	wg       sync.WaitGroup

	matchers journalfield.IncludeMatches
}

func New(
	logger *logp.Logger,
	canceler input.Canceler,
	matchers journalfield.IncludeMatches,
	mode journalread.SeekMode,
	cursor string,
	since time.Duration,
	file string) (*Reader, error) {

	// --file opens an specific file
	// If cursor is set, use --after-cursor
	args := []string{"--utc", "--output=json", "--follow"}
	if file != "" && file != localSystemJournalID {
		args = append(args, "--file", file)
	}

	switch mode {
	case journalread.SeekSince:
		sinceArg := time.Now().Add(since).Format(time.RFC3339)
		args = append(args, "--since", sinceArg)
	case journalread.SeekCursor:
		args = append(args, "--after-cursor", cursor)
	case journalread.SeekTail:
		args = append(args, "--since", "now")
	case journalread.SeekHead:
		// Do not append anything
	default:
		return nil, fmt.Errorf("unknown seek mode %v", mode)
	}

	for _, m := range matchers.Matches {
		args = append(args, m.String())
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
			logger.Debug(">>>>> Got data: ", string(data))

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

func (r *Reader) Start()                                                    {}
func (r *Reader) SeekRealtimeUsec(usec uint64) error                        { return nil }
func (r *Reader) Seek(mode journalread.SeekMode, cursor string) (err error) { return nil }

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

func (r *Reader) Next(input.Canceler) (*sdjournal.JournalEntry, error) {
	d, open := <-r.dataChan
	if !open {
		return nil, errors.New("data chan is closed")
	}
	fields := map[string]string{}
	if err := json.Unmarshal(d, &fields); err != nil {
		return nil, fmt.Errorf("cannot decode Journald JSON: %w", err)
	}

	ts := fields["__REALTIME_TIMESTAMP"]
	unixTS, err := strconv.ParseUint(ts, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not convert timestamp to uint64: %w", err)
	}

	monotomicTs := fields["__MONOTONIC_TIMESTAMP"]
	monotonicTSInt, err := strconv.ParseUint(monotomicTs, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not convert monotomic timestamp to uint64: %w", err)
	}

	cursor := fields["__CURSOR"]

	return &sdjournal.JournalEntry{
		Fields:             fields,
		RealtimeTimestamp:  unixTS,
		Cursor:             cursor,
		MonotonicTimestamp: monotonicTSInt,
	}, nil
}
