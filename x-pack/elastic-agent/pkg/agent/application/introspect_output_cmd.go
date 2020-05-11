// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/urso/ecslog"
	"github.com/urso/ecslog/backend"
	"github.com/urso/ecslog/backend/appender"
	"github.com/urso/ecslog/backend/layout"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filters"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/app/monitoring/noop"
)

// IntrospectOutputCmd is an introspect subcommand that shows configurations of the agent.
type IntrospectOutputCmd struct {
	cfgPath string
	output  string
	program string
}

// NewIntrospectOutputCmd creates a new introspect command.
func NewIntrospectOutputCmd(configPath, output, program string) (*IntrospectOutputCmd, error) {
	return &IntrospectOutputCmd{
		cfgPath: configPath,
		output:  output,
		program: program,
	}, nil
}

// Execute tries to enroll the agent into Fleet.
func (c *IntrospectOutputCmd) Execute() error {
	if c.output == "" {
		return c.introspectOutputs()
	}

	return c.introspectOutput()
}

func (c *IntrospectOutputCmd) introspectOutputs() error {
	cfg, err := loadConfig(c.cfgPath)
	if err != nil {
		return err
	}

	isLocal, err := isLocalMode(cfg)
	if err != nil {
		return err
	}

	l, err := newErrorLogger()
	if err != nil {
		return err
	}

	if isLocal {
		return listOutputsFromConfig(l, cfg)
	}

	fleetConfig, err := loadFleetConfig(cfg)
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

func (c *IntrospectOutputCmd) introspectOutput() error {
	cfg, err := loadConfig(c.cfgPath)
	if err != nil {
		return err
	}

	l, err := newErrorLogger()
	if err != nil {
		return err
	}

	isLocal, err := isLocalMode(cfg)
	if err != nil {
		return err
	}

	if isLocal {
		return printOutputFromConfig(l, c.output, c.program, cfg)
	}

	fleetConfig, err := loadFleetConfig(cfg)
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
			return fmt.Errorf("program '%s' is not recognized within output '%s', try running `elastic-agent introspect output` to find available outputs",
				programName,
				output)
		}

		return nil
	}

	return fmt.Errorf("output '%s' is not recognized, try running `elastic-agent introspect output` to find available outputs", output)

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
	backend, err := appender.Console(backend.Error, layout.Text(true))
	if err != nil {
		return nil, err
	}
	return ecslog.New(backend), nil
}
