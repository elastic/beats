// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filters"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/noop"
)

// InspectOutputCmd is an inspect subcommand that shows configurations of the agent.
type InspectOutputCmd struct {
	cfgPath string
	output  string
	program string
}

// NewInspectOutputCmd creates a new inspect command.
func NewInspectOutputCmd(configPath, output, program string) (*InspectOutputCmd, error) {
	return &InspectOutputCmd{
		cfgPath: configPath,
		output:  output,
		program: program,
	}, nil
}

// Execute tries to enroll the agent into Fleet.
func (c *InspectOutputCmd) Execute() error {
	if c.output == "" {
		return c.inspectOutputs()
	}

	return c.inspectOutput()
}

func (c *InspectOutputCmd) inspectOutputs() error {
	rawConfig, err := loadConfig(c.cfgPath)
	if err != nil {
		return err
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return err
	}

	l, err := newErrorLogger()
	if err != nil {
		return err
	}

	if isStandalone(cfg.Fleet) {
		return listOutputsFromConfig(l, rawConfig)
	}

	fleetConfig, err := loadFleetConfig(rawConfig)
	if err != nil {
		return err
	} else if fleetConfig == nil {
		return fmt.Errorf("no fleet config retrieved yet")
	}

	return listOutputsFromMap(l, fleetConfig)
}

func listOutputsFromConfig(log *logger.Logger, cfg *config.Config) error {
	programsGroup, err := getProgramsFromConfig(log, cfg)
	if err != nil {
		return err

	}

	for k := range programsGroup {
		fmt.Println(k)
	}

	return nil
}

func listOutputsFromMap(log *logger.Logger, cfg map[string]interface{}) error {
	c, err := config.NewConfigFrom(cfg)
	if err != nil {
		return err
	}

	return listOutputsFromConfig(log, c)
}

func (c *InspectOutputCmd) inspectOutput() error {
	rawConfig, err := loadConfig(c.cfgPath)
	if err != nil {
		return err
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return err
	}

	l, err := newErrorLogger()
	if err != nil {
		return err
	}

	if isStandalone(cfg.Fleet) {
		return printOutputFromConfig(l, c.output, c.program, rawConfig)
	}

	fleetConfig, err := loadFleetConfig(rawConfig)
	if err != nil {
		return err
	} else if fleetConfig == nil {
		return fmt.Errorf("no fleet config retrieved yet")
	}

	return printOutputFromMap(l, c.output, c.program, fleetConfig)
}

func printOutputFromConfig(log *logger.Logger, output, programName string, cfg *config.Config) error {
	programsGroup, err := getProgramsFromConfig(log, cfg)
	if err != nil {
		return err

	}

	for k, programs := range programsGroup {
		if k != output {
			continue
		}

		var programFound bool
		for _, p := range programs {
			if programName != "" && programName != p.Spec.Cmd {
				continue
			}

			programFound = true
			fmt.Printf("[%s] %s:\n", k, p.Spec.Cmd)
			printMapStringConfig(p.Configuration())
			fmt.Println("---")
		}

		if !programFound {
			return fmt.Errorf("program '%s' is not recognized within output '%s', try running `elastic-agent inspect output` to find available outputs",
				programName,
				output)
		}

		return nil
	}

	return fmt.Errorf("output '%s' is not recognized, try running `elastic-agent inspect output` to find available outputs", output)

}

func printOutputFromMap(log *logger.Logger, output, programName string, cfg map[string]interface{}) error {
	c, err := config.NewConfigFrom(cfg)
	if err != nil {
		return err
	}

	return printOutputFromConfig(log, output, programName, c)
}

func getProgramsFromConfig(log *logger.Logger, cfg *config.Config) (map[string][]program.Program, error) {
	monitor := noop.NewMonitor()
	router := &inmemRouter{}
	emit := emitter(
		log,
		router,
		&configModifiers{
			Decorators: []decoratorFunc{injectMonitoring},
			Filters:    []filterFunc{filters.ConstraintFilter},
		},
		monitor,
	)

	if err := emit(cfg); err != nil {
		return nil, err
	}
	return router.programs, nil
}

type inmemRouter struct {
	programs map[string][]program.Program
}

func (r *inmemRouter) Dispatch(id string, grpProg map[routingKey][]program.Program) error {
	r.programs = grpProg
	return nil
}

func newErrorLogger() (*logger.Logger, error) {
	return logger.NewWithLogpLevel("", logp.ErrorLevel)
}
