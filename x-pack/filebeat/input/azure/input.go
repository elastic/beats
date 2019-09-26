// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/elastic/beats/filebeat/input/kafka"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
)

func init() {
	err := input.Register("azure", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	context input.Context,
) (input.Input, error) {
	// Wrap log input with custom docker settings

	return kafka.NewInput(cfg, connector, context)
}
