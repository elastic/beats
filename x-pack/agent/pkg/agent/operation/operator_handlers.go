// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app"
)

type handleFunc func(step configrequest.Step) error

func (o *Operator) initHandlerMap() {
	hm := make(map[string]handleFunc)

	hm[configrequest.StepRun] = o.handleRun
	hm[configrequest.StepRemove] = o.handleRemove

	o.handlers = hm
}

func (o *Operator) handleRun(step configrequest.Step) error {
	if step.Process == monitoringName {
		return o.handleStartSidecar(step)
	}

	p, cfg, err := getProgramFromStep(step, o.config.DownloadConfig)
	if err != nil {
		return errors.New(err,
			"operator.handleStart failed to create program",
			errors.TypeApplication,
			errors.M(errors.MetaKeyAppName, step.Process))
	}

	return o.start(p, cfg)
}

func (o *Operator) handleRemove(step configrequest.Step) error {
	if step.Process == monitoringName {
		return o.handleStopSidecar(step)
	}

	p, _, err := getProgramFromStep(step, o.config.DownloadConfig)
	if err != nil {
		return errors.New(err,
			"operator.handleRemove failed to stop program",
			errors.TypeApplication,
			errors.M(errors.MetaKeyAppName, step.Process))
	}

	return o.stop(p)
}

func getProgramFromStep(step configrequest.Step, artifactConfig *artifact.Config) (Descriptor, map[string]interface{}, error) {
	return getProgramFromStepWithTags(step, artifactConfig, nil)
}

func getProgramFromStepWithTags(step configrequest.Step, artifactConfig *artifact.Config, tags map[app.Tag]string) (Descriptor, map[string]interface{}, error) {
	config, err := getConfigFromStep(step)
	if err != nil {
		return nil, nil, err
	}

	p := app.NewDescriptor(step.Process, step.Version, artifactConfig, tags)
	return p, config, nil
}

func getConfigFromStep(step configrequest.Step) (map[string]interface{}, error) {
	metConfig, hasConfig := step.Meta[configrequest.MetaConfigKey]

	if !hasConfig && needsMetaConfig(step) {
		return nil, fmt.Errorf("step: %s, no config in metadata", step.ID)
	}

	var config map[string]interface{}
	if hasConfig {
		var ok bool
		config, ok = metConfig.(map[string]interface{})
		if !ok {
			return nil, errors.New(errors.TypeConfig,
				fmt.Sprintf("step: %s, program config is in invalid format", step.ID))
		}
	}

	return config, nil
}

func needsMetaConfig(step configrequest.Step) bool {
	return step.ID == configrequest.StepRun
}
