package journalctl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

type journalctl struct {
	cmd      *exec.Cmd
	dataChan chan []byte
	errChan  chan string
	stdout   io.ReadCloser
	stderr   io.ReadCloser

	wg       sync.WaitGroup
	logger   *logp.Logger
	canceler input.Canceler
}

// Factory returns an instance of journalctl ready to use.
//
// The returned type is an interface to allow mocking for testing
func Factory(canceller input.Canceler, logger *logp.Logger, binary string, args ...string) (Jctl, error) {
	cmd := exec.Command(binary, args...)

	jctl := journalctl{
		canceler: canceller,
		cmd:      cmd,
		dataChan: make(chan []byte),
		errChan:  make(chan string),
		logger:   logger,
	}

	var err error
	jctl.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return &journalctl{}, fmt.Errorf("cannot get stdout pipe: %w", err)
	}
	jctl.stderr, err = cmd.StderrPipe()
	if err != nil {
		return &journalctl{}, fmt.Errorf("cannot get stderr pipe: %w", err)
	}

	jctl.wg.Add(1)
	go func() {
		defer jctl.logger.Debug("stderr reader goroutine done")
		defer close(jctl.errChan)
		defer jctl.wg.Done()
		reader := bufio.NewReader(jctl.stderr)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				logger.Errorf("cannot read from journalctl stderr: %s", err)
				return
			}

			jctl.errChan <- fmt.Sprintf("Journalctl wrote to stderr: %s", line)
		}
	}()

	// Goroutine to read events from stdout
	jctl.wg.Add(1)
	go func() {
		defer jctl.logger.Debug("stdout reader goroutine done")
		defer close(jctl.dataChan)
		defer jctl.wg.Done()
		reader := bufio.NewReader(jctl.stdout)
		for {
			data, err := reader.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				logger.Errorf("cannot read from journalctl stdout: %s", err)
				return
			}

			select {
			case <-jctl.canceler.Done():
				return
			case jctl.dataChan <- data:
			}
		}
	}()

	logger.Infof("Journalctl command: journalctl %s", strings.Join(args, " "))

	if err := cmd.Start(); err != nil {
		return &journalctl{}, fmt.Errorf("cannot start journalctl: %w", err)
	}

	logger.Debugf("journalctl started with PID %d", cmd.Process.Pid)

	go func() {
		if err := cmd.Wait(); err != nil {
			jctl.logger.Errorf("journalctl exited with an error, exit code %d ", cmd.ProcessState.ExitCode())
		}
	}()

	return &jctl, nil
}

func (j *journalctl) Kill() error {
	return j.cmd.Process.Kill()
}

func (j *journalctl) Next(cancel input.Canceler) ([]byte, error) {
	select {
	case <-cancel.Done():
		return []byte{}, ErrCancelled
	case d, open := <-j.dataChan:
		if !open {
			return []byte{}, errors.New("no more data to read, journalctl might have exited unexpectedly")
		}
		return d, nil
	}
}

func (j *journalctl) Error() <-chan string {
	return j.errChan
}
