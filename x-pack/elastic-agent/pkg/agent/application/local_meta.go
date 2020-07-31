// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

func metadata() (*info.ECSMeta, error) {
	agentInfo, err := info.NewAgentInfo()
	if err != nil {
		return nil, err
	}

	meta, err := agentInfo.ECSMetadata()
	if err != nil {
		return nil, errors.New(err, "failed to gather host metadata")
	}

	return meta, nil
}
