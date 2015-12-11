package harvester

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
)

func NewHarvester(
	prospectorCfg config.ProspectorConfig,
	cfg *config.HarvesterConfig,
	path string,
	signal chan int64,
	spooler chan *input.FileEvent,
) (*Harvester, error) {
	encoding, ok := findEncoding(cfg.Encoding)
	if !ok || encoding == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", cfg.Encoding)
	}

	h := &Harvester{
		Path:             path,
		ProspectorConfig: prospectorCfg,
		Config:           cfg,
		FinishChan:       signal,
		SpoolerChan:      spooler,
		encoding:         encoding,
		backoff:          prospectorCfg.Harvester.BackoffDuration,
	}
	return h, nil
}

// Log harvester reads files line by line and sends events to the defined output
func (h *Harvester) Harvest() {

	err := h.open()

	defer func() {
		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		h.FinishChan <- h.Offset
		// Make sure file is closed as soon as harvester exits
		h.file.Close()
	}()

	if err != nil {
		logp.Err("Stop Harvesting. Unexpected Error: %s", err)
		return
	}

	info, err := h.file.Stat()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected Error: %s", err)
		return
	}

	logp.Info("Harvester started for file: %s", h.Path)

	// Load last offset from registrar
	h.initOffset()

	// TODO: newLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	timedIn := newTimedReader(h.file)
	reader, err := newLineReader(timedIn, h.encoding, h.Config.BufferSize)
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected Error: %s", err)
		return
	}

	// XXX: lastReadTime handling last time a full line was read only?
	//      timedReader provides timestamp some bytes have actually been read from file
	lastReadTime := time.Now()

	// remember size of last partial line being sent. Do not publish partial line, if
	// no new bytes have been processed
	lastPartialLen := 0

	for {
		text, bytesRead, isPartial, err := readLine(reader, &timedIn.lastReadTime, h.Config.PartialLineWaitingDuration)

		if err != nil {

			// In case of err = io.EOF returns nil
			err = h.handleReadlineError(lastReadTime, err)

			if err != nil {
				logp.Err("File reading error. Stopping harvester. Error: %s", err)
				return
			}

			continue
		}

		lastReadTime = time.Now()

		// Reset Backoff
		h.backoff = h.Config.BackoffDuration

		if isPartial {
			if bytesRead <= lastPartialLen {
				// drop partial line event, as no new bytes have been consumed from
				// input stream
				continue
			}

			lastPartialLen = bytesRead
		} else {
			lastPartialLen = 0
		}

		// Sends text to spooler
		event := &input.FileEvent{
			ReadTime:     lastReadTime,
			Source:       &h.Path,
			InputType:    h.Config.InputType,
			DocumentType: h.Config.DocumentType,
			Offset:       h.Offset,
			Bytes:        bytesRead,
			Text:         &text,
			Fields:       &h.Config.Fields,
			Fileinfo:     &info,
			IsPartial:    isPartial,
		}
		if !isPartial {
			h.Offset += int64(bytesRead) // Update offset if complete line has been processed
		}

		event.SetFieldsUnderRoot(h.Config.FieldsUnderRoot)
		h.SpoolerChan <- event // ship the new event downstream
	}
}

// backOff checks the backoff variable and sleeps for the given time
// It also recalculate and sets the next backoff duration
func (h *Harvester) backOff() {
	// Wait before trying to read file which reached EOF again
	time.Sleep(h.backoff)

	// Increment backoff up to maxBackoff
	if h.backoff < h.Config.MaxBackoffDuration {
		h.backoff = h.backoff * time.Duration(h.Config.BackoffFactor)
		if h.backoff > h.Config.MaxBackoffDuration {
			h.backoff = h.Config.MaxBackoffDuration
		}
	}
}

// initOffset finds the current offset of the file and sets it in the harvester as position
func (h *Harvester) initOffset() {
	// get current offset in file
	offset, _ := h.file.Seek(0, os.SEEK_CUR)

	if h.Offset > 0 {
		logp.Debug("harvester", "harvest: %q position:%d (offset snapshot:%d)", h.Path, h.Offset, offset)
	} else if h.Config.TailFiles {
		logp.Debug("harvester", "harvest: (tailing) %q (offset snapshot:%d)", h.Path, offset)
	} else {
		logp.Debug("harvester", "harvest: %q (offset snapshot:%d)", h.Path, offset)
	}

	h.Offset = offset
}

// Sets the offset of the file to the right place. Takes configuration options into account
func (h *Harvester) setFileOffset() {
	if h.Offset > 0 {
		h.file.Seek(h.Offset, os.SEEK_SET)
	} else if h.Config.TailFiles {
		h.file.Seek(0, os.SEEK_END)
	} else {
		h.file.Seek(0, os.SEEK_SET)
	}
}

