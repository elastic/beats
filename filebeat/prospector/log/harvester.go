// Package log harvests different inputs for new information. Currently
// two harvester types exist:
//
//   * log
//   * stdin
//
//  The log harvester reads a file line by line. In case the end of a file is found
//  with an incomplete line, the line pointer stays at the beginning of the incomplete
//  line. As soon as the line is completed, it is read and returned.
//
//  The stdin harvesters reads data from stdin.
package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/satori/go.uuid"
	"golang.org/x/text/transform"
)

var (
	harvesterMetrics = monitoring.Default.NewRegistry("filebeat.harvester")

	harvesterStarted   = monitoring.NewInt(harvesterMetrics, "started")
	harvesterClosed    = monitoring.NewInt(harvesterMetrics, "closed")
	harvesterRunning   = monitoring.NewInt(harvesterMetrics, "running")
	harvesterOpenFiles = monitoring.NewInt(harvesterMetrics, "open_files")

	ErrFileTruncate = errors.New("detected file being truncated")
	ErrRenamed      = errors.New("file was renamed")
	ErrRemoved      = errors.New("file was removed")
	ErrInactive     = errors.New("file inactive")
	ErrClosed       = errors.New("reader closed")
)

// Harvester contains all harvester related data
type Harvester struct {
	forwarder       *harvester.Forwarder
	config          config
	state           file.State
	states          *file.States
	source          harvester.Source // the source being watched
	log             *Log
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
	done            chan struct{}
	stopOnce        sync.Once
	stopWg          *sync.WaitGroup
	id              uuid.UUID
	reader          reader.Reader
}

// NewHarvester creates a new harvester
func NewHarvester(
	config *common.Config,
	state file.State,
	states *file.States,
	outlet harvester.Outlet,
) (*Harvester, error) {

	h := &Harvester{
		config: defaultConfig,
		state:  state,
		states: states,
		done:   make(chan struct{}),
		stopWg: &sync.WaitGroup{},
		id:     uuid.NewV4(),
	}

	if err := config.Unpack(&h.config); err != nil {
		return nil, err
	}

	encodingFactory, ok := encoding.FindEncoding(h.config.Encoding)
	if !ok || encodingFactory == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", h.config.Encoding)
	}
	h.encodingFactory = encodingFactory

	// Add ttl if clean_inactive is set
	if h.config.CleanInactive > 0 {
		h.state.TTL = h.config.CleanInactive
	}

	// Add outlet signal so harvester can also stop itself
	outlet.SetSignal(h.done)

	var err error
	h.forwarder, err = harvester.NewForwarder(config, outlet)
	if err != nil {
		return nil, err
	}

	return h, nil
}

// open does open the file given under h.Path and assigns the file handler to h.log
func (h *Harvester) open() error {

	switch h.config.Type {
	case harvester.StdinType:
		return h.openStdin()
	case harvester.LogType:
		return h.openFile()
	default:
		return fmt.Errorf("Invalid harvester type: %+v", h.config)
	}
}

// ID returns the unique harvester identifier
func (h *Harvester) ID() uuid.UUID {
	return h.id
}

// Setup opens the file handler and creates the reader for the harvester
func (h *Harvester) Setup() error {
	err := h.open()
	if err != nil {
		return fmt.Errorf("Harvester setup failed. Unexpected file opening error: %s", err)
	}

	h.reader, err = h.newLogFileReader()
	if err != nil {
		if h.source != nil {
			h.source.Close()
		}
		return fmt.Errorf("Harvester setup failed. Unexpected encoding line reader error: %s", err)
	}

	return nil

}

