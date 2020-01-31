// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"time"

	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// StorageEvent is the event from Google Cloud Storage
type StorageEvent struct {
	Bucket         string    `json:"bucket"`
	Name           string    `json:"name"`
	Metageneration string    `json:"metageneration"`
	ResourceState  string    `json:"resourceState"`
	Created        time.Time `json:"timeCreated"`
	Updated        time.Time `json:"updated"`
}

// transformPubSub takes a Pub/Sub message and context and transforms it into an event.
func transformPubSub(mData *metadata.Metadata, msg pubsub.Message) (beat.Event, error) {
	return beat.Event{
		Timestamp: mData.Timestamp,
		Fields: common.MapStr{
			"read_timestamp": time.Now(),
			"message":        string(msg.Data),
			"attributes":     msg.Attributes,
			"id":             mData.EventID,
			"pubsub_resource": common.MapStr{
				"service":    mData.Resource.Service,
				"name":       mData.Resource.Name,
				"event_type": mData.Resource.Type,
			},
		},
	}, nil
}

// transformStorage takes a Cloud Storage object and transforms it into an event.
func transformStorage(mData *metadata.Metadata, evt StorageEvent) (beat.Event, error) {

	return beat.Event{
		Timestamp: mData.Timestamp,
		Fields: common.MapStr{
			"read_timestamp": time.Now(),
			"id":             mData.EventID,
			"storage_resource": common.MapStr{
				"service":    mData.Resource.Service,
				"name":       mData.Resource.Name,
				"event_type": mData.Resource.Type,
				"state":      evt.ResourceState,
			},
			"storage_bucket": evt.Bucket,
			"file_name":      evt.Name,
			"metageneration": evt.Metageneration,
			"update_time":    evt.Updated,
			"create_time":    evt.Created,
		},
	}, nil
}
