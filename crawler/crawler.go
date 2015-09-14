package crawler

import (
	"github.com/elastic/filebeat/input"

	cfg "github.com/elastic/filebeat/config"
	"github.com/elastic/libbeat/logp"
)

/*
 The hierarchy for the crawler objects is explained as following

 Crawler: Filebeat has one crawler. The crawler is the single point of control and stores the state. The state is written through the registrar
 Prospector: For every FileConfig the crawler starts a prospector
 Harvestor: For every file found inside the FileConfig, the Prospector starts a Harvestor
 		The harvestor send their events to the spooler
 		The spooler sends the event to the publisher
 		The publisher writes the state down with the registrar
*/

// Last reading state of the prospector
type Crawler struct {
	Files   map[string]*input.FileState
	Persist chan *input.FileState
}

func (crawler *Crawler) Start(files []cfg.FileConfig, persist map[string]*input.FileState, eventChan chan *input.FileEvent) {
	pendingProspectorCnt := 0

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, fileconfig := range files {

		logp.Debug("prospector", "File Config:", fileconfig)

		prospector := &Prospector{
			FileConfig: fileconfig,
			crawler:    crawler,
		}
		go prospector.Prospect(eventChan)
		pendingProspectorCnt++
	}

	// Now determine which states we need to persist by pulling the events from the prospectors
	// When we hit a nil source a prospector had finished so we decrease the expected events
	logp.Debug("prospector", "Waiting for %d prospectors to initialise", pendingProspectorCnt)

	for event := range crawler.Persist {
		if event.Source == nil {
			pendingProspectorCnt--
			if pendingProspectorCnt == 0 {
				break
			}
			continue
		}
		persist[*event.Source] = event
		logp.Debug("prospector", "Registrar will re-save state for %s", *event.Source)
	}

	logp.Info("All prospectors initialised with %d states to persist", len(persist))
}

func (crawler *Cralwer) Stop() {
	// TODO: To be implemented for proper shutdown
}
