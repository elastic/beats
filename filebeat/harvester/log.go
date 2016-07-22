package harvester

import (
	"errors"
	"expvar"
	"io"
	"os"

	"golang.org/x/text/transform"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/processor"
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

// Log harvester reads files line by line and sends events to the defined output
func (h *Harvester) Harvest() {

	harvesterStarted.Add(1)
	harvesterRunning.Add(1)
	defer harvesterRunning.Add(-1)

	// Makes sure file is properly closed when the harvester is stopped
	defer h.close()

	h.state.Finished = false

	err := h.open()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected file opening error: %s", err)
		return
	}

	logp.Info("Harvester started for file: %s", h.state.Source)

	processor, err := h.newLineProcessor()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected encoding line reader error: %s", err)
		return
	}

	// Always report the state before starting a harvester
	// This is useful in case the file was renamed
	if !h.sendStateUpdate() {
		return
	}

	for {
		select {
		case <-h.done:
			return
		default:
		}

		// Partial lines return error and are only read on completion
		ts, text, bytesRead, jsonFields, err := readLine(processor)
		if err != nil {
			switch err {
			case reader.ErrFileTruncate:
				logp.Info("File was truncated. Begin reading file from offset 0: %s", h.state.Source)
				h.state.Offset = 0
				filesTruncated.Add(1)
			case reader.ErrRemoved:
				logp.Info("File was removed: %s. Closing because close_removed is enabled.", h.state.Source)
			case reader.ErrRenamed:
				logp.Info("File was renamed: %s. Closing because close_renamed is enabled.", h.state.Source)
			case io.EOF:
				logp.Info("End of file reached: %s. Closing because close_eof is enabled.", h.state.Source)
			default:
				logp.Info("Read line error: %s", err)
			}
			return
		}

		// Update offset if complete line has been processed
		h.state.Offset += int64(bytesRead)

		event := h.createEvent()

		if h.shouldExportLine(text) {
			event.ReadTime = ts
			event.Bytes = bytesRead
			event.Text = &text
			event.JSONFields = jsonFields
		}

		// Always send event to update state, also if lines was skipped
		// Stop harvester in case of an error
		if !h.sendEvent(event) {
			return
		}
	}
}

// createEvent creates and empty event.
// By default the offset is set to 0, means no bytes read. This can be used to report the status
// of a harvester
func (h *Harvester) createEvent() *input.FileEvent {

	event := &input.FileEvent{
		EventMetadata: h.config.EventMetadata,
		Source:        h.state.Source,
		InputType:     h.config.InputType,
		DocumentType:  h.config.DocumentType,
		Offset:        h.state.Offset,
		Bytes:         0,
		Fileinfo:      h.state.Fileinfo,
		JSONConfig:    h.config.JSON,
		State:         h.getState(),
	}

	return event
}

// sendEvent sends event to the spooler channel
// Return false if event was not sent
func (h *Harvester) sendEvent(event *input.FileEvent) bool {
	select {
	case <-h.done:
		return false
	case h.prospectorChan <- event: // ship the new event downstream
		return true
	}
}

// shouldExportLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (h *Harvester) shouldExportLine(line string) bool {
	if len(h.config.IncludeLines) > 0 {
		if !MatchAnyRegexps(h.config.IncludeLines, line) {
			// drop line
			logp.Debug("harvester", "Drop line as it does not match any of the include patterns %s", line)
			return false
		}
	}
	if len(h.config.ExcludeLines) > 0 {
		if MatchAnyRegexps(h.config.ExcludeLines, line) {
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
		logp.Err("Failed opening %s: %s", h.state.Source, err)
		return err
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
	// Check we are not following a rabbit hole (symlinks, etc.)
	if !file.IsRegular(f) {
		return errors.New("Given file is not a regular file.")
	}

	info, err := f.Stat()
	if err != nil {
		logp.Err("Failed getting stats for file %s: %s", h.state.Source, err)
		return err
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

	// tail file if file is new and tail_files config is set
	if h.config.TailFiles {
		logp.Debug("harvester", "Setting offset for tailing file: %s.", h.state.Source)
		return file.Seek(0, os.SEEK_END)
	}

	// get offset from file in case of encoding factory was required to read some data.
	logp.Debug("harvester", "Setting offset for file based on seek: %s", h.state.Source)
	return file.Seek(0, os.SEEK_CUR)
}

// sendStateUpdate send an empty event with the current state to update the registry
func (h *Harvester) sendStateUpdate() bool {
	logp.Debug("harvester", "Update state: %s, offset: %v", h.state.Source, h.state.Offset)
	event := input.NewEvent(h.getState())
	return h.sendEvent(event)
}

func (h *Harvester) getState() file.State {

	if h.config.InputType == config.StdinInputType {
		return file.State{}
	}

	// refreshes the values in State with the values from the harvester itself
	h.state.FileStateOS = file.GetOSState(h.state.Fileinfo)
	return h.state
}

func (h *Harvester) close() {
	// Mark harvester as finished
	h.state.Finished = true

	logp.Debug("harvester", "Stopping harvester for file: %s", h.state.Source)

	// Make sure file is closed as soon as harvester exits
	// If file was never opened, it can't be closed
	if h.file != nil {

		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		// Only send offset if file object was created successfully
		h.sendStateUpdate()

		h.file.Close()
		logp.Debug("harvester", "Stopping harvester, closing file: %s", h.state.Source)
		harvesterOpenFiles.Add(-1)
	} else {
		logp.Warn("Stopping harvester, NOT closing file as file info not available: %s", h.state.Source)
	}

	harvesterClosed.Add(1)
}

func (h *Harvester) newLogFileReaderConfig() reader.LogFileReaderConfig {
	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	return reader.LogFileReaderConfig{
		CloseRemoved:  h.config.CloseRemoved,
		CloseRenamed:  h.config.CloseRenamed,
		CloseInactive: h.config.CloseInactive,
		CloseEOF:      h.config.CloseEOF,
		Backoff:       h.config.Backoff,
		MaxBackoff:    h.config.MaxBackoff,
		BackoffFactor: h.config.BackoffFactor,
	}
}

func (h *Harvester) newLineProcessor() (processor.LineProcessor, error) {

	readerConfig := h.newLogFileReaderConfig()

	var p processor.LineProcessor
	var err error

	fileReader, err := reader.NewLogFileReader(h.file, readerConfig, h.done)
	if err != nil {
		return nil, err
	}

	p, err = processor.NewLineEncoder(fileReader, h.encoding, h.config.BufferSize)
	if err != nil {
		return nil, err
	}

	if h.config.JSON != nil {
		p = processor.NewJSONProcessor(p, h.config.JSON)
	}

	p = processor.NewStripNewline(p)
	if h.config.Multiline != nil {
		p, err = processor.NewMultiline(p, "\n", h.config.MaxBytes, h.config.Multiline)
		if err != nil {
			return nil, err
		}
	}

	return processor.NewLimitProcessor(p, h.config.MaxBytes), nil
}
