package crawler

import (
	"fmt"
	"os"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
)

/*
 The hierarchy for the crawler objects is explained as following

 Crawler: Filebeat has one crawler. The crawler is the single point of control
 	and stores the state. The state is written through the registrar
 Prospector: For every FileConfig the crawler starts a prospector
 Harvester: For every file found inside the FileConfig, the Prospector starts a Harvester
 		The harvester send their events to the spooler
 		The spooler sends the event to the publisher
 		The publisher writes the state down with the registrar
*/

type Crawler struct {
	// Registrar object to persist the state
	Registrar *Registrar
	running   bool
}

func (crawler *Crawler) Start(prospectorConfigs []config.ProspectorConfig, eventChan chan *input.FileEvent) error {

	pendingProspectorCnt := 0
	crawler.running = true

	if len(prospectorConfigs) == 0 {
		return fmt.Errorf("No prospectors defined. You must have at least one prospector defined in the config file.")
	}

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, prospectorConfig := range prospectorConfigs {

		logp.Debug("prospector", "File Configs: %v", prospectorConfig.Paths)

		prospector := &Prospector{
			ProspectorConfig: prospectorConfig,
			registrar:        crawler.Registrar,
		}

		err := prospector.Init()
		if err != nil {
			logp.Critical("Error in initing prospector: %s", err)
			fmt.Printf("Error in initing prospector: %s", err)
			os.Exit(1)
		}

		go prospector.Run(eventChan)
		pendingProspectorCnt++
	}

	// Now determine which states we need to persist by pulling the events from the prospectors
	// When we hit a nil source a prospector had finished so we decrease the expected events
	logp.Debug("prospector", "Waiting for %d prospectors to initialise", pendingProspectorCnt)

	for event := range crawler.Registrar.Persist {
		if event.Source == nil {

			pendingProspectorCnt--
			if pendingProspectorCnt == 0 {
				logp.Debug("prospector", "No pending prospectors. Finishing setup")
				break
			}
			continue
		}
		crawler.Registrar.State[*event.Source] = event
		logp.Debug("prospector", "Registrar will re-save state for %s", *event.Source)

		if !crawler.running {
			break
		}
	}

	logp.Info("All prospectors initialised with %d states to persist", len(crawler.Registrar.State))

	return nil
}

func (crawler *Crawler) Stop() {
	// TODO: Properly stop prospectors and harvesters
}
