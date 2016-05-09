package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	. "github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

type Registrar struct {
	Channel      chan []*FileEvent
	done         chan struct{}
	registryFile string               // Path to the Registry File
	state        map[string]FileState // Map with all file paths inside and the corresponding state
	stateMutex   sync.Mutex
}

func NewRegistrar(registryFile string) (*Registrar, error) {

	r := &Registrar{
		registryFile: registryFile,
		done:         make(chan struct{}),
		state:        map[string]FileState{},
		Channel:      make(chan []*FileEvent, 1),
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
func (r *Registrar) LoadState() {
	if existing, e := os.Open(r.registryFile); e == nil {
		defer existing.Close()

		logp.Info("Loading registrar data from %s", r.registryFile)
		decoder := json.NewDecoder(existing)
		decoder.Decode(&r.state)
	}
}

func (r *Registrar) Run() {
	logp.Info("Starting Registrar")

	// Writes registry on shutdown
	defer r.writeRegistry()

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

		r.setState(event.Source, event.FileState)
	}
}

func (r *Registrar) Stop() {
	logp.Info("Stopping Registrar")
	close(r.done)
	// Note: don't block using waitGroup, cause this method is run by async signal handler
}

func (r *Registrar) GetFileState(path string) (FileState, bool) {
	state, exist := r.getStateEntry(path)
	return state, exist
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

	encoder := json.NewEncoder(file)

	state := r.getState()
	encoder.Encode(state)

	// Directly close file because of windows
	file.Close()

	logp.Info("Registry file updated. %d states written.", len(state))

	return SafeFileRotate(r.registryFile, tempfile)
}

func (r *Registrar) fetchState(filePath string, fileInfo os.FileInfo) int64 {

	if previous, err := r.getPreviousFile(filePath, fileInfo); err == nil {

		if previous != filePath {
			// File has rotated between shutdown and startup
			// We return last state downstream, with a modified event source with the new file name
			// And return the offset - also force harvest in case the file is old and we're about to skip it
			logp.Info("Detected rename of a previously harvested file: %s -> %s", previous, filePath)
		}

		logp.Info("Previous state for file %s found", filePath)

		lastState, _ := r.GetFileState(previous)
		return lastState.Offset
	}

	logp.Info("New file. Start reading from the beginning: %s", filePath)

	// New file so just start from the beginning
	return 0
}

// getPreviousFile checks in the registrar if there is the newFile already exist with a different name
// In case an old file is found, the path to the file is returned, if not, an error is returned
func (r *Registrar) getPreviousFile(newFilePath string, newFileInfo os.FileInfo) (string, error) {

	newState := input.GetOSFileState(newFileInfo)

	for oldFilePath, oldState := range r.getState() {

		// Compare states
		if newState.IsSame(oldState.FileStateOS) {
			logp.Info("Old file with new name found: %s -> %s", oldFilePath, newFilePath)
			return oldFilePath, nil
		}
	}

	return "", fmt.Errorf("No previous file found")
}

func (r *Registrar) setState(path string, state FileState) {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()

	r.state[path] = state
}

func (r *Registrar) getStateEntry(path string) (FileState, bool) {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()

	state, exist := r.state[path]
	return state, exist
}

func (r *Registrar) getState() map[string]FileState {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()

	copy := make(map[string]FileState)

	for k, v := range r.state {
		copy[k] = v
	}

	return copy
}
