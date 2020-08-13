package main

import (
	"context"
	"errors"
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
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

type pipelineSettings struct {
	Inputs  []inputSettings
	Outputs map[string]*common.Config
}

func newPipelineManager(
	log *logp.Logger,
	info beat.Info,
	inputLoader *inputLoader,
	rawConfig *common.Config,
	settings pipelineSettings,
) (*pipelineManager, error) {

	// Let's configure inputs. Inputs won't do any processing, yet.
	var inputs []*input
	for _, config := range settings.Inputs {
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
		outputSettings: settings.Outputs,
		rawConfig:      rawConfig,
		inputs:         inputs,
	}, nil
}

func (pm *pipelineManager) Run(ctx context.Context) error {
	var autoCancel ctxtool.AutoCancel
	defer autoCancel.Cancel()

	pipelines := map[string]*pipeline.Pipeline{}
	defer func() {
		for _, pipeline := range pipelines {
			pipeline.Close()
		}
	}()

	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, pm.info, pm.rawConfig)
	if err != nil {
		return err
	}

	for name, cfg := range pm.outputSettings {
		typeInfo := struct{ Type string }{}
		if err := cfg.Unpack(&typeInfo); err != nil {
			return err
		}

		var pipeConfig pipeline.Config
		if err := cfg.Unpack(&pipeConfig); err != nil {
			return err
		}

		pipeline, err := pipeline.Load(pm.info,
			pipeline.Monitors{
				Metrics:   nil,
				Telemetry: nil,
				Logger:    pm.log.Named("publish"),
				Tracer:    nil,
			},
			pipeConfig,
			nil,
			makeOutputFactory(pm.info, indexManagement, typeInfo.Type, cfg),
		)
		if err != nil {
			return err
		}
		pipelines[name] = pipeline
	}

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
			return input.Run(inputContext, pipelines[input.useOutput])
		})
	}
	pm.log.Info("Inputs active...")

	<-ctx.Done()
	return inputGroup.Stop()
}

func (pm *pipelineManager) OnConfig(settings pipelineSettings) error {
	// 1. check if input and output assignments -> ignore outputs without inputs

	// 2. create configured output and input instances (configuration AST)

	// 3. forward setup to run loop, updating active pipelines

	return errors.New("reloading is not yet supported")
}

func makeOutputFactory(
	info beat.Info,
	indexManagement idxmgmt.Supporter,
	outputType string,
	cfg *common.Config,
) func(outputs.Observer) (string, outputs.Group, error) {
	return func(outStats outputs.Observer) (string, outputs.Group, error) {
		out, err := outputs.Load(indexManagement, info, outStats, outputType, cfg)
		return outputType, out, err
	}
}
