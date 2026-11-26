// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"
	"maps"
	"slices"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/pkg/mapstr"
	"github.com/elastic/entcollect"
)

// NewPublisher returns an entcollect.Publisher that converts each
// Document to a beat.Event and publishes it via client. The tracker
// is incremented before each Publish so that TxTracker.Wait blocks
// until the pipeline ACKs every event. The returned Publisher must not
// be called once tracker.Wait has been called, for example from a
// goroutine that outlives the caller, because events published after
// Wait is released are not waited for.
func NewPublisher(client beat.Client, inputID string, tracker *TxTracker) entcollect.Publisher {
	return func(ctx context.Context, doc entcollect.Document) error {
		fields := mapstr.M{}
		_, _ = fields.Put("labels.identity_source", inputID)
		_, _ = fields.Put("event.action", doc.Kind.String()+"-"+doc.Action.String())
		_, _ = fields.Put("event.kind", "asset")
		// Put a key before any dotted key under it, so a map value
		// under "a" can't replace an "a.b" value that was put first.
		// A key sorts before every key it is a prefix of.
		for _, k := range slices.Sorted(maps.Keys(doc.Fields)) {
			_, _ = fields.Put(k, doc.Fields[k])
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
