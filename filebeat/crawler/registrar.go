package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	. "github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

type Registrar struct {
	Channel      chan []*FileEvent
	done         chan struct{}
	registryFile string        // Path to the Registry File
	state        *input.States // Map with all file paths inside and the corresponding state
	wg           sync.WaitGroup
}

func NewRegistrar(registryFile string) (*Registrar, error) {

	r := &Registrar{
		registryFile: registryFile,
		done:         make(chan struct{}),
		state:        input.NewStates(),
		Channel:      make(chan []*FileEvent, 1),
		wg:           sync.WaitGroup{},
	}
	err := r.Init()

	return r, err
}

// Init sets up the Registrar and make sure the registry file is setup correctly
func (r *Registrar) Init() error {

	// Set to default in case it is not set
	if r.registryFile == "" {
		r.registryFile = cfg.DefaultRegistryFile
	}

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

// loadState fetches the previous reading state from the configure RegistryFile file
// The default file is `registry` in the data path.
func (r *Registrar) LoadState() error {

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

	file, err := os.Open(r.registryFile)
	if err != nil {
		return err
	}

	defer file.Close()

	logp.Info("Loading registrar data from %s", r.registryFile)

	// DEPRECATED: This should be removed in 6.0
	oldStates := r.loadAndConvertOldState(file)
	if oldStates {
		return nil
	}

	decoder := json.NewDecoder(file)
	states := []input.FileState{}
	decoder.Decode(&states)

	r.state.SetStates(states)
	logp.Info("States Loaded from registrar: %+v", len(states))

	return nil
}

// loadAndConvertOldState loads the old state file and converts it to the new state
// This is designed so it can be easily removed in later versions
func (r *Registrar) loadAndConvertOldState(file *os.File) bool {
	// Make sure file reader is reset afterwards
	defer file.Seek(0, 0)

	decoder := json.NewDecoder(file)
	oldStates := map[string]FileState{}
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
	states := make([]input.FileState, len(oldStates))
	logp.Info("Old registry states found: %v", len(oldStates))
	counter := 0
	for _, state := range oldStates {
		// Makes time last_seen time of migration, as this is the best guess
		state.LastSeen = time.Now()
		states[counter] = state
		counter++
	}

	r.state.SetStates(states)

	// Rewrite registry in new format
	r.writeRegistry()

	logp.Info("Old states converted to new states and written to registrar: %v", len(oldStates))

	return true
}

func (r *Registrar) Start() {
	r.wg.Add(1)
	go r.Run()
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
		case events := <-r.Channel:
			r.processEventStates(events)
		}

		if e := r.writeRegistry(); e != nil {
			logp.Err("Writing of registry returned error: %v. Continuing..", e)
		}
	}
}

// processEventStates gets the states from the events and writes them to the registrar state
func (r *Registrar) processEventStates(events []*FileEvent) {
	logp.Debug("registrar", "Processing %d events", len(events))

	// Take the last event found for each file source
	for _, event := range events {

		// skip stdin
		if event.InputType == cfg.StdinInputType {
			continue
		}
		r.state.Update(event.FileState)
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
	file, e := os.Create(tempfile)
	if e != nil {
		logp.Err("Failed to create tempfile (%s) for writing: %s", tempfile, e)
		return e
	}

	states := r.state.GetStates()

	encoder := json.NewEncoder(file)
	encoder.Encode(states)

	// Directly close file because of windows
	file.Close()

	logp.Info("Registry file updated. %d states written.", len(states))

	return SafeFileRotate(r.registryFile, tempfile)
}
