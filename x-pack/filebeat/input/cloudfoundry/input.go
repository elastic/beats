// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package cloudfoundry

import (
	"fmt"
	"time"

	v2 "github.com/elastic/beats/v8/filebeat/input/v2"
	stateless "github.com/elastic/beats/v8/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/feature"

	"github.com/elastic/beats/v8/x-pack/libbeat/common/cloudfoundry"
)

type cloudfoundryEvent interface {
	Timestamp() time.Time
	ToFields() common.MapStr
}

func Plugin() v2.Plugin {
	return v2.Plugin{
		Name:       "cloudfoundry",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "collect logs from cloudfoundry loggregator",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *common.Config) (stateless.Input, error) {
	config := cloudfoundry.Config{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	switch config.Version {
	case cloudfoundry.ConsumerVersionV1:
		return configureV1(config)
	case cloudfoundry.ConsumerVersionV2:
		return configureV2(config)
	default:
		return nil, fmt.Errorf("not supported consumer version: %s", config.Version)
	}
}

func createEvent(evt cloudfoundryEvent) beat.Event {
	return beat.Event{
		Timestamp: evt.Timestamp(),
		Fields:    evt.ToFields(),
	}
}
