package harvester

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"golang.org/x/text/transform"
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
		backoff:          prospectorCfg.Harvester.BackoffDuration,
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

// Log harvester reads files line by line and sends events to the defined output
func (h *Harvester) Harvest() {

	defer func() {
		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		h.Stat.Return <- h.Offset

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
	timedIn := newTimedReader(h.file)
	reader, err := encoding.NewLineReader(timedIn, enc, h.Config.BufferSize)
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected encoding line reader error: %s", err)
		return
	}

	// XXX: lastReadTime handling last time a full line was read only?
	//      timedReader provides timestamp some bytes have actually been read from file
	lastReadTime := time.Now()

	for {
		// Partial lines return error and are only read on completion
		text, bytesRead, err := readLine(reader, &timedIn.lastReadTime)

		if err != nil {

			// In case of err = io.EOF returns nil
			err = h.handleReadlineError(lastReadTime, err)

			// Return in case of error which leads to stopping harvester and closing file
			if err != nil {
				logp.Info("Read line error: %s", err)
				return
			}

			continue
		}

		lastReadTime = time.Now()

		// Reset Backoff
		h.backoff = h.Config.BackoffDuration

		if h.shouldExportLine(text) {

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
	if err != io.EOF || !h.file.Continuable() {
		logp.Err("Unexpected state reading from %s; error: %s", h.Path, err)
		return err
	}

	// Refetch fileinfo to check if the file was truncated or disappeared.
	// Errors if the file was removed/rotated after reading and before
	// calling the stat function
	info, statErr := h.file.Stat()
	if statErr != nil {
		logp.Err("Unexpected error reading from %s; error: %s", h.Path, statErr)
		return statErr
	}

	// Handle fails if file was truncated
	if info.Size() < h.Offset {
		seeker, ok := h.file.(io.Seeker)
		if !ok {
			logp.Err("Can not seek source")
			return err
		}

		logp.Debug("harvester", "File was truncated as offset (%s) > size (%s). Begin reading file from offset 0: %s", h.Offset, info.Size(), h.Path)

		h.Offset = 0
		seeker.Seek(h.Offset, os.SEEK_SET)
		return nil
	}

	age := time.Since(lastTimeRead)
	if age > h.ProspectorConfig.IgnoreOlderDuration {
		// If the file hasn't change for longer the ignore_older, harvester stops
		// and file handle will be closed.
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

	if err != io.EOF {
		logp.Err("Unexpected state reading from %s; error: %s", h.Path, err)
	}

	logp.Debug("harvester", "End of file reached: %s; Backoff now.", h.Path)

	// Do nothing in case it is just EOF, keep reading the file after backing off
	h.backOff()
	return nil
}

func (h *Harvester) Stop() {
}

const maxConsecutiveEmptyReads = 100

// timedReader keeps track of last time bytes have been read from underlying
// reader.
type timedReader struct {
	reader       io.Reader
	lastReadTime time.Time // last time we read some data from input stream
}

func newTimedReader(reader io.Reader) *timedReader {
	r := &timedReader{
		reader: reader,
	}
	return r
}

func (r *timedReader) Read(p []byte) (int, error) {
	var err error
	n := 0

	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err = r.reader.Read(p)
		if n > 0 {
			r.lastReadTime = time.Now()
			break
		}

		if err != nil {
			break
		}
	}

	return n, err
}
