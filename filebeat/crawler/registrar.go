package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	. "github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

type Registrar struct {
	// Path to the Registry File
	registryFile string
	// Map with all file paths inside and the corresponding state
	State map[string]*FileState
	// Channel used by the prospector and crawler to send FileStates to be persisted
	Persist chan *input.FileState

	Channel chan []*FileEvent
	done    chan struct{}
}

func NewRegistrar(registryFile string) (*Registrar, error) {

	r := &Registrar{
		registryFile: registryFile,
		done:         make(chan struct{}),
	}
	err := r.Init()

	return r, err
}

func (r *Registrar) Init() error {
	// Init state
	r.Persist = make(chan *FileState)
	r.State = make(map[string]*FileState)
	r.Channel = make(chan []*FileEvent, 1)

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
// The default file is .filebeat file which is stored in the same path as the binary is running
func (r *Registrar) LoadState() {
	if existing, e := os.Open(r.registryFile); e == nil {
		defer existing.Close()
		logp.Info("Loading registrar data from %s", r.registryFile)
		decoder := json.NewDecoder(existing)
		decoder.Decode(&r.State)
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
		// Treats new log files to persist with higher priority then new events
		case state := <-r.Persist:
			r.State[*state.Source] = state
			logp.Debug("prospector", "Registrar will re-save state for %s", *state.Source)
		case events := <-r.Channel:
			r.processEvents(events)
		}

		if e := r.writeRegistry(); e != nil {
			// REVU: but we should panic, or something, right?
			logp.Err("Writing of registry returned error: %v. Continuing..", e)
		}
	}
}

func (r *Registrar) processEvents(events []*FileEvent) {
	logp.Debug("registrar", "Processing %d events", len(events))

	// Take the last event found for each file source
	for _, event := range events {

		// skip stdin
		if event.InputType == cfg.StdinInputType {
			continue
		}

		r.State[*event.Source] = event.GetState()
	}
}

func (r *Registrar) Stop() {
	logp.Info("Stopping Registrar")
	close(r.done)
	// Note: don't block using waitGroup, cause this method is run by async signal handler
}

func (r *Registrar) GetFileState(path string) (*FileState, bool) {
	state, exist := r.State[path]
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
	encoder.Encode(r.State)

	// Directly close file because of windows
	file.Close()

	logp.Info("Registry file updated. %d states written.", len(r.State))

	return SafeFileRotate(r.registryFile, tempfile)
}

func (r *Registrar) fetchState(filePath string, fileInfo os.FileInfo) (int64, bool) {

	// Check if there is a state for this file
	lastState, isFound := r.GetFileState(filePath)

	if isFound && input.IsSameFile(filePath, fileInfo) {
		logp.Debug("registrar", "Same file as before found. Fetch the state and persist it.")
		// We're resuming - throw the last state back downstream so we resave it
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		r.Persist <- lastState
		return lastState.Offset, true
	}

	if previous, err := r.getPreviousFile(filePath, fileInfo); err == nil {
		// File has rotated between shutdown and startup
		// We return last state downstream, with a modified event source with the new file name
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		logp.Info("Detected rename of a previously harvested file: %s -> %s", previous, filePath)

		lastState, _ := r.GetFileState(previous)
		lastState.Source = &filePath
		r.Persist <- lastState
		return lastState.Offset, true
	}

	if isFound {
		logp.Info("Not resuming rotated file: %s", filePath)
	}

	logp.Info("New file. Start reading from the beginning: %s", filePath)

	// New file so just start from the beginning
	return 0, false
}

// getPreviousFile checks in the registrar if there is the newFile already exist with a different name
// In case an old file is found, the path to the file is returned, if not, an error is returned
func (r *Registrar) getPreviousFile(newFilePath string, newFileInfo os.FileInfo) (string, error) {

	newState := input.GetOSFileState(&newFileInfo)

	for oldFilePath, oldState := range r.State {

		// Skipping when path the same
		if oldFilePath == newFilePath {
			continue
		}

		// Compare states
		if newState.IsSame(oldState.FileStateOS) {
			logp.Info("Old file with new name found: %s is no %s", oldFilePath, newFilePath)
			return oldFilePath, nil
		}
	}

	return "", fmt.Errorf("No previous file found")
}
