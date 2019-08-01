// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"

	"github.com/elastic/fleet/x-pack/pkg/bus/events"
	"github.com/pkg/errors"
)

type handleFunc func(step events.Step) error

func (o *Operator) initHandlerMap() {
	hm := make(map[string]handleFunc)

	hm[events.StateChangeRun] = o.handleRun
	hm[events.StateChangeRemove] = o.handleRemove
	hm[events.StateChangeStartSidecar] = o.handleStartSidecar
	hm[events.StateChangeStopSidecar] = o.handleStopSidecar

	o.handlers = hm
}

func (o *Operator) handleRun(step events.Step) error {
	p, err := getProgramFromStep(step)
	if err != nil {
		return errors.Wrap(err, "operator.handleStart failed to create program")
	}

	return o.Start(p)
}

func (o *Operator) handleRemove(step events.Step) error {
	p, err := getProgramFromStep(step)
	if err != nil {
		return errors.Wrap(err, "operator.handleStart failed to create program")
	}

	return o.Stop(p)
}

func (o *Operator) handleStartSidecar(step events.Step) error {
	// TODO: add support for monitoring
	return nil
}

func (o *Operator) handleStopSidecar(step events.Step) error {
	// TODO: add support for monitoring
	return nil
}

func getProgramFromStep(step events.Step) (Program, error) {
	metConfig, ok := step.Meta[events.MetaConfigKey]
	if !ok {
		return nil, fmt.Errorf("step: %s, no config in metadata", step.ID)
	}

	config, ok := metConfig.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("step: %s, program config is in invalid format", step.ID)
	}

	p := NewProgram(step.Process, step.Version, config, nil)
	return p, nil
}
