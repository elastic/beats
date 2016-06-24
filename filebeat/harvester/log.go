package harvester

import (
	"errors"
	"os"

	"golang.org/x/text/transform"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/processor"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

// Log harvester reads files line by line and sends events to the defined output
func (h *Harvester) Harvest() {

	// Makes sure file is properly closed when the harvester is stopped
	defer h.close()

	h.State.Finished = false

	enc, err := h.open()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected file opening error: %s", err)
		return
	}

	logp.Info("Harvester started for file: %s", h.Path)

	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	cfg := h.Config
	readerConfig := reader.LogFileReaderConfig{
		ForceClose:         cfg.ForceCloseFiles,
		CloseOlder:         cfg.CloseOlder,
		BackoffDuration:    cfg.Backoff,
		MaxBackoffDuration: cfg.MaxBackoff,
		BackoffFactor:      cfg.BackoffFactor,
	}

	processor, err := createLineProcessor(
		h.file, enc, cfg.BufferSize, cfg.MaxBytes, readerConfig,
		cfg.JSON, cfg.Multiline, h.done)
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected encoding line reader error: %s", err)
		return
	}

	// Always report the state before starting a harvester
	if !h.SendStateUpdate() {
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
			if err == reader.ErrFileTruncate {
				logp.Info("File was truncated. Begin reading file from offset 0: %s", h.Path)
				h.SetOffset(0)
				return
			}

			logp.Info("Read line error: %s", err)
			return
		}

		// Update offset if complete line has been processed
		h.updateOffset(int64(bytesRead))

		event := h.createEvent()

		if h.shouldExportLine(text) {

			event.ReadTime = ts
			event.Bytes = bytesRead
			event.Text = &text
			event.JSONFields = jsonFields
		}

		// Always send event to update state, also if lines was skipped
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
		EventMetadata: h.Config.EventMetadata,
		Source:        h.Path,
		InputType:     h.Config.InputType,
		DocumentType:  h.Config.DocumentType,
		Offset:        h.getOffset(),
		Bytes:         0,
		Fileinfo:      h.State.Fileinfo,
		JSONConfig:    h.Config.JSON,
	}

	if h.Config.InputType != config.StdinInputType {
		event.FileState = h.GetState()
	}
	return event
}

// sendEvent sends event to the spooler channel
// Return false if event was not sent
func (h *Harvester) sendEvent(event *input.FileEvent) bool {
	select {
	case <-h.done:
		return false
	case h.SpoolerChan <- event: // ship the new event downstream
		return true
	}
}

// shouldExportLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (h *Harvester) shouldExportLine(line string) bool {
	if len(h.IncludeLinesRegexp) > 0 {
		if !MatchAnyRegexps(h.IncludeLinesRegexp, line) {
			// drop line
			logp.Debug("harvester", "Drop line as it does not match any of the include patterns %s", line)
			return false
		}
	}
	if len(h.ExcludeLinesRegexp) > 0 {
		if MatchAnyRegexps(h.ExcludeLinesRegexp, line) {
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
func (h *Harvester) openFile() (encoding.Encoding, error) {
	var encoding encoding.Encoding

	f, err := file.ReadOpen(h.Path)
	if err == nil {
		// Check we are not following a rabbit hole (symlinks, etc.)
		if !file.IsRegular(f) {
			return nil, errors.New("Given file is not a regular file.")
		}

		encoding, err = h.encoding(f)
		if err != nil {

			if err == transform.ErrShortSrc {
				logp.Info("Initialising encoding for '%v' failed due to file being too short", f)
			} else {
				logp.Err("Initialising encoding for '%v' failed: %v", f, err)
			}
			return nil, err
		}

	} else {
		logp.Err("Failed opening %s: %s", h.Path, err)
		return nil, err
	}

	// update file offset
	err = h.initFileOffset(f)
	if err != nil {
		return nil, err
	}

	// yay, open file
	h.file = source.File{f}
	return encoding, nil
}

func (h *Harvester) initFileOffset(file *os.File) error {
	offset, err := file.Seek(0, os.SEEK_CUR)

	if h.getOffset() > 0 {
		// continue from last known offset

		logp.Debug("harvester",
			"harvest: %q position:%d (offset snapshot:%d)", h.Path, h.getOffset(), offset)
		_, err = file.Seek(h.getOffset(), os.SEEK_SET)
	} else if h.Config.TailFiles {
		// tail file if file is new and tail_files config is set

		logp.Debug("harvester",
			"harvest: (tailing) %q (offset snapshot:%d)", h.Path, offset)
		offset, err = file.Seek(0, os.SEEK_END)
		h.SetOffset(offset)

	} else {
		// get offset from file in case of encoding factory was
		// required to read some data.
		logp.Debug("harvester", "harvest: %q (offset snapshot:%d)", h.Path, offset)
		h.SetOffset(offset)
	}

	return err
}

func (h *Harvester) SetOffset(offset int64) {
	h.offset = offset
}

func (h *Harvester) getOffset() int64 {
	return h.offset
}

func (h *Harvester) updateOffset(increment int64) {
	h.offset += increment
}

// SendStateUpdate send an empty event with the current state to update the registry
func (h *Harvester) SendStateUpdate() bool {
	logp.Debug("harvester", "Update state: %s, offset: %v", h.Path, h.offset)
	return h.sendEvent(h.createEvent())
}

func (h *Harvester) GetState() file.State {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()

	h.refreshState()
	return h.State
}

// refreshState refreshes the values in State with the values from the harvester itself
func (h *Harvester) refreshState() {
	h.State.Source = h.Path
	h.State.Offset = h.getOffset()
	h.State.FileStateOS = file.GetOSState(h.State.Fileinfo)
}

func (h *Harvester) close() {
	// Mark harvester as finished
	h.State.Finished = true

	// On completion, push offset so we can continue where we left off if we relaunch on the same file
	h.SendStateUpdate()

	logp.Debug("harvester", "Stopping harvester for file: %s", h.Path)

	// Make sure file is closed as soon as harvester exits
	// If file was never properly opened, it can't be closed
	if h.file != nil {
		h.file.Close()
		logp.Debug("harvester", "Stopping harvester, closing file: %s", h.Path)
	} else {
		logp.Warn("harvester", "Stopping harvester, NOT closing file as file info not available: %s", h.Path)
	}
}

func createLineProcessor(
	in source.FileSource,
	codec encoding.Encoding,
	bufferSize int,
	maxBytes int,
	readerConfig reader.LogFileReaderConfig,
	jsonConfig *processor.JSONConfig,
	mlrConfig *processor.MultilineConfig,
	done chan struct{},
) (processor.LineProcessor, error) {
	var p processor.LineProcessor
	var err error

	fileReader, err := reader.NewLogFileReader(in, readerConfig, done)
	if err != nil {
		return nil, err
	}

	p, err = processor.NewLineEncoder(fileReader, codec, bufferSize)
	if err != nil {
		return nil, err
	}

	if jsonConfig != nil {
		p = processor.NewJSONProcessor(p, jsonConfig)
	}

	p = processor.NewStripNewline(p)
	if mlrConfig != nil {
		p, err = processor.NewMultiline(p, "\n", maxBytes, mlrConfig)
		if err != nil {
			return nil, err
		}
	}

	return processor.NewLimitProcessor(p, maxBytes), nil
}
