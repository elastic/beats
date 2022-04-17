// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/text/transform"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	file_helper "github.com/menderesk/beats/v7/libbeat/common/file"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/monitoring"

	"github.com/menderesk/beats/v7/filebeat/channel"
	"github.com/menderesk/beats/v7/filebeat/harvester"
	"github.com/menderesk/beats/v7/filebeat/input/file"
	"github.com/menderesk/beats/v7/libbeat/reader"
	"github.com/menderesk/beats/v7/libbeat/reader/debug"
	"github.com/menderesk/beats/v7/libbeat/reader/multiline"
	"github.com/menderesk/beats/v7/libbeat/reader/readfile"
	"github.com/menderesk/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/menderesk/beats/v7/libbeat/reader/readjson"
)

var (
	harvesterMetrics = monitoring.Default.NewRegistry("filebeat.harvester")
	filesMetrics     = monitoring.GetNamespace("dataset").GetRegistry()

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

// OutletFactory provides an outlet for the harvester
type OutletFactory func() channel.Outleter

// Harvester contains all harvester related data
type Harvester struct {
	logger *logp.Logger

	id     uuid.UUID
	config config
	source harvester.Source // the source being watched

	// shutdown handling
	done     chan struct{}
	doneWg   *sync.WaitGroup
	stopOnce sync.Once
	stopWg   *sync.WaitGroup
	stopLock sync.Mutex

	// internal harvester state
	state  file.State
	states *file.States
	log    *Log

	// file reader pipeline
	reader          reader.Reader
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding

	// event/state publishing
	outletFactory OutletFactory
	publishState  func(file.State) bool

	metrics *harvesterProgressMetrics

	onTerminate func()
}

// stores the metrics of the harvester
type harvesterProgressMetrics struct {
	metricsRegistry             *monitoring.Registry
	filename                    *monitoring.String
	started                     *monitoring.String
	lastPublished               *monitoring.Timestamp
	lastPublishedEventTimestamp *monitoring.Timestamp
	currentSize                 *monitoring.Int
	readOffset                  *monitoring.Int
}

// NewHarvester creates a new harvester
func NewHarvester(
	logger *logp.Logger,
	config *common.Config,
	state file.State,
	states *file.States,
	publishState func(file.State) bool,
	outletFactory OutletFactory,
) (*Harvester, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	logger = logger.Named("harvester").With("harvester_id", id)
	h := &Harvester{
		logger:        logger,
		config:        defaultConfig(),
		state:         state,
		states:        states,
		publishState:  publishState,
		done:          make(chan struct{}),
		stopWg:        &sync.WaitGroup{},
		doneWg:        &sync.WaitGroup{},
		id:            id,
		outletFactory: outletFactory,
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
	return h, nil
}

// open does open the file given under h.Path and assigns the file handler to h.log
func (h *Harvester) open() error {
	switch h.config.Type {
	case harvester.StdinType:
		return h.openStdin()
	case harvester.LogType, harvester.DockerType, harvester.ContainerType:
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

	h.metrics = newHarvesterProgressMetrics(h.id.String())
	h.metrics.filename.Set(h.source.Name())
	h.metrics.started.Set(common.Time(time.Now()).String())
	h.metrics.readOffset.Set(h.state.Offset)
	err = h.updateCurrentSize()
	if err != nil {
		return err
	}

	h.logger.Debugf("Harvester setup successful. Line terminator: %d", h.config.LineTerminator)

	return nil
}

func newHarvesterProgressMetrics(id string) *harvesterProgressMetrics {
	r := filesMetrics.NewRegistry(id)
	return &harvesterProgressMetrics{
		metricsRegistry:             r,
		filename:                    monitoring.NewString(r, "name"),
		started:                     monitoring.NewString(r, "start_time"),
		lastPublished:               monitoring.NewTimestamp(r, "last_event_published_time"),
		lastPublishedEventTimestamp: monitoring.NewTimestamp(r, "last_event_timestamp"),
		currentSize:                 monitoring.NewInt(r, "size"),
		readOffset:                  monitoring.NewInt(r, "read_offset"),
	}
}

func (h *Harvester) updateCurrentSize() error {
	fInfo, err := h.source.Stat()
	if err != nil {
		return err
	}

	h.metrics.currentSize.Set(fInfo.Size())
	return nil
}

// Run start the harvester and reads files line by line and sends events to the defined output
func (h *Harvester) Run() error {
	logger := h.logger

	// Allow for some cleanup on termination
	if h.onTerminate != nil {
		defer h.onTerminate()
	}

	outlet := channel.CloseOnSignal(h.outletFactory(), h.done)
	forwarder := harvester.NewForwarder(outlet)

	// This is to make sure a harvester is not started anymore if stop was already
	// called before the harvester was started. The waitgroup is not incremented afterwards
	// as otherwise it could happened that between checking for the close channel and incrementing
	// the waitgroup, the harvester could be stopped.
	// Here stopLock is used to prevent a data race where stopWg.Add(1) below is called
	// while stopWg.Wait() is executing in a different goroutine, which is forbidden
	// according to sync.WaitGroup docs.
	h.stopLock.Lock()
	h.stopWg.Add(1)
	h.stopLock.Unlock()
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
		h.cleanup()

		harvesterRunning.Add(-1)

		// Marks harvester stopping completed
		h.stopWg.Done()
	}()

	harvesterStarted.Add(1)
	harvesterRunning.Add(1)

	// Closes reader after timeout or when done channel is closed
	// This routine is also responsible to properly stop the reader
	go func(source string) {
		closeTimeout := make(<-chan time.Time)
		// starts close_timeout timer
		if h.config.CloseTimeout > 0 {
			closeTimeout = time.After(h.config.CloseTimeout)
		}

		select {
		// Applies when timeout is reached
		case <-closeTimeout:
			logger.Infof("Closing harvester because close_timeout was reached: %s", source)
		// Required when reader loop returns and reader finished
		case <-h.done:
		}

		h.stop()
		err := h.reader.Close()
		if err != nil {
			logger.Errorf("Failed to stop harvester for file: %v", err)
		}
	}(h.state.Source)

	logger.Infof("Harvester started for paths: %v", h.config.Paths)

	h.doneWg.Add(1)
	go func() {
		h.monitorFileSize()
		h.doneWg.Done()
	}()

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
				logger.Info("File was truncated. Begin reading file from offset 0.")
				h.state.Offset = 0
				filesTruncated.Add(1)
			case ErrRemoved:
				logger.Info("File was removed. Closing because close_removed is enabled.")
			case ErrRenamed:
				logger.Info("File was renamed. Closing because close_renamed is enabled.")
			case ErrClosed:
				logger.Info("Reader was closed. Closing.")
			case io.EOF:
				logger.Info("End of file reached. Closing because close_eof is enabled.")
			case ErrInactive:
				logger.Infof("File is inactive. Closing because close_inactive of %v reached.", h.config.CloseInactive)
			default:
				logger.Errorf("Read line error: %v", err)
			}
			return nil
		}

		// Get copy of state to work on
		// This is important in case sending is not successful so on shutdown
		// the old offset is reported
		state := h.getState()
		startingOffset := state.Offset
		state.Offset += int64(message.Bytes)

		// Stop harvester in case of an error
		if !h.onMessage(forwarder, state, message, startingOffset) {
			return nil
		}

		// Update state of harvester as successfully sent
		h.state = state

		// Update metics of harvester as event was sent
		h.metrics.readOffset.Set(state.Offset)
		h.metrics.lastPublished.Set(time.Now())
		h.metrics.lastPublishedEventTimestamp.Set(message.Ts)
	}
}

