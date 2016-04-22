package harvester

import (
	"errors"
	"io"
	"os"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"golang.org/x/text/transform"
)

// Log harvester reads files line by line and sends events to the defined output
func (h *Harvester) Harvest() {
	defer func() {
		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		if h.Stat != nil {
			h.Stat.Return <- h.GetOffset()
		}

		logp.Debug("harvester", "Stopping harvester for file: %s", h.Path)

		// Make sure file is closed as soon as harvester exits
		// If file was never properly opened, it can't be closed
		if h.file != nil {
			h.file.Close()
			logp.Debug("harvester", "Stopping harvester, closing file: %s", h.Path)
		} else {
			logp.Debug("harvester", "Stopping harvester, NOT closing file as file info not available: %s", h.Path)
		}
	}()

	enc, err := h.open()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected file opening error: %s", err)
		return
	}

	h.fileInfo, err = h.file.Stat()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected file stat rror: %s", err)
		return
	}

	logp.Info("Harvester started for file: %s", h.Path)

	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	config := h.Config
	readerConfig := logFileReaderConfig{
		forceClose:         config.ForceCloseFiles,
		closeOlder:         config.CloseOlderDuration,
		backoffDuration:    config.BackoffDuration,
		maxBackoffDuration: config.MaxBackoffDuration,
		backoffFactor:      config.BackoffFactor,
	}

	reader, err := createLineReader(
		h.file, enc, config.BufferSize, config.MaxBytes, readerConfig,
		config.JSON, config.Multiline)
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected encoding line reader error: %s", err)
		return
	}

	for {
		// Partial lines return error and are only read on completion
		ts, text, bytesRead, jsonFields, err := readLine(reader)
		if err != nil {
			if err == errFileTruncate {
				seeker, ok := h.file.(io.Seeker)
				if !ok {
					logp.Err("can not seek source")
					return
				}

				logp.Info("File was truncated. Begin reading file from offset 0: %s", h.Path)

				h.SetOffset(0)
				seeker.Seek(h.GetOffset(), os.SEEK_SET)
				continue
			}

			logp.Info("Read line error: %s", err)
			return
		}

		// Update offset if complete line has been processed
		h.SetOffset(h.GetOffset() + int64(bytesRead))

		event := h.createEvent()

		if h.shouldExportLine(text) {

			event.ReadTime = ts
			event.Bytes = bytesRead
			event.Text = &text
			event.JSONFields = jsonFields
		}

		// Always send event to update state, also if lines was skipped
		h.sendEvent(event)
	}
}

// createEvent creates and empty event.
// By default the offset is set to 0, means no bytes read. This can be used to report the status
// of a harvester
func (h *Harvester) createEvent() *input.FileEvent {
	return &input.FileEvent{
		EventMetadata: h.Config.EventMetadata,
		Source:        h.Path,
		InputType:     h.Config.InputType,
		DocumentType:  h.Config.DocumentType,
		Offset:        h.GetOffset(),
		Bytes:         0,
		Fileinfo:      &h.fileInfo,
		JSONConfig:    h.Config.JSON,
	}
}

// sendEvent sends event to the spooler channel
func (h *Harvester) sendEvent(event *input.FileEvent) {
	h.SpoolerChan <- event // ship the new event downstream
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

// open does open the file given under h.Path and assigns the file handler to h.file
func (h *Harvester) open() (encoding.Encoding, error) {
	// Special handling that "-" means to read from standard input
	if h.Config.InputType == config.StdinInputType {
		return h.openStdin()
	}
	return h.openFile()
}

func (h *Harvester) openStdin() (encoding.Encoding, error) {
	h.file = pipeSource{os.Stdin}
	return h.encoding(h.file)
}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned
func (h *Harvester) openFile() (encoding.Encoding, error) {
	var file *os.File
	var err error
	var encoding encoding.Encoding

	file, err = input.ReadOpen(h.Path)
	if err == nil {
		// Check we are not following a rabbit hole (symlinks, etc.)
		if !input.IsRegularFile(file) {
			return nil, errors.New("Given file is not a regular file.")
		}

		encoding, err = h.encoding(file)
		if err != nil {

			if err == transform.ErrShortSrc {
				logp.Info("Initialising encoding for '%v' failed due to file being too short", file)
			} else {
				logp.Err("Initialising encoding for '%v' failed: %v", file, err)
			}
			return nil, err
		}

	} else {
		logp.Err("Failed opening %s: %s", h.Path, err)
		return nil, err
	}

	// update file offset
	err = h.initFileOffset(file)
	if err != nil {
		return nil, err
	}

	// yay, open file
	h.file = fileSource{file}
	return encoding, nil
}

func (h *Harvester) initFileOffset(file *os.File) error {
	offset, err := file.Seek(0, os.SEEK_CUR)

	if h.GetOffset() > 0 {
		// continue from last known offset

		logp.Debug("harvester",
			"harvest: %q position:%d (offset snapshot:%d)", h.Path, h.GetOffset(), offset)
		_, err = file.Seek(h.GetOffset(), os.SEEK_SET)
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

// GetState returns current state of harvester
func (h *Harvester) GetState() *input.FileState {

	state := input.FileState{
		Source:      h.Path,
		Offset:      h.GetOffset(),
		FileStateOS: input.GetOSFileState(&h.Stat.Fileinfo),
	}

	return &state
}

func (h *Harvester) SetOffset(offset int64) {
	h.offsetLock.Lock()
	defer h.offsetLock.Unlock()

	h.offset = offset
}

func (h *Harvester) GetOffset() int64 {
	h.offsetLock.Lock()
	defer h.offsetLock.Unlock()

	return h.offset
}
