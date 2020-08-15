package pipeline

import (
	"context"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	beatpipe "github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/club/internal/cell"
	"github.com/elastic/go-concert/unison"
)

type pipeline struct {
	log  *logp.Logger
	info beat.Info

	outputPipeline *beatpipe.Pipeline

	// pipeline config updates Cell[pipelineState]
	shouldState *cell.Cell
}

type managedInput struct {
	canceler  unison.Canceler
	runCancel context.CancelFunc
	input     *input
}

type pipeBundleState struct {
	pipelines map[string]*pipelineState
}

type pipelineState struct {
	inputs []*input
	output *common.Config
}

func newPipeline(
	log *logp.Logger,
	info beat.Info,
	name string,
	st pipelineState,
) (*pipeline, error) {
	typeInfo := struct{ Type string }{}
	if err := st.output.Unpack(&typeInfo); err != nil {
		return nil, err
	}

	var pipeConfig beatpipe.Config
	if err := st.output.Unpack(&pipeConfig); err != nil {
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
		makeOutputFactory(info, indexManagement, typeInfo.Type, st.output),
	)
	if err != nil {
		return nil, err
	}

	return &pipeline{
		log:            log.With("output", name),
		info:           info,
		outputPipeline: outputPipeline,
		shouldState:    cell.NewCell(st),
	}, nil
}

func (p *pipeline) OnReconfigure(st pipelineState) {
	p.shouldState.Set(st)
}

func (p *pipeline) Run(cancel unison.Canceler) error {
	type inputHandle struct {
		ctx    context.Context
		cancel func()
		input  *input
	}

	// XXX: add input metadata
	log := p.log.Named("input")

	var wgActive sync.WaitGroup
	defer wgActive.Wait()

	// TODO: move output init here, so we can safely shut down and call 'Run'
	// again if the manager want to retry to resurrect the pipeline
	defer p.outputPipeline.Close()

	var muInputs sync.Mutex
	inputs := map[string]inputHandle{}
	defer func() {
		muInputs.Lock()
		defer muInputs.Unlock()
		for _, hdl := range inputs {
			hdl.cancel()
		}
	}()

	state := p.shouldState.Get().(pipelineState)
	for {
		inputHashes := common.StringSet{}
		for _, input := range state.inputs {
			inputHashes.Add(input.configHash)
		}

		muInputs.Lock()
		// 1. stop unknown inputs
		for hash, hdl := range inputs {
			if !inputHashes.Has(hash) {
				hdl.cancel()
				delete(inputs, hash)
			}
		}

		// 2. start new inputs
		for _, inp := range state.inputs {
			if _, exists := inputs[inp.configHash]; exists {
				continue
			}

			inpCtx, inpCancel := context.WithCancel(context.Background())
			hdl := inputHandle{ctx: inpCtx, cancel: inpCancel, input: inp}
			inputs[inp.configHash] = hdl

			wgActive.Add(1)
			go func(inp *input) {
				defer func() {
					defer wgActive.Done()
					defer inpCancel()

					// XXX: We always unregister the input on error. On an reconfiguration event
					//      the stopped input will be started again.
					//      Better consider to either:
					//      a) do not remove the handle, but keep the state as failed (more complicate reloading?)
					//      b) input should not fail unless something went really
					//         really wrong. Try to backoff and retry to run the input again (makes reloading more consistent).
					muInputs.Lock()
					defer muInputs.Unlock()
					delete(inputs, inp.configHash)
				}()

				log.Info("start input")
				defer log.Info("stop input")

				inputContext := v2.Context{
					Logger:      log,
					ID:          "to-be-set-by-agent",
					Agent:       p.info,
					Cancelation: inpCtx,
				}

				err := inp.Run(inputContext, p.outputPipeline)
				if err != nil && err != context.Canceled {
					log.Errorf("Input failed with: %v", err)
				}
			}(inp)
		}
		muInputs.Unlock()

		// wait for an update to trigger a reconfiguration
		ifcStateUpdate, err := p.shouldState.Wait(cancel)
		if err != nil {
			break
		}
		state = ifcStateUpdate.(pipelineState)
	}
	return cancel.Err()
}
