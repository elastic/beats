// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	composableCtrl, err := composable.New(cfg)
	if err != nil {
		return nil, err
	}
	composableWaiter := newWaitForCompose(composableCtrl)
	emit, err := emitter(
		ctx,
		log,
		composableWaiter,
		router,
		&configModifiers{
			Decorators: []decoratorFunc{injectMonitoring},
		},
		monitor,
	)
	if err != nil {
		return nil, err
	}

	if err := emit(cfg); err != nil {
		return nil, err
	}
	composableWaiter.Wait()
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

type waitForCompose struct {
	controller composable.Controller
	done       chan bool
}

func newWaitForCompose(wrapped composable.Controller) *waitForCompose {
	return &waitForCompose{
		controller: wrapped,
		done:       make(chan bool),
	}
}

func (w *waitForCompose) Run(ctx context.Context, cb composable.VarsCallback) error {
	err := w.controller.Run(ctx, func(vars []*transpiler.Vars) {
		cb(vars)
		w.done <- true
	})
	return err
}

func (w *waitForCompose) Wait() {
	<-w.done
}
