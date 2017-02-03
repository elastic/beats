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
	reloader          *prospector.ProspectorReloader
	once              bool
}

func New(out prospector.Outlet, prospectorConfigs []*common.Config, once bool) (*Crawler, error) {

	return &Crawler{
		out:               out,
		prospectors:       map[uint64]*prospector.Prospector{},
		prospectorConfigs: prospectorConfigs,
		once:              once,
	}, nil
}

func (c *Crawler) Start(r *registrar.Registrar, reloaderConfig *common.Config) error {

	logp.Info("Loading Prospectors: %v", len(c.prospectorConfigs))

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, prospectorConfig := range c.prospectorConfigs {
		err := c.startProspector(prospectorConfig, r.GetStates())
		if err != nil {
			return err
		}
	}

	if reloaderConfig.Enabled() {
		logp.Warn("EXPERIMENTAL feature dynamic configuration reloading is enabled.")

		c.reloader = prospector.NewProspectorReloader(reloaderConfig, c.out, r)
		go func() {
			c.reloader.Run()
		}()
	}

	logp.Info("Loading and starting Prospectors completed. Enabled prospectors: %v", len(c.prospectors))

	return nil
}

func (c *Crawler) startProspector(config *common.Config, states []file.State) error {
	if !config.Enabled() {
		return nil
	}
	p, err := prospector.NewProspector(config, c.out)
	if err != nil {
		return fmt.Errorf("Error in initing prospector: %s", err)
	}
	p.Once = c.once

	if _, ok := c.prospectors[p.ID]; ok {
		return fmt.Errorf("Prospector with same ID already exists: %v", p.ID)
	}

	err = p.LoadStates(states)
	if err != nil {
		return fmt.Errorf("error loading states for propsector %v: %v", p.ID, err)
	}

	c.prospectors[p.ID] = p
	c.wg.Add(1)

	go func() {
		logp.Debug("crawler", "Starting prospector: %v", p.ID)
		defer logp.Debug("crawler", "Prospector stopped: %v", p.ID)

		defer c.wg.Done()
		p.Run()
	}()

	return nil
}

func (c *Crawler) Stop() {
	logp.Info("Stopping Crawler")

	asyncWaitStop := func(stop func()) {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			stop()
		}()
	}

	logp.Info("Stopping %v prospectors", len(c.prospectors))
	for _, p := range c.prospectors {
		// Stop prospectors in parallel
		asyncWaitStop(p.Stop)
	}

	if c.reloader != nil {
		asyncWaitStop(c.reloader.Stop)
	}

	c.WaitForCompletion()

	logp.Info("Crawler stopped")
}

func (c *Crawler) WaitForCompletion() {
	c.wg.Wait()
}
