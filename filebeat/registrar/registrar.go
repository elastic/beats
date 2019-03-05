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

package registrar

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input/file"
	helper "github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/paths"
)

type Registrar struct {
	Channel      chan []file.State
	out          successLogger
	done         chan struct{}
	registryFile string      // Path to the Registry File
	fileMode     os.FileMode // Permissions to apply on the Registry File
	wg           sync.WaitGroup

	states               *file.States // Map with all file paths inside and the corresponding state
	gcRequired           bool         // gcRequired is set if registry state needs to be gc'ed before the next write
	gcEnabled            bool         // gcEnabled indicates the registry contains some state that can be gc'ed in the future
	flushTimeout         time.Duration
	bufferedStateUpdates int
}

type successLogger interface {
	Published(n int) bool
}

var (
	statesUpdate    = monitoring.NewInt(nil, "registrar.states.update")
	statesCleanup   = monitoring.NewInt(nil, "registrar.states.cleanup")
	statesCurrent   = monitoring.NewInt(nil, "registrar.states.current")
	registryWrites  = monitoring.NewInt(nil, "registrar.writes.total")
	registryFails   = monitoring.NewInt(nil, "registrar.writes.fail")
	registrySuccess = monitoring.NewInt(nil, "registrar.writes.success")
)

// New creates a new Registrar instance, updating the registry file on
// `file.State` updates. New fails if the file can not be opened or created.
func New(cfg config.Registry, out successLogger) (*Registrar, error) {
	home := paths.Resolve(paths.Data, cfg.Path)
	migrateFile := cfg.MigrateFile
	if migrateFile != "" {
		migrateFile = paths.Resolve(paths.Data, migrateFile)
	}

	err := ensureCurrent(home, migrateFile, cfg.Permissions)
	if err != nil {
		return nil, err
	}

	dataFile := filepath.Join(home, "filebeat", "data.json")
	r := &Registrar{
		registryFile: dataFile,
		fileMode:     cfg.Permissions,
		done:         make(chan struct{}),
		states:       file.NewStates(),
		Channel:      make(chan []file.State, 1),
		flushTimeout: cfg.FlushTimeout,
		out:          out,
		wg:           sync.WaitGroup{},
	}
	return r, r.Init()
}

// Init sets up the Registrar and make sure the registry file is setup correctly
func (r *Registrar) Init() error {
	// The registry file is opened in the data path
	r.registryFile = paths.Resolve(paths.Data, r.registryFile)

	// Create directory if it does not already exist.
	registryPath := filepath.Dir(r.registryFile)
	err := os.MkdirAll(registryPath, 0750)
	if err != nil {
		return fmt.Errorf("Failed to created registry file dir %s: %v", registryPath, err)
	}

	// Check if files exists
	fileInfo, err := os.Lstat(r.registryFile)
	if os.IsNotExist(err) {
		logp.Info("No registry file found under: %s. Creating a new registry file.", r.registryFile)
		// No registry exists yet, write empty state to check if registry can be written
		return r.writeRegistry()
	}
	if err != nil {
		return err
	}

	// Check if regular file, no dir, no symlink
	if !fileInfo.Mode().IsRegular() {
		// Special error message for directory
		if fileInfo.IsDir() {
			return fmt.Errorf("Registry file path must be a file. %s is a directory.", r.registryFile)
		}
		return fmt.Errorf("Registry file path is not a regular file: %s", r.registryFile)
	}

	logp.Debug("registrar", "Registry file set to: %s", r.registryFile)

	return nil
}

// GetStates return the registrar states
func (r *Registrar) GetStates() []file.State {
	return r.states.GetStates()
}

// loadStates fetches the previous reading state from the configure RegistryFile file
// The default file is `registry` in the data path.
func (r *Registrar) loadStates() error {
	f, err := os.Open(r.registryFile)
	if err != nil {
		return err
	}

	defer f.Close()

	logp.Info("Loading registrar data from %s", r.registryFile)

	states, err := readStatesFrom(f)
	if err != nil {
		return err
	}
	r.states.SetStates(states)
	logp.Info("States Loaded from registrar: %+v", len(states))

	return nil
}

func readStatesFrom(in io.Reader) ([]file.State, error) {
	states := []file.State{}
	decoder := json.NewDecoder(in)
	if err := decoder.Decode(&states); err != nil {
		return nil, fmt.Errorf("Error decoding states: %s", err)
	}

	states = fixStates(states)
	states = resetStates(states)
	return states, nil
}

// fixStates cleans up the registry states when updating from an older version
// of filebeat potentially writing invalid entries.
func fixStates(states []file.State) []file.State {
	if len(states) == 0 {
		return states
	}

	// we use a map of states here, so to identify and merge duplicate entries.
	idx := map[string]*file.State{}
	for i := range states {
		state := &states[i]
		fixState(state)

		id := state.ID()
		old, exists := idx[id]
		if !exists {
			idx[id] = state
		} else {
			mergeStates(old, state) // overwrite the entry in 'old'
		}
	}

	if len(idx) == len(states) {
		return states
	}

	i := 0
	newStates := make([]file.State, len(idx))
	for _, state := range idx {
		newStates[i] = *state
		i++
	}
	return newStates
}

// fixState updates a read state to fullfil required invariantes:
// - "Meta" must be nil if len(Meta) == 0
func fixState(st *file.State) {
	if len(st.Meta) == 0 {
		st.Meta = nil
	}
}