// open does open the file given under h.Path and assigns the file handler to h.file
func (h *Harvester) open() error {
	// Special handling that "-" means to read from standard input
	if h.Path == "-" {
		h.file = os.Stdin
		return nil
	}

	for {
		var err error
		h.file, err = input.ReadOpen(h.Path)

		if err != nil {
			// TODO: This is currently end endless retry, should be set to a max?
			// retry on failure.
			logp.Err("Failed opening %s: %s", h.Path, err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	file := &input.File{
		File: h.file,
	}

	// Check we are not following a rabbit hole (symlinks, etc.)
	if !file.IsRegularFile() {
		return errors.New("Given file is not a regular file.")
	}

	h.setFileOffset()

	return nil
}

// handleReadlineError handles error which are raised during reading file.
//
// If error is EOF, it will check for:
// * File truncated
// * Older then ignore_older
// * General file error
//
// If none of the above cases match, no error will be returned and file is kept open
//
// In case of a general error, the error itself is returned
func (h *Harvester) handleReadlineError(lastTimeRead time.Time, err error) error {

	if err == io.EOF {
		// Refetch fileinfo to check if the file was truncated or disappeared
		info, statErr := h.file.Stat()

		// This could happen if the file was removed / rotate after reading and before calling the stat function
		if statErr != nil {
			logp.Err("Unexpected error reading from %s; error: %s", h.Path, statErr)
			return statErr
		}

		// Check if file was truncated
		if info.Size() < h.Offset {
			logp.Debug("harvester", "File was truncated as offset (%s) > size (%s). Begin reading file from offset 0: %s", h.Offset, info.Size(), h.Path)
			h.Offset = 0
			h.file.Seek(h.Offset, os.SEEK_SET)
		} else if age := time.Since(lastTimeRead); age > h.ProspectorConfig.IgnoreOlderDuration {
			// If the file hasn't change for longer the ignore_older, harvester stops and file handle will be closed.
			return fmt.Errorf("Stop harvesting as file is older then ignore_older: %s; Last change was: %s ", h.Path, age)
		}

		if h.Config.ForceCloseFiles {

			// Check if the file name exists (see #93)
			_, statErr := os.Stat(h.file.Name())

			// Error means file does not exist. If no error, check if same file. If not close as rotated.
			if statErr != nil || !input.IsSameFile(h.file.Name(), info) {
				logp.Info("Force close file: %s; error: %s", h.Path, statErr)

				// Return directly on windows -> file is closing
				return fmt.Errorf("Force closing file: %s", h.Path)
			}
		}

		h.backOff()

		// Do nothing in case it is just EOF, keep reading the file
		return nil
	} else {
		logp.Err("Unexpected state reading from %s; error: %s", h.Path, err)
		return err
	}
}

func (h *Harvester) Stop() {
}

/*** Utility Functions ***/

// isLine checks if the given byte array is a line, means has a line ending \n
func isLine(line []byte) bool {
	if line == nil || len(line) == 0 {
		return false
	}

	if line[len(line)-1] != '\n' {
		return false
	}
	return true
}

// lineEndingChars returns the number of line ending chars the given by array has
// In case of Unix/Linux files, it is -1, in case of Windows mostly -2
func lineEndingChars(line []byte) int {
	if !isLine(line) {
		return 0
	}

	if line[len(line)-1] == '\n' {
		if len(line) > 1 && line[len(line)-2] == '\r' {
			return 2
		}

		return 1
	}
	return 0
}

// readLine reads a full line into buffer and returns it.
// In case of partial lines, readLine waits for a maximum of partialLineWaiting seconds for new segments to arrive.
// This could potentialy be improved / replaced by https://github.com/elastic/beats/libbeat/tree/master/common/streambuf
func readLine(
	reader *lineReader,
	lastReadTime *time.Time,
	partialLineWaiting time.Duration,
) (string, int, bool, error) {
	for {
		line, sz, err := reader.next()
		if err != nil {
			if err == io.EOF {
				return "", 0, false, err
			}
		}

		if sz != 0 {
			return readlineString(line, sz, false)
		}

		// test for no file updates longer than partialLineWaiting
		if time.Since(*lastReadTime) >= partialLineWaiting {
			// return all bytes read for current line to be processed.
			// Line might grow with further read attempts
			line, sz, err = reader.partial()
			return readlineString(line, sz, true)
		}

		// wait for file updates before reading new lines
		time.Sleep(1 * time.Second)
	}
}

func readlineString(bytes []byte, sz int, partial bool) (string, int, bool, error) {
	s := string(bytes)[:len(bytes)-lineEndingChars(bytes)]
	return s, sz, partial, nil
}
