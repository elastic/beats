package pipeline

import (
	"context"
	"fmt"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/club/internal/cell"
	"github.com/elastic/go-concert/unison"
)

type Controller struct {
	log  *logp.Logger
	info beat.Info

	inputLoader *inputLoader

	// processing config updates Cell[map[string]*pipelineState]
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
	var pipelineGroup managedGroup
	defer pipelineGroup.Stop()

	// keep track of active pipeline in order to send configuration updates
	var muPipelines sync.Mutex
	pipelines := map[string]*pipeline{}

	pipelineState := pm.getState()
	for {
		muPipelines.Lock()

		stopped := pipelineGroup.FindAll(func(name string) bool {
			_, exists := pipelineState[name]
			return !exists
		})
		for _, hdl := range stopped {
			hdl.cancel()
		}

		// 2. reconfigure existing pipelines
		for name, pipeline := range pipelines {
			if st := pipelineState[name]; st != nil {
				pipeline.OnReconfigure(*st)
			}
		}

		// 3. start new pipelines
		for name, st := range pipelineState {
			if _, exists := pipelines[name]; exists {
				continue
			}

			pipeline := newPipeline(pm.log.With("output", name), pm.info, name, *st)
			pipelines[name] = pipeline

			pipelineGroup.Go(name, func(cancel unison.Canceler) {
				// XXX: We always unregister a pipeline on error. On an reconfiguration event
				//      the stopped pipeline will be started again.
				//      Better consider to either:
				//      a) do not remove the handle, but keep the state as failed (more complicate reloading?)
				//      b) pipelines should not fail unless something went really
				//         really wrong. Try to backoff and retry to run the pipeline (makes reloading more consistent).
				defer func() {
					muPipelines.Lock()
					defer muPipelines.Unlock()
					delete(pipelines, pipeline.name)
				}()

				pipeline.log.Infof("Starting pipeline %v", pipeline.name)
				defer pipeline.log.Infof("Pipeline %v stopped", pipeline.name)

				err := pipeline.Run(cancel)
				if err != nil && err != context.Canceled {
					pipeline.log.Errorf("Pipeline %v failed: %v", pipeline.name, err)
				}
			})
		}

		muPipelines.Unlock()

		var err error
		waitAll(stopped)
		pipelineState, err = pm.waitStateUpdate(ctx)
		if err != nil {
			break
		}
	}

	return ctx.Err()
}

func (pm *Controller) getState() map[string]*pipelineState {
	return pm.shouldState.Get().(map[string]*pipelineState)
}

func (pm *Controller) waitStateUpdate(ctx unison.Canceler) (map[string]*pipelineState, error) {
	// wait for an update to trigger a reconfiguration
	ifcStateUpdate, err := pm.shouldState.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return ifcStateUpdate.(map[string]*pipelineState), nil
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

func makePipelineStates(loader *inputLoader, settings Settings) (map[string]*pipelineState, error) {
	var inputs []*input
	for _, config := range settings.Inputs {
		tmp, err := loader.Configure(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure inputs: %w", err)
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

	return pipelineStates, nil
}
