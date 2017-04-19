package crawler

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Crawler struct {
	prospectors       map[uint64]*prospector.Prospector
	prospectorConfigs []*common.Config
	out               prospector.Outlet
	wg                sync.WaitGroup
	reloader          *cfgfile.Reloader
	once              bool
	beatDone          chan struct{}
}

func New(out prospector.Outlet, prospectorConfigs []*common.Config, beatDone chan struct{}, once bool) (*Crawler, error) {

	return &Crawler{
		out:               out,
		prospectors:       map[uint64]*prospector.Prospector{},
		prospectorConfigs: prospectorConfigs,
		once:              once,
		beatDone:          beatDone,
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
		logp.Warn("BETA feature dynamic configuration reloading is enabled.")

		c.reloader = cfgfile.NewReloader(reloaderConfig)
		factory := prospector.NewFactory(c.out, r, c.beatDone)
		go func() {
			c.reloader.Run(factory)
		}()
	}

	logp.Info("Loading and starting Prospectors completed. Enabled prospectors: %v", len(c.prospectors))

	return nil
}

func (c *Crawler) startProspector(config *common.Config, states []file.State) error {
	if !config.Enabled() {
		return nil
	}
	p, err := prospector.NewProspector(config, c.out, c.beatDone)
	if err != nil {
		return fmt.Errorf("Error in initing prospector: %s", err)
	}
	p.Once = c.once

	if _, ok := c.prospectors[p.ID()]; ok {
		return fmt.Errorf("Prospector with same ID already exists: %v", p.ID())
	}

	err = p.LoadStates(states)
	if err != nil {
		return fmt.Errorf("error loading states for prospector %v: %v", p.ID(), err)
	}

	c.prospectors[p.ID()] = p

	p.Start()

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
