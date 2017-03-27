package harvester

import (
	"bytes"
	"errors"
	"expvar"
	"io"
	"os"
	"time"

	"golang.org/x/text/transform"

	"fmt"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	harvesterStarted   = expvar.NewInt("filebeat.harvester.started")
	harvesterClosed    = expvar.NewInt("filebeat.harvester.closed")
	harvesterRunning   = expvar.NewInt("filebeat.harvester.running")
	harvesterOpenFiles = expvar.NewInt("filebeat.harvester.open_files")
	filesTruncated     = expvar.NewInt("filebeat.harvester.files.truncated")
)

// Setup opens the file handler and creates the reader for the harvester
func (h *Harvester) Setup() (reader.Reader, error) {
	err := h.open()
	if err != nil {
		return nil, fmt.Errorf("Harvester setup failed. Unexpected file opening error: %s", err)
	}

	r, err := h.newLogFileReader()
	if err != nil {
		if h.file != nil {
			h.file.Close()
		}
		return nil, fmt.Errorf("Harvester setup failed. Unexpected encoding line reader error: %s", err)
	}

	return r, nil

}

// Harvest reads files line by line and sends events to the defined output
func (h *Harvester) Harvest(r reader.Reader) {

	harvesterStarted.Add(1)
	harvesterRunning.Add(1)

	h.stopWg.Add(1)
	defer func() {
		// Channel to stop internal harvester routines
		h.stop()
		// Makes sure file is properly closed when the harvester is stopped
		h.close()

		harvesterRunning.Add(-1)

		// Marks harvester stopping completed
		h.stopWg.Done()
	}()

	// Closes reader after timeout or when done channel is closed
	// This routine is also responsible to properly stop the reader
	go func() {

		closeTimeout := make(<-chan time.Time)
		// starts close_timeout timer
		if h.config.CloseTimeout > 0 {
			closeTimeout = time.After(h.config.CloseTimeout)
		}

		select {
		// Applies when timeout is reached
		case <-closeTimeout:
			logp.Info("Closing harvester because close_timeout was reached.")
		// Required when reader loop returns and reader finished
		case <-h.done:
		}

		h.stop()
		h.fileReader.Close()
	}()

	logp.Info("Harvester started for file: %s", h.state.Source)

	for {
		select {
		case <-h.done:
			return
		default:
		}

		message, err := r.Next()
		if err != nil {
			switch err {
			case ErrFileTruncate:
				logp.Info("File was truncated. Begin reading file from offset 0: %s", h.state.Source)
				h.state.Offset = 0
				filesTruncated.Add(1)
			case ErrRemoved:
				logp.Info("File was removed: %s. Closing because close_removed is enabled.", h.state.Source)
			case ErrRenamed:
				logp.Info("File was renamed: %s. Closing because close_renamed is enabled.", h.state.Source)
			case ErrClosed:
				logp.Info("Reader was closed: %s. Closing.", h.state.Source)
			case io.EOF:
				logp.Info("End of file reached: %s. Closing because close_eof is enabled.", h.state.Source)
			case ErrInactive:
				logp.Info("File is inactive: %s. Closing because close_inactive of %v reached.", h.state.Source, h.config.CloseInactive)
			default:
				logp.Err("Read line error: %s; File: ", err, h.state.Source)
			}
			return
		}

		// Strip UTF-8 BOM if beginning of file
		// As all BOMS are converted to UTF-8 it is enough to only remove this one
		if h.state.Offset == 0 {
			message.Content = bytes.Trim(message.Content, "\xef\xbb\xbf")
		}

		// Update offset
		h.state.Offset += int64(message.Bytes)

		state := h.getState()

		// Create state event
		event := input.NewEvent(state)
		text := string(message.Content)

		// Check if data should be added to event. Only export non empty events.
		if !message.IsEmpty() && h.shouldExportLine(text) {
			event.ReadTime = message.Ts
			event.Bytes = message.Bytes
			event.Text = &text
			event.EventMetadata = h.config.EventMetadata
			event.Data = message.Fields
			event.InputType = h.config.InputType
			event.DocumentType = h.config.DocumentType
			event.JSONConfig = h.config.JSON
			event.Pipeline = h.config.Pipeline
			event.Module = h.config.Module
			event.Fileset = h.config.Fileset
		}

		// Always send event to update state, also if lines was skipped
		// Stop harvester in case of an error
		if !h.sendEvent(event) {
			return
		}
		// Update state of harvester as successfully sent
		h.state = state
	}
}

// stop is intended for internal use and closed the done channel to stop execution
func (h *Harvester) stop() {
	h.stopOnce.Do(func() {
		close(h.done)
	})
}

// Stop stops harvester and waits for completion
func (h *Harvester) Stop() {
	h.stop()
	h.stopWg.Wait()
}

// sendEvent sends event to the spooler channel
// Return false if event was not sent
func (h *Harvester) sendEvent(event *input.Event) bool {
	return h.outlet.OnEventSignal(event)
}

// sendStateUpdate send an empty event with the current state to update the registry
// close_timeout does not apply here to make sure a harvester is closed properly. In
// case the output is blocked the harvester will stay open to make sure no new harvester
// is started. As soon as the output becomes available again, the finished state is written
// and processing can continue.
func (h *Harvester) sendStateUpdate() {
	logp.Debug("harvester", "Update state: %s, offset: %v", h.state.Source, h.state.Offset)
	event := input.NewEvent(h.state)
	h.outlet.OnEvent(event)
}

// shouldExportLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (h *Harvester) shouldExportLine(line string) bool {
	if len(h.config.IncludeLines) > 0 {
		if !MatchAny(h.config.IncludeLines, line) {
			// drop line
			logp.Debug("harvester", "Drop line as it does not match any of the include patterns %s", line)
			return false
		}
	}
	if len(h.config.ExcludeLines) > 0 {
		if MatchAny(h.config.ExcludeLines, line) {
			// drop line
			logp.Debug("harvester", "Drop line as it does match one of the exclude patterns%s", line)
			return false
		}
	}

	return true

}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned
func (h *Harvester) openFile() error {

	f, err := file.ReadOpen(h.state.Source)
	if err != nil {
		return fmt.Errorf("Failed opening %s: %s", h.state.Source, err)
	}

	harvesterOpenFiles.Add(1)

	// Makes sure file handler is also closed on errors
	err = h.validateFile(f)
	if err != nil {
		f.Close()
		harvesterOpenFiles.Add(-1)
		return err
	}

	h.file = source.File{f}
	return nil
}

func (h *Harvester) validateFile(f *os.File) error {

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("Failed getting stats for file %s: %s", h.state.Source, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("Tried to open non regular file: %q %s", info.Mode(), info.Name())
	}

	// Compares the stat of the opened file to the state given by the prospector. Abort if not match.
	if !os.SameFile(h.state.Fileinfo, info) {
		return errors.New("File info is not identical with opened file. Aborting harvesting and retrying file later again.")
	}

	h.encoding, err = h.encodingFactory(f)
	if err != nil {

		if err == transform.ErrShortSrc {
			logp.Info("Initialising encoding for '%v' failed due to file being too short", f)
		} else {
			logp.Err("Initialising encoding for '%v' failed: %v", f, err)
		}
		return err
	}

	// get file offset. Only update offset if no error
	offset, err := h.initFileOffset(f)
	if err != nil {
		return err
	}

	logp.Debug("harvester", "Setting offset for file: %s. Offset: %d ", h.state.Source, offset)
	h.state.Offset = offset

	return nil
}

func (h *Harvester) initFileOffset(file *os.File) (int64, error) {

	// continue from last known offset
	if h.state.Offset > 0 {
		logp.Debug("harvester", "Set previous offset for file: %s. Offset: %d ", h.state.Source, h.state.Offset)
		return file.Seek(h.state.Offset, os.SEEK_SET)
	}

	// get offset from file in case of encoding factory was required to read some data.
	logp.Debug("harvester", "Setting offset for file based on seek: %s", h.state.Source)
	return file.Seek(0, os.SEEK_CUR)
}

// getState returns an updated copy of the harvester state
func (h *Harvester) getState() file.State {

	if h.config.InputType == config.StdinInputType {
		return file.State{}
	}

	state := h.state

	// refreshes the values in State with the values from the harvester itself
	state.FileStateOS = file.GetOSState(h.state.Fileinfo)
	return state
}

func (h *Harvester) close() {

	// Mark harvester as finished
	h.state.Finished = true

	logp.Debug("harvester", "Stopping harvester for file: %s", h.state.Source)

	// Make sure file is closed as soon as harvester exits
	// If file was never opened, it can't be closed
	if h.file != nil {

		// close file handler
		h.file.Close()

		logp.Debug("harvester", "Closing file: %s", h.state.Source)
		harvesterOpenFiles.Add(-1)

		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		// Only send offset if file object was created successfully
		h.sendStateUpdate()
	} else {
		logp.Warn("Stopping harvester, NOT closing file as file info not available: %s", h.state.Source)
	}

	harvesterClosed.Add(1)
}

// newLogFileReader creates a new reader to read log files
//
// It creates a chain of readers which looks as following:
//
//   limit -> (multiline -> timeout) -> strip_newline -> json -> encode -> line -> log_file
//
// Each reader on the left, contains the reader on the right and calls `Next()` to fetch more data.
// At the base of all readers the the log_file reader. That means in the data is flowing in the opposite direction:
//
//   log_file -> line -> encode -> json -> strip_newline -> (timeout -> multiline) -> limit
//
// log_file implements io.Reader interface and encode reader is an adapter for io.Reader to
// reader.Reader also handling file encodings. All other readers implement reader.Reader
func (h *Harvester) newLogFileReader() (reader.Reader, error) {

	var r reader.Reader
	var err error

	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	h.fileReader, err = NewLogFile(h.file, h.config)
	if err != nil {
		return nil, err
	}

	r, err = reader.NewEncode(h.fileReader, h.encoding, h.config.BufferSize)
	if err != nil {
		return nil, err
	}

	if h.config.JSON != nil {
		r = reader.NewJSON(r, h.config.JSON)
	}

	r = reader.NewStripNewline(r)

	if h.config.Multiline != nil {
		r, err = reader.NewMultiline(r, "\n", h.config.MaxBytes, h.config.Multiline)
		if err != nil {
			return nil, err
		}
	}

	return reader.NewLimit(r, h.config.MaxBytes), nil
}