func (h *Harvester) monitorFileSize() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			err := h.updateCurrentSize()
			if err != nil {
				h.logger.Errorf("Error updating file size: %v", err)
			}
		}
	}
}

// stop is intended for internal use and closed the done channel to stop execution
func (h *Harvester) stop() {
	h.stopOnce.Do(func() {
		close(h.done)
		// Wait for goroutines monitoring h.done to terminate before closing source.
		h.doneWg.Wait()
		filesMetrics.Remove(h.id.String())
	})
}

// Stop stops harvester and waits for completion
func (h *Harvester) Stop() {
	h.stop()
	// Prevent stopWg.Wait() to be called at the same time as stopWg.Add(1)
	h.stopLock.Lock()
	h.stopWg.Wait()
	h.stopLock.Unlock()
}

// onMessage processes a new message read from the reader.
// This results in a state update and possibly an event would be send.
// A state update first updates the in memory state held by the prospector,
// and finally sends the file.State indirectly to the registrar.
// The events Private field is used to forward the file state update.
//
// onMessage returns 'false' if it was interrupted in the process of sending the event.
// This normally signals a harvester shutdown.
func (h *Harvester) onMessage(
	forwarder *harvester.Forwarder,
	state file.State,
	message reader.Message,
	messageOffset int64,
) bool {
	if h.source.HasState() {
		h.states.Update(state)
	}

	text := string(message.Content)
	if message.IsEmpty() || !h.shouldExportLine(text) {
		// No data or event is filtered out -> send empty event with state update
		// only. The call can fail on filebeat shutdown.
		// The event will be filtered out, but forwarded to the registry as is.
		err := forwarder.Send(beat.Event{Private: state})
		return err == nil
	}

	fields := common.MapStr{
		"log": common.MapStr{
			"offset": messageOffset, // Offset here is the offset before the starting char.
			"file": common.MapStr{
				"path": state.Source,
			},
		},
	}
	fields.DeepUpdate(message.Fields)

	// Check if json fields exist
	var jsonFields common.MapStr
	if f, ok := fields["json"]; ok {
		jsonFields = f.(common.MapStr)
	}

	var meta common.MapStr
	timestamp := message.Ts
	if h.config.JSON != nil && len(jsonFields) > 0 {
		id, ts := readjson.MergeJSONFields(fields, jsonFields, &text, *h.config.JSON)
		if !ts.IsZero() {
			// there was a `@timestamp` key in the event, so overwrite
			// the resulting timestamp
			timestamp = ts
		}

		if id != "" {
			meta = common.MapStr{
				"_id": id,
			}
		}
	} else if &text != nil {
		if fields == nil {
			fields = common.MapStr{}
		}
		fields["message"] = text
	}

	err := forwarder.Send(beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
		Meta:      meta,
		Private:   state,
	})
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

	h.publishState(h.state)

	h.logger.Debugf("Update state (offset: %v).", h.state.Offset)
	h.states.Update(h.state)
}

// shouldExportLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (h *Harvester) shouldExportLine(line string) bool {
	if len(h.config.IncludeLines) > 0 {
		if !harvester.MatchAny(h.config.IncludeLines, line) {
			// drop line
			h.logger.Debugf("Drop line as it does not match any of the include patterns %s", line)
			return false
		}
	}
	if len(h.config.ExcludeLines) > 0 {
		if harvester.MatchAny(h.config.ExcludeLines, line) {
			// drop line
			h.logger.Debugf("Drop line as it does match one of the exclude patterns%s", line)
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
	fi, err := os.Stat(h.state.Source)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %v", h.state.Source, err)
	}
	if fi.Mode()&os.ModeNamedPipe != 0 {
		return fmt.Errorf("failed to open file %s, named pipes are not supported", h.state.Source)
	}

	f, err := file_helper.ReadOpen(h.state.Source)
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
	logger := h.logger

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("Failed getting stats for file %s: %s", h.state.Source, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("Tried to open non regular file: %q %s", info.Mode(), info.Name())
	}

	// Compares the stat of the opened file to the state given by the input. Abort if not match.
	if !os.SameFile(h.state.Fileinfo, info) {
		return errors.New("file info is not identical with opened file. Aborting harvesting and retrying file later again")
	}

	h.encoding, err = h.encodingFactory(f)
	if err != nil {

		if err == transform.ErrShortSrc {
			logger.Infof("Initialising encoding for '%v' failed due to file being too short", f)
		} else {
			logger.Errorf("Initialising encoding for '%v' failed: %v", f, err)
		}
		return err
	}

	// get file offset. Only update offset if no error
	offset, err := h.initFileOffset(f)
	if err != nil {
		return err
	}

	logger.Debugf("Setting offset: %d ", offset)
	h.state.Offset = offset

	return nil
}

