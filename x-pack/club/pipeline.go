package main

import (
	"context"
	"errors"
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

type pipelineManager struct {
	log  *logp.Logger
	info beat.Info

	inputLoader *inputLoader
	inputs      []*input

	// XXX: output settings copied from `app`, required for the libbeat/publisher/pipeline
	// setup. These settings will be replaced for dynamic output setup and reloading.
	outputSettings map[string]*common.Config
	rawConfig      *common.Config
}

func newPipelineManager(
	log *logp.Logger,
	info beat.Info,
	inputLoader *inputLoader,
	rawConfig *common.Config,
	outputSettings map[string]*common.Config,
	inputSettings []inputSettings,
) (*pipelineManager, error) {

	// Let's configure inputs. Inputs won't do any processing, yet.
	var inputs []*input
	for _, config := range inputSettings {
		input, err := inputLoader.Configure(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure inputs: %w", err)
		}
		inputs = append(inputs, input)
	}

	return &pipelineManager{
		log:            log,
		info:           info,
		inputLoader:    inputLoader,
		outputSettings: outputSettings,
		rawConfig:      rawConfig,
		inputs:         inputs,
	}, nil
}

func (pm *pipelineManager) Run(ctx context.Context) error {
	var autoCancel ctxtool.AutoCancel
	defer autoCancel.Cancel()

	outputManager, err := configureOutputs(pm.log, pm.info, pm.outputSettings, pm.rawConfig)
	if err != nil {
		return err
	}
	defer outputManager.Close()

	var inputGroup unison.TaskGroup
	ctx = autoCancel.With(ctxtool.WithFunc(ctx, func() {
		pm.log.Info("Stopping inputs...")
		if err := inputGroup.Stop(); err != nil {
			pm.log.Errorf("input failures detected: %v", err)
		}
		pm.log.Info("Inputs stopped.")
	}))

	pm.log.Info("Starting inputs...")
	inputLogger := pm.log.Named("input")

	for _, input := range pm.inputs {
		input := input
		inputGroup.Go(func(cancel unison.Canceler) error {
			inputLogger.Info("start input")
			defer inputLogger.Info("stop input")

			inputContext := v2.Context{
				Logger:      inputLogger,
				ID:          "to-be-set-by-agent",
				Agent:       pm.info,
				Cancelation: cancel,
			}
			return input.Run(inputContext, outputManager.GetPipeline(input.useOutput))
		})
	}
	pm.log.Info("Inputs active...")

	<-ctx.Done()
	return inputGroup.Stop()
}

func (pm *pipelineManager) OnConfig(settings dynamicSettings) error {
	// 1. check if input and output assignments -> ignore outputs without inputs

	// 2. create configured output and input instances (configuration AST)

	// 3. forward setup to run loop, updating active pipelines

	return errors.New("reloading is not yet supported")
}
