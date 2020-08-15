package pipeline

import (
	"context"
	"fmt"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/club/internal/cell"
)

type Controller struct {
	log  *logp.Logger
	info beat.Info

	inputLoader *inputLoader

	// processing config updates Cell[pipeBundleState]
	shouldState *cell.Cell
}

func NewController(
	log *logp.Logger,
	info beat.Info,
	inputsRegistry v2.Registry,
	settings Settings,
) (*Controller, error) {
	inputLoader := newInputLoader(log, inputsRegistry)

	state, err := makePipelineStates(inputLoader, settings)
	if err != nil {
		return nil, err
	}

	return &Controller{
		log:         log,
		info:        info,
		inputLoader: inputLoader,
		shouldState: cell.NewCell(state),
	}, nil
}

func (pm *Controller) Run(ctx context.Context) error {
	type pipelineHandle struct {
		ctx      context.Context
		cancel   func()
		pipeline *pipeline
	}

	var wgActive sync.WaitGroup
	defer wgActive.Wait()

	var muPipelines sync.Mutex
	pipelines := map[string]pipelineHandle{}
	defer func() {
		muPipelines.Lock()
		defer muPipelines.Unlock()
		for _, hdl := range pipelines {
			hdl.cancel()
		}
	}()

	pipelineState := pm.shouldState.Get().(pipeBundleState)
	for {
		muPipelines.Lock()
		// 1. stop pipelines
		for name, hdl := range pipelines {
			if _, exists := pipelineState.pipelines[name]; !exists {
				hdl.cancel()
				delete(pipelines, name)
			}
		}

		// 2. reconfigure existing pipelines
		for name, hdl := range pipelines {
			hdl.pipeline.OnReconfigure(*pipelineState.pipelines[name])
		}

		// 3. start new pipelines
		for name, st := range pipelineState.pipelines {
			if _, exists := pipelines[name]; exists {
				continue
			}

			pipeline, err := newPipeline(pm.log, pm.info, name, *st)
			if err != nil {
				pm.log.Error("Failed to create pipeline: %v", name)
				continue
			}

			pipeCtx, pipeCancel := context.WithCancel(context.Background())
			hdl := pipelineHandle{ctx: pipeCtx, cancel: pipeCancel, pipeline: *&pipeline}
			pipelines[name] = hdl

			wgActive.Add(1)
			go func() {
				defer func() {
					defer wgActive.Done()
					defer pipeCancel()

					// XXX: We always unregister a pipeline on error. On an reconfiguration event
					//      the stopped pipeline will be started again.
					//      Better consider to either:
					//      a) do not remove the handle, but keep the state as failed (more complicate reloading?)
					//      b) pipelines should not fail unless something went really
					//         really wrong. Try to backoff and retry to run the pipeline (makes reloading more consistent).
					muPipelines.Lock()
					defer muPipelines.Unlock()
					delete(pipelines, name)
				}()

				pm.log.Infof("Starting pipeline %v", name)
				defer pm.log.Infof("Pipeline %v stopped", name)

				err := pipeline.Run(hdl.ctx)
				if err != nil && err != context.Canceled {
					pm.log.Errorf("Pipeline %v failed: %v", name, err)
				}
			}()
		}
		muPipelines.Unlock()

		// wait for an update to trigger a reconfiguration
		ifcStateUpdate, err := pm.shouldState.Wait(ctx)
		if err != nil {
			break
		}
		pipelineState = ifcStateUpdate.(pipeBundleState)
	}

	return ctx.Err()
}

func (pm *Controller) OnConfig(settings Settings) error {
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

func makePipelineStates(loader *inputLoader, settings Settings) (pipeBundleState, error) {
	var inputs []*input
	for _, config := range settings.Inputs {
		tmp, err := loader.Configure(config)
		if err != nil {
			return pipeBundleState{}, fmt.Errorf("Failed to configure inputs: %w", err)
		}
		inputs = append(inputs, tmp...)
	}

	pipelineStates := make(map[string]*pipelineState, len(settings.Outputs))
	for name, outConfig := range settings.Outputs {
		pipelineStates[name] = &pipelineState{output: outConfig}
	}

	for _, input := range inputs {
		st := pipelineStates[input.useOutput]
		st.inputs = append(st.inputs, input)
	}

	return pipeBundleState{pipelineStates}, nil
}
