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

// Package checkpoint persists event log state information to disk so that
// event log monitoring can resume from the last read event in the case of a
// restart or unexpected interruption.
package checkpoint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Checkpoint persists event log state information to disk.
type Checkpoint struct {
	wg            sync.WaitGroup // WaitGroup used to wait on the shutdown of the checkpoint worker.
	done          chan struct{}  // Channel for shutting down the checkpoint worker.
	once          sync.Once      // Used to guarantee shutdown happens once.
	file          string         // File where the state is persisted.
	fileLock      sync.RWMutex   // Lock that protects concurrent reads/writes to file.
	numUpdates    int            // Number of updates received since last persisting to disk.
	flushInterval time.Duration  // Maximum time interval that can pass before persisting to disk.
	sort          []string       // Slice used for sorting states map (store to save on mallocs).

	lock   sync.RWMutex
	states map[string]EventLogState

	save chan EventLogState
}

// PersistedState represents the format of the data persisted to disk.
type PersistedState struct {
	UpdateTime time.Time       `yaml:"update_time"`
	States     []EventLogState `yaml:"event_logs"`
}

// EventLogState represents the state of an individual event log.
type EventLogState struct {
	Name         string    `yaml:"name"`
	RecordNumber uint64    `yaml:"record_number"`
	Timestamp    time.Time `yaml:"timestamp"`
	Bookmark     string    `yaml:"bookmark,omitempty"`
}

// NewCheckpoint creates and returns a new Checkpoint. This method loads state
// information from disk if it exists and starts a goroutine for persisting
// state information to disk. Shutdown should be called when finished to
// guarantee any in-memory state information is flushed to disk.
//
// file is the name of the file where event log state is persisted as YAML.
// interval is maximum amount of time that can pass since the last flush
// before triggering a flush to disk (minimum value is 1s).
func NewCheckpoint(file string, interval time.Duration) (*Checkpoint, error) {
	c := &Checkpoint{
		done:          make(chan struct{}),
		file:          file,
		flushInterval: interval,
		sort:          make([]string, 0, 10),
		states:        make(map[string]EventLogState),
		save:          make(chan EventLogState, 1),
	}

	// Minimum flush interval.
	if c.flushInterval < time.Second {
		c.flushInterval = time.Second
	}

	// Read existing state information:
	ps, err := c.read()
	if err != nil {
		return nil, err
	}

	if ps != nil {
		for _, state := range ps.States {
			c.states[state.Name] = state
		}
	}

	// Write the state file to verify we have have permissions.
	err = c.flush()
	if err != nil {
		return nil, err
	}

	c.wg.Add(1)
	go c.run()
	return c, nil
}

// run is worker loop that reads incoming state information from the save
// channel and persists it when the number of changes reaches maxEvents or
// the amount of time since the last disk write reaches flushInterval.
func (c *Checkpoint) run() {
	defer c.wg.Done()
	defer c.persist()

	flushTimer := time.NewTimer(c.flushInterval)
	defer flushTimer.Stop()
loop:
	for {
		select {
		case <-c.done:
			break loop
		case s := <-c.save:
			c.lock.Lock()
			c.states[s.Name] = s
			c.lock.Unlock()
			c.numUpdates++
		case <-flushTimer.C:
		}

		c.persist()
		flushTimer.Reset(c.flushInterval)
	}
}

// Shutdown stops the checkpoint worker (which persists any state to disk as
// it stops). This method blocks until the checkpoint worker shutdowns. Calling
// this method more once is safe and has no effect.
func (c *Checkpoint) Shutdown() {
	c.once.Do(func() {
		close(c.done)
		c.wg.Wait()
	})
}

// States returns the current in-memory event log state. This state information
// is bootstrapped with any data found on disk at creation time.
func (c *Checkpoint) States() map[string]EventLogState {
	c.lock.RLock()
	defer c.lock.RUnlock()

	copy := make(map[string]EventLogState)
	for k, v := range c.states {
		copy[k] = v
	}

	return copy
}

// Persist queues the given event log state information to be written to disk.
func (c *Checkpoint) Persist(name string, recordNumber uint64, ts time.Time, bookmark string) {
	c.PersistState(EventLogState{
		Name:         name,
		RecordNumber: recordNumber,
		Timestamp:    ts,
		Bookmark:     bookmark,
	})
}

// PersistState queues the given event log state to be written to disk.
func (c *Checkpoint) PersistState(st EventLogState) {
	c.save <- st
}

// persist writes the current state to disk if the in-memory state is dirty.
func (c *Checkpoint) persist() bool {
	if c.numUpdates == 0 {
		return false
	}

	err := c.flush()
	if err != nil {
		logp.Err("%v", err)
		return false
	}

	logp.Debug("checkpoint", "Checkpoint saved to disk. numUpdates=%d",
		c.numUpdates)
	c.numUpdates = 0
	return true
}

// flush writes the current state to disk.
func (c *Checkpoint) flush() error {
	c.fileLock.Lock()
	defer c.fileLock.Unlock()

	tempFile := c.file + ".new"
	file, err := create(tempFile)
	if os.IsNotExist(err) {
		// Try to create directory if it does not exist.
		if createDirErr := c.createDir(); createDirErr == nil {
			file, err = create(tempFile)
		}
	}

	if err != nil {
		return fmt.Errorf("Failed to flush state to disk. %v", err)
	}

	// Sort persisted eventLogs by name.
	c.sort = c.sort[:0]
	for k := range c.states {
		c.sort = append(c.sort, k)
	}
	sort.Strings(c.sort)

	ps := PersistedState{
		UpdateTime: time.Now().UTC(),
		States:     make([]EventLogState, len(c.sort)),
	}
	for i, name := range c.sort {
		ps.States[i] = c.states[name]
	}

	data, err := yaml.Marshal(ps)
	if err != nil {
		file.Close()
		return fmt.Errorf("Failed to flush state to disk. Could not marshal "+
			"data to YAML. %v", err)
	}

	_, err = file.Write(data)
	if err != nil {
		file.Close()
		return fmt.Errorf("Failed to flush state to disk. Could not write to "+
			"%s. %v", tempFile, err)
	}

	file.Close()
	err = os.Rename(tempFile, c.file)
	return err
}

// read loads the persisted state from disk. If the file does not exists then
// the method returns nil and no error.
func (c *Checkpoint) read() (*PersistedState, error) {
	c.fileLock.RLock()
	defer c.fileLock.RUnlock()

	contents, err := ioutil.ReadFile(c.file)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}

	ps := &PersistedState{}
	err = yaml.Unmarshal(contents, ps)
	if err != nil {
		return nil, fmt.Errorf("failed to read persisted state: %w", err)
	}

	return ps, nil
}

// createDir creates the directory in which the state file will reside if the
// directory does not already exist.
func (c *Checkpoint) createDir() error {
	dir := filepath.Dir(c.file)
	logp.Info("Creating %s if it does not exist.", dir)
	return os.MkdirAll(dir, os.FileMode(0o750))
}
