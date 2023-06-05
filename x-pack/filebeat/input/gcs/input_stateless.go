// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"context"
	"time"

	"cloud.google.com/go/storage"
	gax "github.com/googleapis/gax-go/v2"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type statelessInput struct {
	config config
}

func (statelessInput) Name() string {
	return "gcs-stateless"
}

func newStatelessInput(config config) *statelessInput {
	return &statelessInput{config: config}
}

func (in *statelessInput) Test(v2.TestContext) error {
	return nil
}

type statelessPublisher struct {
	wrapped stateless.Publisher
}

func (pub statelessPublisher) Publish(event beat.Event, _ interface{}) error {
	pub.wrapped.Publish(event)
	return nil
}

// Run starts the input and blocks until it ends the execution.
// It will return on context cancellation, any other error will be retried.
func (in *statelessInput) Run(inputCtx v2.Context, publisher stateless.Publisher, client *storage.Client) error {
	pub := statelessPublisher{wrapped: publisher}
	var source cursor.Source
	for _, b := range in.config.Buckets {
		bucket := tryOverrideOrDefault(in.config, b)
		source = &Source{
			ProjectId:     in.config.ProjectId,
			BucketName:    bucket.Name,
			BucketTimeOut: *bucket.BucketTimeOut,
			MaxWorkers:    *bucket.MaxWorkers,
			Poll:          *bucket.Poll,
			PollInterval:  *bucket.PollInterval,
			ParseJSON:     *bucket.ParseJSON,
		}

		st := newState()
		currentSource := source.(*Source)
		log := inputCtx.Logger.With("project_id", currentSource.ProjectId).With("bucket", currentSource.BucketName)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-inputCtx.Cancelation.Done()
			cancel()
		}()

		bkt := client.Bucket(currentSource.BucketName).Retryer(
			// Use WithBackoff to change the timing of the exponential backoff.
			storage.WithBackoff(gax.Backoff{
				Initial: 2 * time.Second,
			}),
			// RetryAlways will retry the operation even if it is non-idempotent.
			// Since we are only reading, the operation is always idempotent
			storage.WithPolicy(storage.RetryAlways),
		)

		scheduler := newScheduler(pub, bkt, currentSource, &in.config, st, log)

		return scheduler.schedule(ctx)
	}
	return nil
}
