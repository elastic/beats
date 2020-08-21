package pipeline

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/collector/internal/cell"
	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"
)

type pipeline struct {
	log  *logp.Logger
	info beat.Info
	name string

	// pipeline config updates Cell[pipelineState]
	shouldState *cell.Cell
}

type pipelineState struct {
	inputs []*input
	output output
}

func newPipeline(
	log *logp.Logger,
	info beat.Info,
	name string,
	st pipelineState,
) *pipeline {
	return &pipeline{
		log:         log.With("output", name),
		name:        name,
		info:        info,
		shouldState: cell.NewCell(st),
	}
}

func (p *pipeline) OnReconfigure(st pipelineState) {
	p.shouldState.Set(st)
}

func (p *pipeline) Run(cancel unison.Canceler) error {
	// XXX: add input metadata
	log := p.log.Named("input")

	state := p.readState()

	outputPipeline, err := createPublishPipeline(p.log, p.info, state.output)
	if err != nil {
		return err
	}
	defer outputPipeline.Close()

	var inputGroup managedGroup
	defer inputGroup.Stop()

	for {
		inputHashes := common.StringSet{}
		for _, input := range state.inputs {
			inputHashes.Add(input.configHash)
		}

		// 1. stop unknown inputs
		stopped := inputGroup.FindAll(func(hash string) bool {
			return !inputHashes.Has(hash)
		})
		cancelAll(stopped)

		// 2. update output
		if err := outputPipeline.UpdateOutput(state.output); err != nil {
			return fmt.Errorf("pipeline without output after reconfiguration attempt: %w", err)
		}

		// 3. start new inputs
		for _, inp := range state.inputs {
			inp := inp
			if inputGroup.Has(inp.configHash) {
				continue
			}

			inputGroup.Go(inp.configHash, func(cancel unison.Canceler) {
				log.Info("start input")
				defer log.Info("stop input")

				inputContext := v2.Context{
					Logger:      log,
					ID:          "to-be-set-by-agent",
					Agent:       p.info,
					Cancelation: cancel,
				}

				// Input main loop. We restart the input (after a short wait delay), in
				// case an error or panic was encountered
				for {
					err := inp.Run(inputContext, outputPipeline)
					if err != nil && err != context.Canceled {
						log.Errorf("Input failed with: %v", err)
					}

					if cancel.Err() != nil {
						break
					}

					// TODO: exponential backoff? Shall we kill the input at some point?
					if err := timed.Wait(cancel, 5*time.Second); err != nil {
						break
					}

					log.Info("Restarting failed input")
				}
			})
		}

		// Wait for completion before we check for another state update.  Go routines
		// started by the managedGroup unregister themselves from the
		// group after they have returned. We need to wait for the internal state
		// to finally match the shouldState, so we do not try to
		// restart/reconfigure an input that is still shutting down, potentially
		// overwriting state in the managedGroup, of having an unbound number of
		// go-routines still shutting down if updates are too fast.
		waitAll(stopped)

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
