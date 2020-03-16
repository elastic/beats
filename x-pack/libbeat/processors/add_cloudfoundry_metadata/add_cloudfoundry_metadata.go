// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin windows

package add_cloudfoundry_metadata

import (
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
)

const (
	processorName = "add_cloudfoundry_metadata"
)

func init() {
	processors.RegisterPlugin(processorName, New)
}

type addCloudFoundryMetadata struct {
	log    *logp.Logger
	client cloudfoundry.Client
}

const selector = "add_cloudfoundry_metadata"

// New constructs a new add_cloudfoundry_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	var config cloudfoundry.Config
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	log := logp.NewLogger(selector)
	hub := cloudfoundry.NewHub(&config, "add_cloudfoundry_metadata", log)
	client, err := hub.Client()
	if err != nil {
		log.Debugf("%s: failed to created cloudfoundry client: %+v", processorName, err)
	}

	// Janitor run every 5 minutes to clean up the client cache.
	client.StartJanitor(5 * time.Minute)

	return &addCloudFoundryMetadata{
		log:    log,
		client: client,
	}, nil
}

func (d *addCloudFoundryMetadata) Run(event *beat.Event) (*beat.Event, error) {
	if d.client == nil {
		return event, nil
	}
	valI, err := event.GetValue("cloudfoundry.app.id")
	if err != nil {
		// doesn't have the required cf.app.id value to add more information
		return event, nil
	}
	val, _ := valI.(string)
	if val == "" {
		// wrong type or not set
		return event, nil
	}
	app, err := d.client.GetAppByGuid(val)
	if err != nil {
		d.log.Warnf("failed to get application info for GUID(%s): %v", val, err)
		return event, nil
	}
	event.Fields.DeepUpdate(common.MapStr{
		"cloudfoundry": common.MapStr{
			"app": common.MapStr{
				"name": app.Name,
			},
			"space": common.MapStr{
				"id":   app.SpaceData.Meta.Guid,
				"name": app.SpaceData.Entity.Name,
			},
			"org": common.MapStr{
				"id":   app.SpaceData.Entity.OrgData.Meta.Guid,
				"name": app.SpaceData.Entity.OrgData.Entity.Name,
			},
		},
	})
	return event, nil
}

func (d *addCloudFoundryMetadata) String() string {
	return processorName
}
