package beat

import (
	"encoding/json"
	. "github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
	"os"
)

func Registrar(state map[string]*FileState, input chan []*FileEvent) {
	logp.Info("registrar", "Starting Registrar")
	for events := range input {
		logp.Info("registrar", "Registrar: processing %d events", len(events))
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
			logp.Info("registrar", "WARNING: (continuing) update of registry returned error: %s", e)
		}
	}
	logp.Info("registrar", "Ending Registrar")
}

// writeRegistry Writes the new json registry file  to disk
func writeRegistry(state map[string]*FileState, path string) error {
	tempfile := path + ".new"
	file, e := os.Create(tempfile)
	if e != nil {
		logp.Info("registrar", "Failed to create tempfile (%s) for writing: %s", tempfile, e)
		return e
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(state)

	return SafeFileRotate(path, tempfile)
}
