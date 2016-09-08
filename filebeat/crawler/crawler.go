package crawler

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Crawler struct {
	prospectors       []*prospector.Prospector
	prospectorConfigs []*common.Config
	out               prospector.Outlet
	wg                sync.WaitGroup
}

func New(out prospector.Outlet, prospectorConfigs []*common.Config) (*Crawler, error) {

	if len(prospectorConfigs) == 0 {
		return nil, fmt.Errorf("No prospectors defined. You must have at least one prospector defined in the config file.")
	}

	return &Crawler{
		out:               out,
		prospectorConfigs: prospectorConfigs,
	}, nil
}

func (c *Crawler) Start(states file.States) error {

	logp.Info("Loading Prospectors: %v", len(c.prospectorConfigs))

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, prospectorConfig := range c.prospectorConfigs {

		prospector, err := prospector.NewProspector(prospectorConfig, states, c.out)
		if err != nil {
			return fmt.Errorf("Error in initing prospector: %s", err)
		}
		c.prospectors = append(c.prospectors, prospector)
	}

	logp.Info("Loading Prospectors completed. Number of prospectors: %v", len(c.prospectors))

	for i, p := range c.prospectors {
		c.wg.Add(1)

		go func(id int, prospector *prospector.Prospector) {
			defer func() {
				c.wg.Done()
				logp.Debug("crawler", "Prospector %v stopped", id)
			}()
			logp.Debug("crawler", "Starting prospector %v", id)
			prospector.Run()
		}(i, p)
	}

	logp.Info("All prospectors are initialised and running with %d states to persist", states.Count())

	return nil
}

func (c *Crawler) Stop() {
	logp.Info("Stopping Crawler")
	stopProspector := func(p *prospector.Prospector) {
		defer c.wg.Done()
		p.Stop()
	}

	logp.Info("Stopping %v prospectors", len(c.prospectors))
	for _, p := range c.prospectors {
		// Stop prospectors in parallel
		c.wg.Add(1)
		go stopProspector(p)
	}
	c.wg.Wait()
	logp.Info("Crawler stopped")
}
