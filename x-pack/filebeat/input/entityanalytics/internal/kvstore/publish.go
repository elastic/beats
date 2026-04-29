// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/entcollect"
)

// NewPublisher returns an entcollect.Publisher that converts each
// Document to a beat.Event and publishes it via client. The tracker
// is incremented before each Publish so that TxTracker.Wait blocks
// until the pipeline ACKs every event.
func NewPublisher(client beat.Client, inputID string, tracker *TxTracker) entcollect.Publisher {
	return func(ctx context.Context, doc entcollect.Document) error {
		fields := mapstr.M{}
		_, _ = fields.Put("labels.identity_source", inputID)
		_, _ = fields.Put("event.action", doc.Kind.String()+"-"+doc.Action.String())
		_, _ = fields.Put("event.kind", "asset")
		for k, v := range doc.Fields {
			_, _ = fields.Put(k, v)
		}

		event := beat.Event{
			Timestamp: doc.Timestamp,
			Fields:    fields,
			Private:   tracker,
		}
		tracker.Add()
		client.Publish(event)
		return ctx.Err()
	}
}
