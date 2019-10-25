// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/configrequest"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/app"
)

type handleFunc func(step configrequest.Step) error

func (o *Operator) initHandlerMap() {
	hm := make(map[string]handleFunc)

	hm[configrequest.StepRun] = o.handleRun
	hm[configrequest.StepRemove] = o.handleRemove
	hm[configrequest.StepStartSidecar] = o.handleStartSidecar
	hm[configrequest.StepStopSidecar] = o.handleStopSidecar

	o.handlers = hm
}

func (o *Operator) handleRun(step configrequest.Step) error {
	p, cfg, err := getProgramFromStep(step)
	if err != nil {
		return errors.Wrap(err, "operator.handleStart failed to create program")
	}

	return o.start(p, cfg)
}

func (o *Operator) handleRemove(step configrequest.Step) error {
	p, _, err := getProgramFromStep(step)
	if err != nil {
		return errors.Wrap(err, "operator.handleStart failed to create program")
	}

	return o.stop(p)
}

func (o *Operator) handleStartSidecar(step configrequest.Step) error {
	// TODO: add support for monitoring
	return nil
}

func (o *Operator) handleStopSidecar(step configrequest.Step) error {
	// TODO: add support for monitoring
	return nil
}

func getProgramFromStep(step configrequest.Step) (Descriptor, map[string]interface{}, error) {
	metConfig, ok := step.Meta[configrequest.MetaConfigKey]
	if !ok {
		return nil, nil, fmt.Errorf("step: %s, no config in metadata", step.ID)
	}

	config, ok := metConfig.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("step: %s, program config is in invalid format", step.ID)
	}

	p := app.NewDescriptor(step.Process, step.Version, nil)
	return p, config, nil
}
