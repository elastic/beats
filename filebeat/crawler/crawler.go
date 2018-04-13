package crawler

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/fileset"
	"github.com/elastic/beats/filebeat/input/file"
	input "github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	_ "github.com/elastic/beats/filebeat/include"
)

type Crawler struct {
	inputs          map[uint64]*input.Runner
	inputConfigs    []*common.Config
	out             channel.Factory
	wg              sync.WaitGroup
	InputsFactory   cfgfile.RunnerFactory
	ModulesFactory  cfgfile.RunnerFactory
	modulesReloader *cfgfile.Reloader
	inputReloader   *cfgfile.Reloader
	once            bool
	beatVersion     string
	beatDone        chan struct{}
}

func New(out channel.Factory, inputConfigs []*common.Config, beatVersion string, beatDone chan struct{}, once bool) (*Crawler, error) {
	return &Crawler{
		out:          out,
		inputs:       map[uint64]*input.Runner{},
		inputConfigs: inputConfigs,
		once:         once,
		beatVersion:  beatVersion,
		beatDone:     beatDone,
	}, nil
}

// Start starts the crawler with all inputs
func (c *Crawler) Start(r *registrar.Registrar, configInputs *common.Config,
	configModules *common.Config, pipelineLoaderFactory fileset.PipelineLoaderFactory, overwritePipelines bool) error {

	logp.Info("Loading Inputs: %v", len(c.inputConfigs))

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, inputConfig := range c.inputConfigs {
		err := c.startInput(inputConfig, r.GetStates())
		if err != nil {
			return err
		}
	}

	c.InputsFactory = input.NewRunnerFactory(c.out, r, c.beatDone)
	if configInputs.Enabled() {
		c.inputReloader = cfgfile.NewReloader(configInputs)
		if err := c.inputReloader.Check(c.InputsFactory); err != nil {
			return err
		}

		go func() {
			c.inputReloader.Run(c.InputsFactory)
		}()
	}

	c.ModulesFactory = fileset.NewFactory(c.out, r, c.beatVersion, pipelineLoaderFactory, overwritePipelines, c.beatDone)
	if configModules.Enabled() {
		c.modulesReloader = cfgfile.NewReloader(configModules)
		if err := c.modulesReloader.Check(c.ModulesFactory); err != nil {
			return err
		}

		go func() {
			c.modulesReloader.Run(c.ModulesFactory)
		}()
	}

	logp.Info("Loading and starting Inputs completed. Enabled inputs: %v", len(c.inputs))

	return nil
}

func (c *Crawler) startInput(config *common.Config, states []file.State) error {
	if !config.Enabled() {
		return nil
	}
	p, err := input.New(config, c.out, c.beatDone, states, nil)
	if err != nil {
		return fmt.Errorf("Error in initing input: %s", err)
	}
	p.Once = c.once

	if _, ok := c.inputs[p.ID]; ok {
		return fmt.Errorf("Input with same ID already exists: %d", p.ID)
	}

	c.inputs[p.ID] = p

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

	logp.Info("Stopping %v inputs", len(c.inputs))
	for _, p := range c.inputs {
		// Stop inputs in parallel
		asyncWaitStop(p.Stop)
	}

	if c.inputReloader != nil {
		asyncWaitStop(c.inputReloader.Stop)
	}

	if c.modulesReloader != nil {
		asyncWaitStop(c.modulesReloader.Stop)
	}

	c.WaitForCompletion()

	logp.Info("Crawler stopped")
}

func (c *Crawler) WaitForCompletion() {
	c.wg.Wait()
}
