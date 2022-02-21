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
	file_helper "github.com/elastic/beats/libbeat/common/file"
	"io"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/reader/readjson"
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
	ErrReadTimeout  = errors.New("reader timeout")
)

// OutletFactory provides an outlet for the harvester
type OutletFactory func() channel.Outleter

// Harvester contains all harvester related data
type Harvester struct {
	id     uuid.UUID
	config config

	// shutdown handling
	done     chan struct{}
	stopOnce sync.Once
	stopWg   *sync.WaitGroup
	stopLock sync.Mutex

	// internal harvester state
	state  file.State
	states *file.States

	// file reader pipeline
	reader *ReuseHarvester

	// event/state publishing
	outletFactory OutletFactory
	publishState  func(*util.Data) bool

	onTerminate func()
}

// NewHarvester creates a new harvester
func NewHarvester(
	config *common.Config,
	state file.State,
	states *file.States,
	publishState func(*util.Data) bool,
	outletFactory OutletFactory,
) (*Harvester, error) {

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	h := &Harvester{
		config:        defaultConfig,
		state:         state,
		states:        states,
		publishState:  publishState,
		done:          make(chan struct{}),
		stopWg:        &sync.WaitGroup{},
		id:            id,
		outletFactory: outletFactory,
	}

	if err := config.Unpack(&h.config); err != nil {
		return nil, err
	}

	// Add outlet signal so harvester can also stop itself
	return h, nil
}

// ID returns the unique harvester identifier
func (h *Harvester) ID() uuid.UUID {
	return h.id
}

// Setup opens the file handler and creates the reader for the harvester
func (h *Harvester) Setup() error {
	//init reuse reader
	var err error
	h.reader, err = NewReuseHarvester(h.id, h.config, h.state)
	if err != nil {
		return fmt.Errorf("harvester init failed. Unexpected encoding line reader error: %s", err)
	}
	return nil
}

// Run start the harvester and reads files line by line and sends events to the defined output
func (h *Harvester) Run() error {
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
		// Close reader
		h.reader.Stop()

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
			logp.Info("Closing harvester because close_timeout was reached: %s", source)
		// Required when reader loop returns and reader finished
		case <-h.done:
		}

		h.stop()

		// Close reader
		h.reader.Stop()
	}(h.state.Source)

	logp.Info("Harvester started for file: %s, offset: %d", h.state.Source, h.state.Offset)

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
				logp.Err("Read line error: %v; File: %v", err, h.state.Source)
			}
			return nil
		}

		// Get copy of state to work on
		// This is important in case sending is not successful so on shutdown
		// the old offset is reported
		state := h.getState()
		state.Offset += int64(message.Bytes)

		// Create state event
		data := util.NewData()
		if h.reader.HasState() {
			data.SetState(state)
		}

		text := string(message.Content)

		// Check if data should be added to event. Only export non empty events.
		if !message.IsEmpty() && h.shouldExportLine(text) {
			fields := common.MapStr{}
			fields.DeepUpdate(message.Fields)

			// Check if json fields exist
			var jsonFields common.MapStr
			if f, ok := fields["json"]; ok {
				jsonFields = f.(common.MapStr)
			}

			data.Event = beat.Event{
				Timestamp: message.Ts,
			}

			if h.config.JSON != nil && len(jsonFields) > 0 {
				ts := readjson.MergeJSONFields(fields, jsonFields, &text, *h.config.JSON)
				if !ts.IsZero() {
					// there was a `@timestamp` key in the event, so overwrite
					// the resulting timestamp
					data.Event.Timestamp = ts
				}
			} else if &text != nil {
				if fields == nil {
					fields = common.MapStr{}
				}
				fields["data"] = text
			}

			data.Event.Fields = fields
		}

		// Always send event to update state, also if lines was skipped
		// Stop harvester in case of an error
		if !h.sendEvent(data, forwarder) {
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
	// Prevent stopWg.Wait() to be called at the same time as stopWg.Add(1)
	h.stopLock.Lock()
	h.stopWg.Wait()
	h.stopLock.Unlock()
}

// sendEvent sends event to the spooler channel
// Return false if event was not sent
func (h *Harvester) sendEvent(data *util.Data, forwarder *harvester.Forwarder) bool {
	if h.reader.HasState() {
		h.states.Update(data.GetState())
	}

	err := forwarder.Send(data)
	return err == nil
}

// SendStateUpdate send an empty event with the current state to update the registry
// close_timeout does not apply here to make sure a harvester is closed properly. In
// case the output is blocked the harvester will stay open to make sure no new harvester
// is started. As soon as the output becomes available again, the finished state is written
// and processing can continue.
func (h *Harvester) SendStateUpdate() {
	if !h.reader.HasState() {
		return
	}

	logp.Debug("harvester", "Update state: %s, offset: %v", h.state.Source, h.state.Offset)
	h.states.Update(h.state)

	d := util.NewData()
	d.SetState(h.state)
	h.publishState(d)
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

func (h *Harvester) cleanup() {
	// Mark harvester as finished
	h.state.Finished = true

	logp.Debug("harvester", "Stopping harvester for file: %s", h.state.Source)
	defer logp.Debug("harvester", "harvester cleanup finished for file: %s", h.state.Source)

	// Make sure file is closed as soon as harvester exits
	// If file was never opened, it can't be closed
	if h.reader != nil {
		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		// Only send offset if file object was created successfully
		h.SendStateUpdate()
	} else {
		logp.Warn("Stopping harvester, NOT closing file as file info not available: %s", h.state.Source)
	}

	harvesterClosed.Add(1)
}

// getState returns an updated copy of the harvester state
func (h *Harvester) getState() file.State {
	if !h.reader.HasState() {
		return file.State{}
	}

	state := h.state

	// refreshes the values in State with the values from the harvester itself
	fileState := h.reader.GetState()
	state.Source = fileState.Source
	state.TTL = fileState.TTL
	state.FileStateOS = file_helper.GetOSState(fileState.Fileinfo)
	return state
}
