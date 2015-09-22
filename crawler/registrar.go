package crawler

import (
	"encoding/json"
	"os"

	cfg "github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
	. "github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
)

type Registrar struct {
	// Path to the Registry File
	registryFile string
	// Map with all file paths inside and the corresponding state
	State map[string]*FileState
	// Channel used by the prospector and crawler to send FileStates to be persisted
	Persist chan *input.FileState
}

func NewRegistrar(registryFile string) *Registrar {

	r := &Registrar{
		registryFile: registryFile,
	}
	r.Init()

	return r
}

func (r *Registrar) Init() {
	// Init state
	r.Persist = make(chan *FileState)
	r.State = make(map[string]*FileState)

	// Set to default in case it is not set
	if r.registryFile == "" {
		r.registryFile = cfg.DefaultRegistryFile
	}

	logp.Debug("registrar", "Registry file set to: %s", r.registryFile)
}

// loadState fetches the previous reading state from the configure RegistryFile file
// The default file is .filebeat file which is stored in the same path as the binary is running
func (r *Registrar) LoadState() {

	if existing, e := os.Open(r.registryFile); e == nil {
		defer existing.Close()
		wd := ""
		if wd, e = os.Getwd(); e != nil {
			logp.Warn("WARNING: os.Getwd retuned unexpected error %s -- ignoring", e.Error())
		}
		logp.Info("Loading registrar data from %s/%s", wd, r.registryFile)

		decoder := json.NewDecoder(existing)
		decoder.Decode(&r.State)
	}
}

func (r *Registrar) WriteState(input chan []*FileEvent) {
	logp.Debug("registrar", "Starting Registrar")
	for events := range input {
		logp.Debug("registrar", "Registrar: processing %d events", len(events))
		// Take the last event found for each file source
		for _, event := range events {
			// skip stdin
			if *event.Source == "-" {
				continue
			}

			r.State[*event.Source] = event.GetState()
		}

		if e := r.writeRegistry(); e != nil {
			// REVU: but we should panic, or something, right?
			logp.Err("Update of registry returned error: %v. Continuing..", e)
		}
	}
	logp.Debug("registrar", "Ending Registrar")
}

func (r *Registrar) GetFileState(path string) (*FileState, bool) {
	state, exist := r.State[path]
	return state, exist
}

// writeRegistry Writes the new json registry file  to disk
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

	return SafeFileRotate(r.registryFile, tempfile)
}

func (r *Registrar) fetchState(filePath string, fileInfo os.FileInfo) (int64, bool) {

	// Check if there is a state for this file
	lastState, isFound := r.GetFileState(filePath)

	if isFound && input.IsSameFile(filePath, fileInfo) {
		// We're resuming - throw the last state back downstream so we resave it
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		r.Persist <- lastState
		return lastState.Offset, true
	}

	if previous := r.getPreviousFile(filePath, fileInfo); previous != "" {
		// File has rotated between shutdown and startup
		// We return last state downstream, with a modified event source with the new file name
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		logp.Debug("prospector", "Detected rename of a previously harvested file: %s -> %s", previous, filePath)

		lastState, _ := r.GetFileState(previous)
		lastState.Source = &filePath
		r.Persist <- lastState
		return lastState.Offset, true
	}

	if isFound {
		logp.Debug("prospector", "Not resuming rotated file: %s", filePath)
	}

	// New file so just start from an automatic position
	return 0, false
}
