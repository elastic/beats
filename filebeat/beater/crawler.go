// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beater

import (
	"fmt"
	"sync"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type crawler struct {
	log             *logp.Logger
	inputs          map[uint64]cfgfile.Runner
	inputConfigs    []*common.Config
	wg              sync.WaitGroup
	inputsFactory   cfgfile.RunnerFactory
	modulesFactory  cfgfile.RunnerFactory
	modulesReloader *cfgfile.Reloader
	inputReloader   *cfgfile.Reloader
	once            bool
	beatDone        chan struct{}
}

func newCrawler(
	inputFactory, module cfgfile.RunnerFactory,
	inputConfigs []*common.Config,
	beatDone chan struct{},
	once bool,
) (*crawler, error) {
	return &crawler{
		log:            logp.NewLogger("crawler"),
		inputs:         map[uint64]cfgfile.Runner{},
		inputsFactory:  inputFactory,
		modulesFactory: module,
		inputConfigs:   inputConfigs,
		once:           once,
		beatDone:       beatDone,
	}, nil
}

// Start starts the crawler with all inputs
func (c *crawler) Start(
	pipeline beat.PipelineConnector,
	configInputs *common.Config,
	configModules *common.Config,
) error {
	log := c.log

	log.Infof("Loading Inputs: %d", len(c.inputConfigs))

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, inputConfig := range c.inputConfigs {
		err := c.startInput(pipeline, inputConfig)
		if err != nil {
			return fmt.Errorf("starting input failed: %+v", err)
		}
	}

	if configInputs.Enabled() {
		c.inputReloader = cfgfile.NewReloader(pipeline, configInputs)
		if err := c.inputReloader.Check(c.inputsFactory); err != nil {
			return fmt.Errorf("creating input reloader failed: %+v", err)
		}
	}

	if configModules.Enabled() {
		c.modulesReloader = cfgfile.NewReloader(pipeline, configModules)
		if err := c.modulesReloader.Check(c.modulesFactory); err != nil {
			return fmt.Errorf("creating module reloader failed: %+v", err)
		}
	}

	if c.inputReloader != nil {
		go func() {
			c.inputReloader.Run(c.inputsFactory)
		}()
	}
	if c.modulesReloader != nil {
		go func() {
			c.modulesReloader.Run(c.modulesFactory)
		}()
	}

	log.Infof("Loading and starting Inputs completed. Enabled inputs: %d", len(c.inputs))

	return nil
}

func (c *crawler) startInput(
	pipeline beat.PipelineConnector,
	config *common.Config,
) error {
	if !config.Enabled() {
		c.log.Infof("input disabled, skipping it")
		return nil
	}

	var h map[string]interface{}
	err := config.Unpack(&h)
	if err != nil {
		return fmt.Errorf("could not unpack config: %w", err)
	}
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return fmt.Errorf("can not compute id from configuration: %w", err)
	}
	if _, ok := c.inputs[id]; ok {
		return fmt.Errorf("input with same ID already exists: %d", id)
	}

	runner, err := c.inputsFactory.Create(pipeline, config)
	if err != nil {
		return fmt.Errorf("error while initializing input: %w", err)
	}
	if inputRunner, ok := runner.(*input.Runner); ok {
		inputRunner.Once = c.once
	}

	c.inputs[id] = runner

	c.log.Infof("Starting input (ID: %d)", id)
	runner.Start()

	return nil
}

func (c *crawler) Stop() {
	logp.Info("Stopping Crawler")

	asyncWaitStop := func(stop func()) {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			stop()
		}()
	}

	logp.Info("Stopping %d inputs", len(c.inputs))
	// Stop inputs in parallel
	for id, p := range c.inputs {
		id, p := id, p
		asyncWaitStop(func() {
			c.log.Infof("Stopping input: %d", id)
			p.Stop()
		})
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

func (c *crawler) WaitForCompletion() {
	c.wg.Wait()
}
