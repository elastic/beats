package beat

import (
	"encoding/json"
	"os"

	cfg "github.com/elastic/filebeat/config"
	. "github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
)

type Registrar struct {
	registryFile string
}

func (r *Registrar) Init() {
	// Set to default in case it is not set
	if r.registryFile == "" {
		r.registryFile = cfg.DefaultRegistryFile
	}

	logp.Debug("registrar", "Registry file set to: %s", r.registryFile)

}

// loadState fetches the previous reading state from the configure registryFile file
// The default file is .filebeat file which is stored in the same path as the binary is running
func (r *Registrar) LoadState(files map[string]*FileState) {

	if existing, e := os.Open(r.registryFile); e == nil {
		defer existing.Close()
		wd := ""
		if wd, e = os.Getwd(); e != nil {
			logp.Warn("WARNING: os.Getwd retuned unexpected error %s -- ignoring", e.Error())
		}
		logp.Info("Loading registrar data from %s/%s", wd, r.registryFile)

		decoder := json.NewDecoder(existing)
		decoder.Decode(&files)
	}
}

func (r *Registrar) WriteState(state map[string]*FileState, input chan []*FileEvent) {
	logp.Debug("registrar", "Starting Registrar")
	for events := range input {
		logp.Debug("registrar", "Registrar: processing %d events", len(events))
		// Take the last event found for each file source
		for _, event := range events {
			// skip stdin
			if *event.Source == "-" {
				continue
			}

			state[*event.Source] = event.GetState()
		}

		if e := r.writeRegistry(state); e != nil {
			// REVU: but we should panic, or something, right?
			logp.Warn("WARNING: (continuing) update of registry returned error: %s", e)
		}
	}
	logp.Debug("registrar", "Ending Registrar")
}

// writeRegistry Writes the new json registry file  to disk
func (r *Registrar) writeRegistry(state map[string]*FileState) error {
	logp.Debug("registrar", "Write registry file: %s", r.registryFile)

	tempfile := r.registryFile + ".new"
	file, e := os.Create(tempfile)
	if e != nil {
		logp.Err("Failed to create tempfile (%s) for writing: %s", tempfile, e)
		return e
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(state)

	return SafeFileRotate(r.registryFile, tempfile)
}
