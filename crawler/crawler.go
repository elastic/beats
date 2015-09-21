package crawler

import (
	"fmt"
	"os"

	"github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
)

/*
 The hierarchy for the crawler objects is explained as following

 Crawler: Filebeat has one crawler. The crawler is the single point of control
 	and stores the state. The state is written through the registrar
 Prospector: For every FileConfig the crawler starts a prospector
 Harvestor: For every file found inside the FileConfig, the Prospector starts a Harvestor
 		The harvester send their events to the spooler
 		The spooler sends the event to the publisher
 		The publisher writes the state down with the registrar
*/

type Crawler struct {
	// List of all files which were crawled with the state
	//Files map[string]*input.FileState
	// TODO: Better explanation and potential renaming needed here for what this variable is.
	Registrar *Registrar
}

func (crawler *Crawler) Start(files []config.ProspectorConfig, eventChan chan *input.FileEvent) {

	pendingProspectorCnt := 0

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, fileconfig := range files {

		logp.Debug("prospector", "File Configs: %v", fileconfig.Paths)

		prospector := &Prospector{
			ProspectorConfig: fileconfig,
			crawler:          crawler,
		}

		err := prospector.Init()
		if err != nil {
			logp.Critical("Error in initing propsecptor: %s", err)
			fmt.Printf("Error in initing propsecptor: %s", err)
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
				break
			}
			continue
		}
		crawler.Registrar.State[*event.Source] = event
		logp.Debug("prospector", "Registrar will re-save state for %s", *event.Source)
	}

	logp.Info("All prospectors initialised with %d states to persist", len(crawler.Registrar.State))
}

func (crawler *Crawler) Stop() {
	// TODO: To be implemented for proper shutdown
}
