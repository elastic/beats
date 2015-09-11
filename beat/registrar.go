package beat

import (
	"encoding/json"
	"os"

	. "github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
)

func Registrar(state map[string]*FileState, input chan []*FileEvent) {
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

		if e := writeRegistry(state, ".filebeat"); e != nil {
			// REVU: but we should panic, or something, right?
			logp.Warn("WARNING: (continuing) update of registry returned error: %s", e)
		}
	}
	logp.Debug("registrar", "Ending Registrar")
}

// writeRegistry Writes the new json registry file  to disk
func writeRegistry(state map[string]*FileState, path string) error {
	tempfile := path + ".new"
	file, e := os.Create(tempfile)
	if e != nil {
		logp.Err("Failed to create tempfile (%s) for writing: %s", tempfile, e)
		return e
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(state)

	return SafeFileRotate(path, tempfile)
}
