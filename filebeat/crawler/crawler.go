package crawler

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Crawler struct {
	prospectors       map[uint64]*prospector.Prospector
	prospectorConfigs []*common.Config
	out               prospector.Outlet
	wg                sync.WaitGroup
	once              bool
}

func New(out prospector.Outlet, prospectorConfigs []*common.Config, once bool) (*Crawler, error) {

	if len(prospectorConfigs) == 0 {
		return nil, fmt.Errorf("No prospectors defined. You must have at least one prospector defined in the config file.")
	}

	return &Crawler{
		out:               out,
		prospectors:       map[uint64]*prospector.Prospector{},
		prospectorConfigs: prospectorConfigs,
		once:              once,
	}, nil
}

func (c *Crawler) Start(r *registrar.Registrar) error {

	logp.Info("Loading Prospectors: %v", len(c.prospectorConfigs))

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, prospectorConfig := range c.prospectorConfigs {
		err := c.startProspector(prospectorConfig, r.GetStates())
		if err != nil {
			return err
		}
	}

	logp.Info("Loading and starting Prospectors completed. Enabled prospectors: %v", len(c.prospectors))

	return nil
}

func (c *Crawler) startProspector(config *common.Config, states []file.State) error {
	if !config.Enabled() {
		return nil
	}
	prospector, err := prospector.NewProspector(config, states, c.out)
	if err != nil {
		return fmt.Errorf("Error in initing prospector: %s", err)
	}
	prospector.Once = c.once

	if _, ok := c.prospectors[prospector.ID]; ok {
		return fmt.Errorf("Prospector with same ID already exists: %v", prospector.ID)
	}

	c.prospectors[prospector.ID] = prospector
	c.wg.Add(1)

	go func() {
		logp.Debug("crawler", "Starting prospector: %v", prospector.ID)
		defer logp.Debug("crawler", "Prospector stopped: %v", prospector.ID)

		defer c.wg.Done()
		prospector.Run()
	}()

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
	c.WaitForCompletion()
	logp.Info("Crawler stopped")
}

func (c *Crawler) WaitForCompletion() {
	c.wg.Wait()
}
