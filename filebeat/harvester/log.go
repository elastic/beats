package harvester

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/processor"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"golang.org/x/text/transform"
)

const (
	defaultMaxBytes = 10 * (1 << 20) // 10MB
)

func NewHarvester(
	prospectorCfg config.ProspectorConfig,
	cfg *config.HarvesterConfig,
	path string,
	stat *FileStat,
	spooler chan *input.FileEvent,
) (*Harvester, error) {
	var err error
	encoding, ok := encoding.FindEncoding(cfg.Encoding)
	if !ok || encoding == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", cfg.Encoding)
	}

	h := &Harvester{
		Path:             path,
		ProspectorConfig: prospectorCfg,
		Config:           cfg,
		Stat:             stat,
		SpoolerChan:      spooler,
		encoding:         encoding,
	}
	h.ExcludeLinesRegexp, err = InitRegexps(cfg.ExcludeLines)
	if err != nil {
		return h, err
	}
	h.IncludeLinesRegexp, err = InitRegexps(cfg.IncludeLines)
	if err != nil {
		return h, err
	}
	return h, nil
}

func createLineReader(
	in FileSource,
	codec encoding.Encoding,
	bufferSize int,
	maxBytes int,
	readerConfig logFileReaderConfig,
	mlrConfig *config.MultilineConfig,
) (processor.LineProcessor, error) {
	var p processor.LineProcessor
	var err error

	fileReader, err := newLogFileReader(in, readerConfig)
	if err != nil {
		return nil, err
	}

	p, err = processor.NewLineSource(fileReader, codec, bufferSize)
	if err != nil {
		return nil, err
	}

	if mlrConfig != nil {
		p, err = processor.NewMultiline(p, maxBytes, mlrConfig)
		if err != nil {
			return nil, err
		}

		return processor.NewStripNewline(p), nil
	}

	p = processor.NewStripNewline(p)
	return processor.NewLimitProcessor(p, maxBytes), nil
}

// Log harvester reads files line by line and sends events to the defined output
func (h *Harvester) Harvest() {
	defer func() {
		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		if h.Stat != nil {
			h.Stat.Return <- h.Offset
		}

		// Make sure file is closed as soon as harvester exits
		// If file was never properly opened, it can't be closed
		if h.file != nil {
			h.file.Close()
			logp.Debug("harvester", "Closing file: %s", h.Path)
		}
	}()

	enc, err := h.open()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected file opening error: %s", err)
		return
	}

	info, err := h.file.Stat()
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
		closeOlder:         h.ProspectorConfig.CloseOlderDuration,
		backoffDuration:    config.BackoffDuration,
		maxBackoffDuration: config.MaxBackoffDuration,
		backoffFactor:      config.BackoffFactor,
	}

	maxBytes := defaultMaxBytes
	if config.MaxBytes != nil {
		maxBytes = *config.MaxBytes
	}

	reader, err := createLineReader(
		h.file, enc, config.BufferSize, maxBytes, readerConfig, config.Multiline)
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected encoding line reader error: %s", err)
		return
	}

	for {
		// Partial lines return error and are only read on completion
		ts, text, bytesRead, err := readLine(reader)
		if err != nil {
			if err == errFileTruncate {
				seeker, ok := h.file.(io.Seeker)
				if !ok {
					logp.Err("can not seek source")
					return
				}

				logp.Info("File was truncated. Begin reading file from offset 0: %s", h.Path)

				h.Offset = 0
				seeker.Seek(h.Offset, os.SEEK_SET)
				continue
			}

			logp.Info("Read line error: %s", err)
			return
		}

		if h.shouldExportLine(text) {
			// Sends text to spooler
			event := &input.FileEvent{
				ReadTime:     ts,
				Source:       &h.Path,
				InputType:    h.Config.InputType,
				DocumentType: h.Config.DocumentType,
				Offset:       h.Offset,
				Bytes:        bytesRead,
				Text:         &text,
				Fields:       &h.Config.Fields,
				Fileinfo:     &info,
			}

			event.SetFieldsUnderRoot(h.Config.FieldsUnderRoot)
			h.SpoolerChan <- event // ship the new event downstream
		}

		// Set Offset
		h.Offset += int64(bytesRead) // Update offset if complete line has been processed
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

	if h.Offset > 0 {
		// continue from last known offset

		logp.Debug("harvester",
			"harvest: %q position:%d (offset snapshot:%d)", h.Path, h.Offset, offset)
		_, err = file.Seek(h.Offset, os.SEEK_SET)
	} else if h.Config.TailFiles {
		// tail file if file is new and tail_files config is set

		logp.Debug("harvester",
			"harvest: (tailing) %q (offset snapshot:%d)", h.Path, offset)
		h.Offset, err = file.Seek(0, os.SEEK_END)

	} else {
		// get offset from file in case of encoding factory was
		// required to read some data.
		logp.Debug("harvester", "harvest: %q (offset snapshot:%d)", h.Path, offset)
		h.Offset = offset
	}

	return err
}

func (h *Harvester) Stop() {
}

const maxConsecutiveEmptyReads = 100
