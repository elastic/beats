package registrar

import (
	"encoding/json"
	"expvar"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/publisher"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

type Registrar struct {
	Channel      chan []*input.Event
	out          publisher.SuccessLogger
	done         chan struct{}
	registryFile string       // Path to the Registry File
	states       *file.States // Map with all file paths inside and the corresponding state
	wg           sync.WaitGroup
}

var (
	statesUpdate   = expvar.NewInt("registrar.states.update")
	statesCleanup  = expvar.NewInt("registrar.states.cleanup")
	statesCurrent  = expvar.NewInt("registar.states.current")
	registryWrites = expvar.NewInt("registrar.writes")
)

func New(registryFile string, out publisher.SuccessLogger) (*Registrar, error) {

	r := &Registrar{
		registryFile: registryFile,
		done:         make(chan struct{}),
		states:       file.NewStates(),
		Channel:      make(chan []*input.Event, 1),
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
	err := os.MkdirAll(registryPath, 0755)
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
func (r *Registrar) GetStates() file.States {
	return *r.states
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

	// DEPRECATED: This should be removed in 6.0
	oldStates := r.loadAndConvertOldState(f)
	if oldStates {
		return nil
	}

	decoder := json.NewDecoder(f)
	states := []file.State{}
	err = decoder.Decode(&states)
	if err != nil {
		return fmt.Errorf("Error decoding states: %s", err)
	}

	// Set all states to finished and disable TTL on restart
	// For all states covered by a prospector, TTL will be overwritten with the prospector value
	for key, state := range states {
		state.Finished = true
		// Set ttl to -2 to easily spot which states are not managed by a prospector
		state.TTL = -2
		states[key] = state
	}

	r.states.SetStates(states)
	logp.Info("States Loaded from registrar: %+v", len(states))

	return nil
}

// loadAndConvertOldState loads the old state file and converts it to the new state
// This is designed so it can be easily removed in later versions
func (r *Registrar) loadAndConvertOldState(f *os.File) bool {
	// Make sure file reader is reset afterwards
	defer f.Seek(0, 0)

	stat, err := f.Stat()
	if err != nil {
		logp.Err("Error getting stat for old state: %+v", err)
		return false
	}

	// Empty state does not have to be transformed ({} + newline)
	if stat.Size() <= 4 {
		return false
	}

	// Check if already new state format
	decoder := json.NewDecoder(f)
	newState := []file.State{}
	err = decoder.Decode(&newState)
	// No error means registry is already in new format
	if err == nil {
		return false
	}

	// Reset file offset
	f.Seek(0, 0)
	oldStates := map[string]file.State{}
	err = decoder.Decode(&oldStates)
	if err != nil {
		logp.Err("Error decoding old state: %+v", err)
		return false
	}

	// No old states found -> probably already new format
	if oldStates == nil {
		return false
	}

	// Convert old states to new states
	logp.Info("Old registry states found: %v", len(oldStates))
	states := convertOldStates(oldStates)
	r.states.SetStates(states)

	// Rewrite registry in new format
	r.writeRegistry()

	logp.Info("Old states converted to new states and written to registrar: %v", len(oldStates))

	return true
}

func convertOldStates(oldStates map[string]file.State) []file.State {
	// Convert old states to new states
	states := []file.State{}
	for _, state := range oldStates {
		// Makes timestamp time of migration, as this is the best guess
		state.Timestamp = time.Now()

		// Check for duplicates
		dupe := false
		for i, other := range states {
			if state.FileStateOS.IsSame(other.FileStateOS) {
				dupe = true
				if state.Offset > other.Offset {
					// replace other
					states[i] = state
					break
				}
			}
		}
		if !dupe {
			states = append(states, state)
		}
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
		var events []*input.Event

		select {
		case <-r.done:
			logp.Info("Ending Registrar")
			return
		case events = <-r.Channel:
		}

		r.processEventStates(events)

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
			r.out.Published(events)
		}
	}
}

// processEventStates gets the states from the events and writes them to the registrar state
func (r *Registrar) processEventStates(events []*input.Event) {
	logp.Debug("registrar", "Processing %d events", len(events))

	// Take the last event found for each file source
	for _, event := range events {

		// skip stdin
		if event.InputType == cfg.StdinInputType {
			continue
		}
		r.states.Update(event.State)
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
	f, err := os.Create(tempfile)
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

	err = file.SafeFileRotate(r.registryFile, tempfile)

	logp.Debug("registrar", "Registry file updated. %d states written.", len(states))
	registryWrites.Add(1)
	statesCurrent.Set(int64(len(states)))

	return err
}
