// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package add_cloudfoundry_metadata

import (
	"github.com/gofrs/uuid"
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

	// ShardID is required in cloudfoundry config to consume from the firehose,
	// but not for metadata requests, randomly generate one and use it.
	config.ShardID = uuid.Must(uuid.NewV4()).String()

	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	log := logp.NewLogger(selector)
	hub := cloudfoundry.NewHub(&config, "add_cloudfoundry_metadata", log)
	client, err := hub.ClientWithCache()
	if err != nil {
		return nil, errors.Wrapf(err, "%s: creating cloudfoundry client", processorName)
	}

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
		// doesn't have the required cloudfoundry.app.id value to add more information
		return event, nil
	}
	val, _ := valI.(string)
	if val == "" {
		// wrong type or not set
		return event, nil
	}
	if hasMetadataFields(event) {
		// nothing to do, fields already present
		return event, nil
	}
	app, err := d.client.GetAppByGuid(val)
	if err != nil {
		d.log.Debugf("failed to get application info for GUID(%s): %v", val, err)
		return event, nil
	}
	event.Fields.DeepUpdate(common.MapStr{
		"cloudfoundry": common.MapStr{
			"app": common.MapStr{
				"name": app.Name,
			},
			"space": common.MapStr{
				"id":   app.SpaceGuid,
				"name": app.SpaceName,
			},
			"org": common.MapStr{
				"id":   app.OrgGuid,
				"name": app.OrgName,
			},
		},
	})
	return event, nil
}

// String returns this processor name.
func (d *addCloudFoundryMetadata) String() string {
	return processorName
}

// Close closes the underlying client and releases its resources.
func (d *addCloudFoundryMetadata) Close() error {
	if d.client == nil {
		return nil
	}
	err := d.client.Close()
	if err != nil {
		return errors.Wrap(err, "closing client")
	}
	return nil
}

var metadataFields = []string{
	"cloudfoundry.app.id",
	"cloudfoundry.app.name",
	"cloudfoundry.space.id",
	"cloudfoundry.space.name",
	"cloudfoundry.org.id",
	"cloudfoundry.org.name",
}

func hasMetadataFields(event *beat.Event) bool {
	for _, name := range metadataFields {
		if value, err := event.GetValue(name); value == "" || err != nil {
			return false
		}
	}
	return true
}
