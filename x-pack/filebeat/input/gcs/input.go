// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"context"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googleapis/gax-go/v2"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/state"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/types"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type gcsInput struct {
	config config
}

const (
	inputName = "gcs"
)

func Plugin(log *logp.Logger, store cursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "Google Cloud Storage",
		Doc:        "Collect logs from Google Cloud Storage Service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       inputName,
			Configure:  configure,
		},
	}
}

func configure(cfg *conf.C) ([]cursor.Source, cursor.Input, error) {
	config := config{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	var sources []cursor.Source
	for _, b := range config.Buckets {
		bucket := tryOverrideOrDefault(config, b)
		sources = append(sources, &types.Source{
			ProjectId:     config.ProjectId,
			BucketName:    bucket.Name,
			BucketTimeOut: *bucket.BucketTimeOut,
			MaxWorkers:    *bucket.MaxWorkers,
			Poll:          *bucket.Poll,
			PollInterval:  *bucket.PollInterval,
			ParseJSON:     *bucket.ParseJSON,
		})
	}

	return sources, &gcsInput{config: config}, nil
}

// tryOverrideOrDefault, overrides global values with local
// bucket level values if present. If both global & local values
// are absent, assigns default values
func tryOverrideOrDefault(cfg config, b bucket) bucket {
	if b.MaxWorkers == nil {
		maxWorkers := 1
		if cfg.MaxWorkers != nil {
			maxWorkers = *cfg.MaxWorkers
		}
		b.MaxWorkers = &maxWorkers
	}

	if b.Poll == nil {
		var poll bool
		if cfg.Poll != nil {
			poll = *cfg.Poll
		}
		b.Poll = &poll
	}

	if b.PollInterval == nil {
		interval := time.Second * 300
		if cfg.PollInterval != nil {
			interval = *cfg.PollInterval
		}
		b.PollInterval = &interval
	}

	if b.ParseJSON == nil {
		parse := false
		if cfg.ParseJSON != nil {
			parse = *cfg.ParseJSON
		}
		b.ParseJSON = &parse
	}

	if b.BucketTimeOut == nil {
		timeOut := time.Second * 50
		if cfg.BucketTimeOut != nil {
			timeOut = *cfg.BucketTimeOut
		}
		b.BucketTimeOut = &timeOut
	}

	return b
}

func (input *gcsInput) Name() string {
	return inputName
}

func (input *gcsInput) Test(src cursor.Source, ctx v2.TestContext) error {
	return nil
}

func (input *gcsInput) Run(inputCtx v2.Context, src cursor.Source,
	cursor cursor.Cursor, publisher cursor.Publisher) error {
	var cp *state.Checkpoint
	st := state.NewState()
	currentSource := src.(*types.Source)

	log := inputCtx.Logger.With("project_id", currentSource.ProjectId).With("bucket", currentSource.BucketName)
	log.Infof("Running google cloud storage for project: %s", input.config.ProjectId)

	if !cursor.IsNew() {
		if err := cursor.Unpack(&cp); err != nil {
			return err
		}

		st.SetCheckpoint(cp)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-inputCtx.Cancelation.Done()
		cancel()
	}()

	client, err := fetchStorageClient(ctx, input.config, log)
	if err != nil {
		return err
	}
	bucket := client.Bucket(currentSource.BucketName).Retryer(
		// Use WithBackoff to change the timing of the exponential backoff.
		storage.WithBackoff(gax.Backoff{
			Initial: 2 * time.Second,
		}),
		// RetryAlways will retry the operation even if it is non-idempotent.
		// Since we are only reading, the operation is always idempotent
		storage.WithPolicy(storage.RetryAlways),
	)
	scheduler := NewGcsInputScheduler(publisher, bucket, currentSource, &input.config, st, log)

	return scheduler.Schedule(ctx)
}
