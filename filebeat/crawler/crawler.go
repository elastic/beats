package crawler

import (
	"fmt"
	"sync"

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
	Registrar   *Registrar
	prospectors []*Prospector
	wg          sync.WaitGroup
}

func (c *Crawler) Start(prospectorConfigs []config.ProspectorConfig, eventChan chan *input.FileEvent) error {

	if len(prospectorConfigs) == 0 {
		return fmt.Errorf("No prospectors defined. You must have at least one prospector defined in the config file.")
	}

	logp.Info("Loading Prospectors: %v", len(prospectorConfigs))

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, prospectorConfig := range prospectorConfigs {

		logp.Debug("prospector", "File Configs: %v", prospectorConfig.Paths)

		prospector, err := NewProspector(prospectorConfig, c.Registrar, eventChan)
		if err != nil {
			return fmt.Errorf("Error in initing prospector: %s", err)
		}
		c.prospectors = append(c.prospectors, prospector)
	}

	logp.Info("Loading Prospectors completed. Number of prospectors: %v", len(c.prospectors))

	c.wg = sync.WaitGroup{}
	for _, prospector := range c.prospectors {
		c.wg.Add(1)
		go prospector.Run(&c.wg)
	}

	logp.Info("All prospectors are initialised and running with %d states to persist", len(c.Registrar.getStateCopy()))

	return nil
}

func (c *Crawler) Stop() {
	logp.Info("Stopping Crawler")

	logp.Info("Stopping %v prospectors", len(c.prospectors))
	for _, prospector := range c.prospectors {
		prospector.Stop()
	}
	c.wg.Wait()
	logp.Info("Crawler stopped")
}
