// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
)

func init() {
	err := input.Register("cloudfoundry", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new udp input
func NewInput(
	cfg *common.Config,
	outlet channel.Connector,
	context input.Context,
) (input.Input, error) {
	cfgwarn.Beta("The cloudfoundry input is beta")

	log := logp.NewLogger("cloudfoundry")

	out, err := outlet.Connect(cfg)
	if err != nil {
		return nil, err
	}

	var conf cloudfoundry.Config
	if err = cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	switch conf.Version {
	case cloudfoundry.ConsumerVersionV1:
		return newInputV1(log, conf, out, context)
	case cloudfoundry.ConsumerVersionV2:
		return newInputV2(log, conf, out, context)
	default:
		return nil, fmt.Errorf("not supported consumer version: %s", conf.Version)
	}
}