// Run start the harvester and reads files line by line and sends events to the defined output
func (h *Harvester) Run() error {

	// This is to make sure a harvester is not started anymore if stop was already
	// called before the harvester was started. The waitgroup is not incremented afterwards
	// as otherwise it could happend that between checking for the close channel and incrementing
	// the waitgroup, the harvester could be stopped.
	h.stopWg.Add(1)
	select {
	case <-h.done:
		h.stopWg.Done()
		return nil
	default:
	}

	defer func() {
		// Channel to stop internal harvester routines
		h.stop()
		// Makes sure file is properly closed when the harvester is stopped
		h.close()

		harvesterRunning.Add(-1)

		// Marks harvester stopping completed
		h.stopWg.Done()
	}()

	harvesterStarted.Add(1)
	harvesterRunning.Add(1)

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
		h.log.Close()
	}()

	logp.Info("Harvester started for file: %s", h.state.Source)

	for {
		select {
		case <-h.done:
			return nil
		default:
		}

		message, err := h.reader.Next()
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
			return nil
		}

		// Strip UTF-8 BOM if beginning of file
		// As all BOMS are converted to UTF-8 it is enough to only remove this one
		if h.state.Offset == 0 {
			message.Content = bytes.Trim(message.Content, "\xef\xbb\xbf")
		}

		// Get copy of state to work on
		// This is important in case sending is not successful so on shutdown
		// the old offset is reported
		state := h.getState()
		state.Offset += int64(message.Bytes)

		// Create state event
		data := util.NewData()
		if h.source.HasState() {
			data.SetState(state)
		}

		text := string(message.Content)

		// Check if data should be added to event. Only export non empty events.
		if !message.IsEmpty() && h.shouldExportLine(text) {

			data.Event = common.MapStr{
				"@timestamp": common.Time(message.Ts),
				"source":     state.Source,
				"offset":     state.Offset, // Offset here is the offset before the starting char.
			}
			data.Event.DeepUpdate(message.Fields)

			// Check if json fields exist
			var jsonFields common.MapStr
			if fields, ok := data.Event["json"]; ok {
				jsonFields = fields.(common.MapStr)
			}

			if h.config.JSON != nil && len(jsonFields) > 0 {
				reader.MergeJSONFields(data.Event, jsonFields, &text, *h.config.JSON)
			} else if &text != nil {
				if data.Event == nil {
					data.Event = common.MapStr{}
				}
				data.Event["message"] = text
			}
		}

		// Always send event to update state, also if lines was skipped
		// Stop harvester in case of an error
		if !h.sendEvent(data) {
			return nil
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
func (h *Harvester) sendEvent(data *util.Data) bool {
	if h.source.HasState() {
		h.states.Update(data.GetState())
	}

	err := h.forwarder.Send(data)
	return err == nil
}

// SendStateUpdate send an empty event with the current state to update the registry
// close_timeout does not apply here to make sure a harvester is closed properly. In
// case the output is blocked the harvester will stay open to make sure no new harvester
// is started. As soon as the output becomes available again, the finished state is written
// and processing can continue.
func (h *Harvester) SendStateUpdate() {

	if !h.source.HasState() {
		return
	}

	logp.Debug("harvester", "Update state: %s, offset: %v", h.state.Source, h.state.Offset)
	h.states.Update(h.state)

	d := util.NewData()
	d.SetState(h.state)
	h.forwarder.Outlet.OnEvent(d)
}

// shouldExportLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (h *Harvester) shouldExportLine(line string) bool {
	if len(h.config.IncludeLines) > 0 {
		if !harvester.MatchAny(h.config.IncludeLines, line) {
			// drop line
			logp.Debug("harvester", "Drop line as it does not match any of the include patterns %s", line)
			return false
		}
	}
	if len(h.config.ExcludeLines) > 0 {
		if harvester.MatchAny(h.config.ExcludeLines, line) {
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

	h.source = File{File: f}
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
		return errors.New("file info is not identical with opened file. Aborting harvesting and retrying file later again")
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

	if !h.source.HasState() {
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
	if h.source != nil {

		// close file handler
		h.source.Close()

		logp.Debug("harvester", "Closing file: %s", h.state.Source)
		harvesterOpenFiles.Add(-1)

		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		// Only send offset if file object was created successfully
		h.SendStateUpdate()
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
	h.log, err = NewLog(h.source, h.config.LogConfig)
	if err != nil {
		return nil, err
	}

	r, err = reader.NewEncode(h.log, h.encoding, h.config.BufferSize)
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
