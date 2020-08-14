package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	beatpipe "github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/club/internal/cell"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

type pipelineManager struct {
	log  *logp.Logger
	info beat.Info

	inputLoader *inputLoader

	// pipeline config updates Cell[pipeBundleState]
	shouldState *cell.Cell
}

type pipeline struct {
	log *logp.Logger

	inputGroup unison.TaskGroup
	inputs     map[string]*input

	outputConfig   *common.Config
	outputPipeline *beatpipe.Pipeline
}

type pipelineSettings struct {
	Inputs  []inputSettings           `config:"club.inputs"`
	Outputs map[string]*common.Config `config:"outputs"`
}

type pipeBundleState struct {
	inputs  []*input
	outputs map[string]*common.Config
}

func newPipelineManager(
	log *logp.Logger,
	info beat.Info,
	inputLoader *inputLoader,
	settings pipelineSettings,
) (*pipelineManager, error) {

	state, err := makePipelineStates(inputLoader, settings)
	if err != nil {
		return nil, err
	}

	return &pipelineManager{
		log:         log,
		info:        info,
		inputLoader: inputLoader,
		shouldState: cell.NewCell(state),
	}, nil
}

func (s *pipelineSettings) Validate() error {
	if _, exists := s.Outputs["default"]; !exists {
		return errors.New("no default output configured")
	}

	for _, inp := range s.Inputs {
		if inp.UseOutput == "" {
			continue
		}
		if _, exist := s.Outputs[inp.UseOutput]; !exist {
			return fmt.Errorf("output '%v' not defined", inp.UseOutput)
		}
	}

	return nil
}

func (pm *pipelineManager) Run(ctx context.Context) error {
	var autoCancel ctxtool.AutoCancel
	defer autoCancel.Cancel()

	pipelines := map[string]*pipeline{}
	defer func() {
		for _, pipeline := range pipelines {
			pipeline.Close()
		}
	}()

	pipelineState := pm.shouldState.Get().(pipeBundleState)

	for name, cfg := range pipelineState.outputs {
		pipeline, err := newPipeline(pm.log, pm.info, name, cfg)
		if err != nil {
			return err
		}
		pipelines[name] = pipeline
	}

	pm.log.Info("Starting inputs...")
	for _, input := range pipelineState.inputs {

		// configuration validation did ensure that the output must exist
		pipeline := pipelines[input.useOutput]
		if pipeline == nil {
			panic(fmt.Errorf("unknown pipeline requested: %v", input.useOutput))
		}

		pipeline.startInput(pm.info, input)
	}
	pm.log.Info("Inputs active...")

	for {
		ifcStateUpdate, err := pm.shouldState.Wait(ctx)
		if err != nil {
			break
		}

		shouldState := ifcStateUpdate.(pipeBundleState)

		var pipelineNames common.StringSet
		for name := range pipelines {
			pipelineNames.Add(name)
		}
		for name := range shouldState.outputs {
			pipelineNames.Add(name)
		}

		var removed []*pipeline
		for name := range pipelineNames {
			if _, exists := shouldState.outputs[name]; !exists {
				removed = append(removed, pipelines[name])
				delete(pipelines, name)
			}
			if _, exists := pipelines[name]; !exists {
				pipeline, err := newPipeline(pm.log, pm.info, name, shouldState.outputs[name])
				if err != nil {
					// TODO: pipeline manager is running in degraded state -> report health to status.Reporter
					pm.log.Errorf("Failed to initialize pipeline for %v", name)
					continue
				}
				pipelines[name] = pipeline
			}
		}

		var wg sync.WaitGroup

		wg.Add(len(removed))
		for _, pipeline := range removed {
			pipeline := pipeline
			go func() {
				defer wg.Done()
				pipeline.Close()
			}()
		}
		removed = nil

		/*
			wg.Add(len(pipelines))
			for name, pipeline := range pipelines {
				name, pipeline := name, pipeline
				go func() {
					defer wg.Done()
					for _, input := range shouldState.inputs {
						if input.useOutput != name {
							continue
						}

					}
				}()
			}
		*/

		wg.Wait()
	}

	return ctx.Err()
}

func (pm *pipelineManager) OnConfig(settings pipelineSettings) error {
	if err := settings.Validate(); err != nil {
		return err
	}

	state, err := makePipelineStates(pm.inputLoader, settings)
	if err != nil {
		return err
	}

	pm.shouldState.Set(state)
	return nil
}

func newPipeline(
	log *logp.Logger,
	info beat.Info,
	name string,
	cfg *common.Config,
) (*pipeline, error) {
	typeInfo := struct{ Type string }{}
	if err := cfg.Unpack(&typeInfo); err != nil {
		return nil, err
	}

	var pipeConfig beatpipe.Config
	if err := cfg.Unpack(&pipeConfig); err != nil {
		return nil, err
	}

	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagementConfig := common.MustNewConfigFrom(map[string]interface{}{
		"setup.ilm.enabled":      false,
		"setup.template.enabled": false,
		"output.something":       map[string]interface{}{},
	})

	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, info, indexManagementConfig)
	if err != nil {
		return nil, err
	}

	outputPipeline, err := beatpipe.Load(info,
		beatpipe.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log.Named("publish"),
			Tracer:    nil,
		},
		pipeConfig,
		nil,
		makeOutputFactory(info, indexManagement, typeInfo.Type, cfg),
	)
	if err != nil {
		return nil, err
	}

	return &pipeline{
		log:            log.With("output", name),
		outputConfig:   cfg,
		outputPipeline: outputPipeline,
	}, nil
}

func (p *pipeline) Close() error {
	errInputs := p.inputGroup.Stop()
	if errInputs != nil {
		p.log.Errorf("Pipeline error during shutdown: %v", errInputs)
	}

	p.outputPipeline.Close()
	return errInputs
}

func (p *pipeline) startInput(info beat.Info, inp *input) {
	// XXX: add input metadata
	inputLogger := p.log.Named("input")

	p.inputGroup.Go(func(cancel unison.Canceler) error {
		inputLogger.Info("start input")
		defer inputLogger.Info("stop input")

		inputContext := v2.Context{
			Logger:      inputLogger,
			ID:          "to-be-set-by-agent",
			Agent:       info,
			Cancelation: cancel,
		}
		return inp.Run(inputContext, p.outputPipeline)
	})
}

func makePipelineStates(loader *inputLoader, settings pipelineSettings) (pipeBundleState, error) {
	var inputs []*input
	for _, config := range settings.Inputs {
		tmp, err := loader.Configure(config)
		if err != nil {
			return pipeBundleState{}, fmt.Errorf("Failed to configure inputs: %w", err)
		}
		inputs = append(inputs, tmp...)
	}

	return pipeBundleState{
		inputs:  inputs,
		outputs: settings.Outputs,
	}, nil
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
