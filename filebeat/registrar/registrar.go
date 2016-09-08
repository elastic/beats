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
	. "github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/publisher"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

type Registrar struct {
	Channel      chan []*Event
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
		Channel:      make(chan []*Event, 1),
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
		return fmt.Errorf("Failed to created registry file dir %s: %v",
			registryPath, err)
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

	// Check if files exists
	_, err := os.Stat(r.registryFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Error means no file found
	if err != nil {
		logp.Info("No registry file found under: %s. Creating a new registry file.", r.registryFile)
		return nil
	}

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
		logp.Err("Error decoding states: %s", err)
		return err
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

	decoder := json.NewDecoder(f)
	oldStates := map[string]file.State{}
	err := decoder.Decode(&oldStates)

	if err != nil {
		logp.Debug("registrar", "Error decoding old state: %+v", err)
		return false
	}

	// No old states found -> probably already new format
	if oldStates == nil {
		return false
	}

	// Convert old states to new states
	states := make([]file.State, len(oldStates))
	logp.Info("Old registry states found: %v", len(oldStates))
	counter := 0
	for _, state := range oldStates {
		// Makes timestamp time of migration, as this is the best guess
		state.Timestamp = time.Now()
		states[counter] = state
		counter++
	}

	r.states.SetStates(states)

	// Rewrite registry in new format
	r.writeRegistry()

	logp.Info("Old states converted to new states and written to registrar: %v", len(oldStates))

	return true
}

func (r *Registrar) Start() error {

	// Load the previous log file locations now, for use in prospector
	err := r.loadStates()
	if err != nil {
		logp.Err("Error loading state: %v", err)
		return err
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
		var events []*Event

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
			"Registrar states cleaned up. Before: %d , After: %d",
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
func (r *Registrar) processEventStates(events []*Event) {
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
		logp.Err("Error when encoding the states: %s", err)
		return err
	}

	// Directly close file because of windows
	f.Close()

	logp.Debug("registrar", "Registry file updated. %d states written.", len(states))
	registryWrites.Add(1)
	statesCurrent.Set(int64(len(states)))

	return file.SafeFileRotate(r.registryFile, tempfile)
}
