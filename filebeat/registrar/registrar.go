package registrar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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
	registryFile string       // Path to the Registry File
	states       *file.States // Map with all file paths inside and the corresponding state
	wg           sync.WaitGroup
}

type successLogger interface {
	Published(n int) bool
}

var (
	statesUpdate   = monitoring.NewInt(nil, "registrar.states.update")
	statesCleanup  = monitoring.NewInt(nil, "registrar.states.cleanup")
	statesCurrent  = monitoring.NewInt(nil, "registrar.states.current")
	registryWrites = monitoring.NewInt(nil, "registrar.writes")
)

func New(registryFile string, out successLogger) (*Registrar, error) {
	r := &Registrar{
		registryFile: registryFile,
		done:         make(chan struct{}),
		states:       file.NewStates(),
		Channel:      make(chan []file.State, 1),
		out:          out,
		wg:           sync.WaitGroup{},
	}
	err := r.Init()

	return r, err
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

	logp.Info("Registry file set to: %s", r.registryFile)

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

	decoder := json.NewDecoder(f)
	states := []file.State{}
	err = decoder.Decode(&states)
	if err != nil {
		return fmt.Errorf("Error decoding states: %s", err)
	}

	states = resetStates(states)
	r.states.SetStates(states)
	logp.Info("States Loaded from registrar: %+v", len(states))

	return nil
}

// resetStates sets all states to finished and disable TTL on restart
// For all states covered by a prospector, TTL will be overwritten with the prospector value
func resetStates(states []file.State) []file.State {
	for key, state := range states {
		state.Finished = true
		// Set ttl to -2 to easily spot which states are not managed by a prospector
		state.TTL = -2
		states[key] = state
	}
	return states
}

func (r *Registrar) Start() error {
	// Load the previous log file locations now, for use in prospector
	err := r.loadStates()
	if err != nil {
		return fmt.Errorf("Error loading state: %v", err)
	}

	r.wg.Add(1)
	go r.Run()

	return nil
}

func (r *Registrar) Run() {
	logp.Info("Starting Registrar")
	// Writes registry on shutdown
	defer func() {
		r.writeRegistry()
		r.wg.Done()
	}()

	for {
		select {
		case <-r.done:
			logp.Info("Ending Registrar")
			return
		case states := <-r.Channel:
			r.onEvents(states)
		}
	}
}

// onEvents processes events received from the publisher pipeline
func (r *Registrar) onEvents(states []file.State) {
	r.processEventStates(states)

	beforeCount := r.states.Count()
	cleanedStates := r.states.Cleanup()
	statesCleanup.Add(int64(cleanedStates))

	logp.Debug("registrar",
		"Registrar states cleaned up. Before: %d, After: %d",
		beforeCount, beforeCount-cleanedStates)

	if err := r.writeRegistry(); err != nil {
		logp.Err("Writing of registry returned error: %v. Continuing...", err)
	}

	if r.out != nil {
		r.out.Published(len(states))
	}
}

// processEventStates gets the states from the events and writes them to the registrar state
func (r *Registrar) processEventStates(states []file.State) {
	logp.Debug("registrar", "Processing %d events", len(states))

	for i := range states {
		r.states.Update(states[i])
		statesUpdate.Add(1)
	}
}

// Stop stops the registry. It waits until Run function finished.
func (r *Registrar) Stop() {
	logp.Info("Stopping Registrar")
	close(r.done)
	r.wg.Wait()
}

// writeRegistry writes the new json registry file to disk.
func (r *Registrar) writeRegistry() error {
	logp.Debug("registrar", "Write registry file: %s", r.registryFile)

	tempfile := r.registryFile + ".new"
	f, err := os.OpenFile(tempfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_SYNC, 0600)
	if err != nil {
		logp.Err("Failed to create tempfile (%s) for writing: %s", tempfile, err)
		return err
	}

	// First clean up states
	states := r.states.GetStates()

	encoder := json.NewEncoder(f)
	err = encoder.Encode(states)
	if err != nil {
		f.Close()
		logp.Err("Error when encoding the states: %s", err)
		return err
	}

	// Directly close file because of windows
	f.Close()

	err = helper.SafeFileRotate(r.registryFile, tempfile)

	logp.Debug("registrar", "Registry file updated. %d states written.", len(states))
	registryWrites.Add(1)
	statesCurrent.Set(int64(len(states)))

	return err
}
