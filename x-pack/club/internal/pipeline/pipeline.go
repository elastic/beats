package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	beatpipe "github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/club/internal/cell"
	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"
)

type pipeline struct {
	log  *logp.Logger
	info beat.Info

	// pipeline config updates Cell[pipelineState]
	shouldState *cell.Cell
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
) *pipeline {
	return &pipeline{
		log:         log.With("output", name),
		info:        info,
		shouldState: cell.NewCell(st),
	}
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

	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagementConfig := common.MustNewConfigFrom(map[string]interface{}{
		"setup.ilm.enabled":      false,
		"setup.template.enabled": false,
		"output.something":       map[string]interface{}{},
	})

	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, p.info, indexManagementConfig)
	if err != nil {
		// the config is hard coded, if we panic here, we've messed up
		panic(err)
	}

	state := p.readState()

	var pipeConfig beatpipe.Config
	if err := state.output.Unpack(&pipeConfig); err != nil {
		return err
	}

	typeInfo := struct{ Type string }{}
	if err := state.output.Unpack(&typeInfo); err != nil {
		return err
	}

	outputPipeline, err := beatpipe.Load(p.info,
		beatpipe.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log.Named("publish"),
			Tracer:    nil,
		},
		pipeConfig,
		nil,
		makeOutputFactory(p.info, indexManagement, typeInfo.Type, state.output),
	)
	if err != nil {
		return fmt.Errorf("invalid output configuration: %w", err)
	}
	defer outputPipeline.Close()

	var muInputs sync.Mutex
	inputs := map[string]inputHandle{}
	defer func() {
		muInputs.Lock()
		defer muInputs.Unlock()
		for _, hdl := range inputs {
			hdl.cancel()
		}
	}()

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

				// Input main loop. We restart the input (after a short wait delay), in
				// case an error or panic was encountered
				for {
					err := inp.Run(inputContext, outputPipeline)
					if err != nil && err != context.Canceled {
						log.Errorf("Input failed with: %v", err)
					}

					// TODO: exponential backoff? Shall we kill the input at some point?
					if err := timed.Wait(inpCtx, 5*time.Second); err != nil {
						break
					}

					log.Info("Restarting failed input")
				}
			}(inp)
		}
		muInputs.Unlock()

		// wait for an update to trigger a reconfiguration
		var err error
		state, err = p.waitStateUpdate(cancel)
		if err != nil {
			break
		}
	}
	return cancel.Err()
}

func (p *pipeline) readState() pipelineState {
	return p.shouldState.Get().(pipelineState)
}

func (p *pipeline) waitStateUpdate(cancel unison.Canceler) (pipelineState, error) {
	ifcStateUpdate, err := p.shouldState.Wait(cancel)
	if err != nil {
		return pipelineState{}, err
	}
	return ifcStateUpdate.(pipelineState), nil
}
