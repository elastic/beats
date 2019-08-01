// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/elastic/fleet/x-pack/pkg/agent/program"
	"github.com/elastic/fleet/x-pack/pkg/agent/transpiler"
	"github.com/elastic/fleet/x-pack/pkg/bus"
	"github.com/elastic/fleet/x-pack/pkg/bus/events"
	"github.com/elastic/fleet/x-pack/pkg/bus/topic"
	"github.com/elastic/fleet/x-pack/pkg/core/logger"
)

// ConfigReader points the configuration to a local file on disk, everything that we call load,
// it will go to disk to read the current configuration and return a new configuration object.
type configReader struct {
	log            *logger.Logger
	pathConfigFile string
	defaultsFunc   func() *Config
}

// NewConfigReader returns a new ConfigReader.
func newConfigReader(
	log *logger.Logger,
	pathConfigFile string,
) *configReader {
	return &configReader{log: log, pathConfigFile: pathConfigFile}
}

// Load is loading the configuration from disk and returning a new config object.
func (c *configReader) load() (map[string]interface{}, error) {
	c.log.Debugf("Reading configuration file: %s", c.pathConfigFile)
	// TODO(ph) normalize configuration and hook into ucfg.

	b, err := ioutil.ReadFile(c.pathConfigFile)
	if err != nil {
		return nil, fmt.Errorf("could not read configuration file, error: %+v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("could not read YAML, error: %+v", err)
	}
	return m, nil
}

// OnlyOnceConfigSource reads the configuration once and send it to the bus, if and errors happens
// we will not retry and instead we will bubble the errors and agent should terminate.
type onlyOnceConfigSource struct {
	log          *logger.Logger
	configReader *configReader
	bus          bus.Bus
	outTopic     topic.Topic
}

func newOnlyOnceConfigSource(
	log *logger.Logger,
	configReader *configReader,
	bus bus.Bus,
	outTopic topic.Topic,
) *onlyOnceConfigSource {
	return &onlyOnceConfigSource{
		log:          log,
		configReader: configReader,
		bus:          bus,
		outTopic:     outTopic,
	}
}

func (o *onlyOnceConfigSource) start() error {
	o.log.Debug("Reading configuration once")
	raw, err := o.configReader.load()
	if err != nil {
		return err
	}

	o.log.Debug("Transforming configuration into a tree")
	fmt.Println(raw)
	ast, err := transpiler.NewAST(raw)
	if err != nil {
		return err
	}

	o.log.Debugf("Supported programs: %s", strings.Join(program.KnownProgramNames(), ", "))
	o.log.Debug("Converting single configuration into specific programs configuration")
	programsToRun, err := program.Programs(ast)
	if err != nil {
		return err
	}

	for _, program := range programsToRun {
		o.log.Debugf("Need to run program %v", program)
		event, err := events.NewConfigChanged(program)
		if err != nil {
			return err
		}
		_, err = o.bus.Push(o.outTopic, event)
	}

	return nil
}

func (o *onlyOnceConfigSource) stop() error {
	return nil
}