func (h *Harvester) initFileOffset(file *os.File) (int64, error) {
	// continue from last known offset
	if h.state.Offset > 0 {
		h.logger.Debugf("Set previous offset: %d ", h.state.Offset)
		return file.Seek(h.state.Offset, os.SEEK_SET)
	}

	// get offset from file in case of encoding factory was required to read some data.
	h.logger.Debug("Setting offset to: 0")
	return file.Seek(0, os.SEEK_CUR)
}

// getState returns an updated copy of the harvester state
func (h *Harvester) getState() file.State {
	if !h.source.HasState() {
		return file.State{}
	}

	state := h.state

	// refreshes the values in State with the values from the harvester itself
	state.FileStateOS = file_helper.GetOSState(h.state.Fileinfo)
	return state
}

func (h *Harvester) cleanup() {
	// Mark harvester as finished
	h.state.Finished = true

	h.logger.Debugf("Stopping harvester.")
	defer h.logger.Debugf("harvester cleanup finished.")

	// Make sure file is closed as soon as harvester exits
	// If file was never opened, it can't be closed
	if h.source != nil {

		// close file handler
		h.source.Close()

		h.logger.Debugf("Closing file")
		harvesterOpenFiles.Add(-1)

		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		// Only send offset if file object was created successfully
		h.SendStateUpdate()
	} else {
		h.logger.Warn("Stopping harvester, NOT closing file as file info not available.")
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

	h.logger.Debugf("newLogFileReader with config.MaxBytes: %d", h.config.MaxBytes)

	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	h.log, err = NewLog(h.logger, h.source, h.config.LogConfig)
	if err != nil {
		return nil, err
	}

	reader, err := debug.AppendReaders(h.log)
	if err != nil {
		return nil, err
	}

	// Configure MaxBytes limit for EncodeReader as multiplied by 4
	// for the worst case scenario where incoming UTF32 charchers are decoded to the single byte UTF-8 characters.
	// This limit serves primarily to avoid memory bload or potential OOM with expectedly long lines in the file.
	// The further size limiting is performed by LimitReader at the end of the readers pipeline as needed.
	encReaderMaxBytes := h.config.MaxBytes * 4

	r, err = readfile.NewEncodeReader(reader, readfile.Config{
		Codec:      h.encoding,
		BufferSize: h.config.BufferSize,
		Terminator: h.config.LineTerminator,
		MaxBytes:   encReaderMaxBytes,
	})
	if err != nil {
		return nil, err
	}

	if h.config.DockerJSON != nil {
		// Docker json-file format, add custom parsing to the pipeline
		r = readjson.New(r, h.config.DockerJSON.Stream, h.config.DockerJSON.Partial, h.config.DockerJSON.Format, h.config.DockerJSON.CRIFlags)
	}

	if h.config.JSON != nil {
		r = readjson.NewJSONReader(r, h.config.JSON)
	}

	r = readfile.NewStripNewline(r, h.config.LineTerminator)

	if h.config.Multiline != nil {
		r, err = multiline.New(r, "\n", h.config.MaxBytes, h.config.Multiline)
		if err != nil {
			return nil, err
		}
	}

	return readfile.NewLimitReader(r, h.config.MaxBytes), nil
}
