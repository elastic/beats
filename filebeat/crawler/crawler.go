package crawler

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
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

func (c *Crawler) Start(prospectorConfigs []*common.Config, eventChan chan *input.FileEvent) error {

	if len(prospectorConfigs) == 0 {
		return fmt.Errorf("No prospectors defined. You must have at least one prospector defined in the config file.")
	}

	logp.Info("Loading Prospectors: %v", len(prospectorConfigs))

	// Get existing states
	states := *c.Registrar.state

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, prospectorConfig := range prospectorConfigs {

		prospector, err := NewProspector(prospectorConfig, states, eventChan)
		if err != nil {
			return fmt.Errorf("Error in initing prospector: %s", err)
		}
		c.prospectors = append(c.prospectors, prospector)
	}

	logp.Info("Loading Prospectors completed. Number of prospectors: %v", len(c.prospectors))

	c.wg = sync.WaitGroup{}
	for i, p := range c.prospectors {
		c.wg.Add(1)

		go func(id int, prospector *Prospector) {
			defer func() {
				c.wg.Done()
				logp.Debug("crawler", "Prospector %v stopped", id)
			}()
			logp.Debug("crawler", "Starting prospector %v", id)
			prospector.Run()
		}(i, p)
	}

	logp.Info("All prospectors are initialised and running with %d states to persist", c.Registrar.state.Count())

	return nil
}

func (c *Crawler) Stop() {
	logp.Info("Stopping Crawler")

	logp.Info("Stopping %v prospectors", len(c.prospectors))
	for _, prospector := range c.prospectors {
		// Stop prospectors in parallel
		c.wg.Add(1)
		go prospector.Stop(&c.wg)
	}
	c.wg.Wait()
	logp.Info("Crawler stopped")
}
