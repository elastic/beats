package journalctl

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync/atomic"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalread"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Reader struct {
	cmd      *exec.Cmd
	count    *atomic.Uint64
	dataChan chan []byte
	logger   *logp.Logger
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	canceler input.Canceler

	matchers journalfield.IncludeMatches
}

func New(logger *logp.Logger, canceler input.Canceler, matchers journalfield.IncludeMatches, src string) (Reader, error) {
	// --file opens an specific file
	args := []string{"--output=json", "--follow", "--file", src}
	for _, m := range matchers.Matches {
		args = append(args, m.String())
	}

	fmt.Println(">>>>> Args", args)
	cmd := exec.Command("journalctl", args...)

	logger.Debug("Starting Journalctl reader")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Reader{}, fmt.Errorf("cannot get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Reader{}, fmt.Errorf("cannot get stderr pipe: %w", err)
	}

	r := Reader{
		cmd:      cmd,
		dataChan: make(chan []byte),
		logger:   logger,
		stdout:   stdout,
		stderr:   stderr,
	}

	go func() {
		fmt.Println("Reader goroutine started")
		reader := bufio.NewReader(r.stdout)
		for {
			data, err := reader.ReadBytes('\n')
			if errors.Is(err, io.EOF) {
				close(r.dataChan)
				return
			}
			fmt.Println(">>>>> Got data: ", string(data))
			r.dataChan <- data
		}
	}()

	if err := cmd.Start(); err != nil {
		return Reader{}, fmt.Errorf("cannot start journalctl: %w", err)
	}

	return r, nil
}

func (r Reader) Start() {
	r.logger.Debug("Start called")
}

func (r Reader) Close() error                                              { return nil }
func (r Reader) Seek(mode journalread.SeekMode, cursor string) (err error) { return nil }
func (r Reader) SeekRealtimeUsec(usec uint64) error                        { return nil }
func (r Reader) Next(input.Canceler) (*sdjournal.JournalEntry, error) {
	// TODO: find out why this is not being called any more
	fmt.Println("Next called")
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