// mergeStates merges 2 states by trying to determine the 'newer' state.
// The st state is overwritten with the updated fields.
func mergeStates(st, other *file.State) {
	st.Finished = st.Finished || other.Finished
	if st.Offset < other.Offset { // always select the higher offset
		st.Offset = other.Offset
	}

	// update file meta-data. As these are updated concurrently by the
	// inputs, select the newer state based on the update timestamp.
	var meta, metaOld, metaNew map[string]string
	if st.Timestamp.Before(other.Timestamp) {
		st.Source = other.Source
		st.Timestamp = other.Timestamp
		st.TTL = other.TTL
		st.FileStateOS = other.FileStateOS

		metaOld, metaNew = st.Meta, other.Meta
	} else {
		metaOld, metaNew = other.Meta, st.Meta
	}

	if len(metaOld) == 0 || len(metaNew) == 0 {
		meta = metaNew
	} else {
		meta = map[string]string{}
		for k, v := range metaOld {
			meta[k] = v
		}
		for k, v := range metaNew {
			meta[k] = v
		}
	}

	if len(meta) == 0 {
		meta = nil
	}
	st.Meta = meta
}

// resetStates sets all states to finished and disable TTL on restart
// For all states covered by an input, TTL will be overwritten with the input value
func resetStates(states []file.State) []file.State {
	for key, state := range states {
		state.Finished = true
		// Set ttl to -2 to easily spot which states are not managed by a input
		state.TTL = -2
		states[key] = state
	}
	return states
}

func (r *Registrar) Start() error {
	// Load the previous log file locations now, for use in input
	err := r.loadStates()
	if err != nil {
		return fmt.Errorf("Error loading state: %v", err)
	}

	r.wg.Add(1)
	go r.Run()

	return nil
}

func (r *Registrar) Run() {
	logp.Debug("registrar", "Starting Registrar")
	// Writes registry on shutdown
	defer func() {
		r.writeRegistry()
		r.wg.Done()
	}()

	var (
		timer  *time.Timer
		flushC <-chan time.Time
	)

	for {
		select {
		case <-r.done:
			logp.Info("Ending Registrar")
			return
		case <-flushC:
			flushC = nil
			timer.Stop()
			r.flushRegistry()
		case states := <-r.Channel:
			r.onEvents(states)
			if r.flushTimeout <= 0 {
				r.flushRegistry()
			} else if flushC == nil {
				timer = time.NewTimer(r.flushTimeout)
				flushC = timer.C
			}
		}
	}
}

// onEvents processes events received from the publisher pipeline
func (r *Registrar) onEvents(states []file.State) {
	r.processEventStates(states)
	r.bufferedStateUpdates += len(states)

	// check if we need to enable state cleanup
	if !r.gcEnabled {
		for i := range states {
			if states[i].TTL >= 0 || states[i].Finished {
				r.gcEnabled = true
				break
			}
		}
	}

	logp.Debug("registrar", "Registrar state updates processed. Count: %v", len(states))

	// new set of events received -> mark state registry ready for next
	// cleanup phase in case gc'able events are stored in the registry.
	r.gcRequired = r.gcEnabled
}

// gcStates runs a registry Cleanup. The method check if more event in the
// registry can be gc'ed in the future. If no potential removable state is found,
// the gcEnabled flag is set to false, indicating the current registrar state being
// stable. New registry update events can re-enable state gc'ing.
func (r *Registrar) gcStates() {
	if !r.gcRequired {
		return
	}

	beforeCount := r.states.Count()
	cleanedStates, pendingClean := r.states.Cleanup()
	statesCleanup.Add(int64(cleanedStates))

	logp.Debug("registrar",
		"Registrar states cleaned up. Before: %d, After: %d, Pending: %d",
		beforeCount, beforeCount-cleanedStates, pendingClean)

	r.gcRequired = false
	r.gcEnabled = pendingClean > 0
}

// processEventStates gets the states from the events and writes them to the registrar state
func (r *Registrar) processEventStates(states []file.State) {
	logp.Debug("registrar", "Processing %d events", len(states))

	ts := time.Now()
	for i := range states {
		r.states.UpdateWithTs(states[i], ts)
		statesUpdate.Add(1)
	}
}

// Stop stops the registry. It waits until Run function finished.
func (r *Registrar) Stop() {
	logp.Info("Stopping Registrar")
	close(r.done)
	r.wg.Wait()
}

func (r *Registrar) flushRegistry() {
	if err := r.writeRegistry(); err != nil {
		logp.Err("Writing of registry returned error: %v. Continuing...", err)
	}

	if r.out != nil {
		r.out.Published(r.bufferedStateUpdates)
	}
	r.bufferedStateUpdates = 0
}

// writeRegistry writes the new json registry file to disk.
func (r *Registrar) writeRegistry() error {
	// First clean up states
	r.gcStates()
	states := r.states.GetStates()
	statesCurrent.Set(int64(len(states)))

	registryWrites.Inc()

	tempfile, err := writeTmpFile(r.registryFile, r.fileMode, states)
	if err != nil {
		registryFails.Inc()
		return err
	}

	err = helper.SafeFileRotate(r.registryFile, tempfile)
	if err != nil {
		registryFails.Inc()
		return err
	}

	logp.Debug("registrar", "Registry file updated. %d states written.", len(states))
	registrySuccess.Inc()

	return nil
}

func writeTmpFile(baseName string, perm os.FileMode, states []file.State) (string, error) {
	logp.Debug("registrar", "Write registry file: %s (%v)", baseName, len(states))

	tempfile := baseName + ".new"
	f, err := os.OpenFile(tempfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_SYNC, perm)
	if err != nil {
		logp.Err("Failed to create tempfile (%s) for writing: %s", tempfile, err)
		return "", err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)

	if err := encoder.Encode(states); err != nil {
		logp.Err("Error when encoding the states: %s", err)
		return "", err
	}

	// Commit the changes to storage to avoid corrupt registry files
	if err = f.Sync(); err != nil {
		logp.Err("Error when syncing new registry file contents: %s", err)
		return "", err
	}

	return tempfile, nil
}
