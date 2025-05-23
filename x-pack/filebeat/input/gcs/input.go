// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	gax "github.com/googleapis/gax-go/v2"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type gcsInput struct {
	config config
}

// defines the valid range for Unix timestamps for 64 bit integers
var (
	minTimestamp = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxTimestamp = time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
)

const (
	inputName = "gcs"
)

func Plugin(log *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
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
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}
	sources := make([]cursor.Source, 0, len(config.Buckets))
	// This is to maintain backward compatibility with the old config.
	if config.BatchSize == 0 {
		config.BatchSize = config.MaxWorkers
	}
	for _, b := range config.Buckets {
		bucket := tryOverrideOrDefault(config, b)
		if bucket.TimeStampEpoch != nil && !isValidUnixTimestamp(*bucket.TimeStampEpoch) {
			return nil, nil, fmt.Errorf("invalid timestamp epoch: %d", *bucket.TimeStampEpoch)
		}
		sources = append(sources, &Source{
			ProjectId:                config.ProjectId,
			BucketName:               bucket.Name,
			BatchSize:                *bucket.BatchSize,
			MaxWorkers:               *bucket.MaxWorkers,
			Poll:                     *bucket.Poll,
			PollInterval:             *bucket.PollInterval,
			ParseJSON:                *bucket.ParseJSON,
			TimeStampEpoch:           bucket.TimeStampEpoch,
			ExpandEventListFromField: bucket.ExpandEventListFromField,
			FileSelectors:            bucket.FileSelectors,
			ReaderConfig:             bucket.ReaderConfig,
			Retry:                    config.Retry,
		})
	}

	return sources, &gcsInput{config: config}, nil
}

// tryOverrideOrDefault, overrides the bucket level values with global values if the bucket fields are not set
func tryOverrideOrDefault(cfg config, b bucket) bucket {
	if b.MaxWorkers == nil {
		b.MaxWorkers = &cfg.MaxWorkers
	}
	if b.BatchSize == nil && cfg.BatchSize != 0 {
		// If the global batch size is set, use it
		b.BatchSize = &cfg.BatchSize
	} else {
		// If the global batch size is not set, use the local max_workers as the batch size
		// since at this point we know `b.MaxWorkers` will be set to a non-nil value.
		// This is to maintain backward compatibility with the old config.
		b.BatchSize = b.MaxWorkers
	}
	if b.Poll == nil {
		b.Poll = &cfg.Poll
	}
	if b.PollInterval == nil {
		b.PollInterval = &cfg.PollInterval
	}
	if b.ParseJSON == nil {
		b.ParseJSON = &cfg.ParseJSON
	}
	if b.TimeStampEpoch == nil {
		b.TimeStampEpoch = cfg.TimeStampEpoch
	}
	if b.ExpandEventListFromField == "" {
		b.ExpandEventListFromField = cfg.ExpandEventListFromField
	}
	if len(b.FileSelectors) == 0 && len(cfg.FileSelectors) != 0 {
		b.FileSelectors = cfg.FileSelectors
	}
	b.ReaderConfig = cfg.ReaderConfig

	return b
}

// isValidUnixTimestamp checks if the timestamp is a valid Unix timestamp
func isValidUnixTimestamp(timestamp int64) bool {
	// checks if the timestamp is within the valid range
	return minTimestamp <= timestamp && timestamp <= maxTimestamp
}

func (input *gcsInput) Name() string {
	return inputName
}

func (input *gcsInput) Test(src cursor.Source, ctx v2.TestContext) error {
	return nil
}

func (input *gcsInput) Run(inputCtx v2.Context, src cursor.Source,
	cursor cursor.Cursor, publisher cursor.Publisher,
) error {
	st := newState()
	currentSource := src.(*Source)

	log := inputCtx.Logger.With("project_id", currentSource.ProjectId).With("bucket", currentSource.BucketName)
	log.Infof("Running google cloud storage for project: %s", input.config.ProjectId)
	// create a new inputMetrics instance
	metrics := newInputMetrics(inputCtx.ID+":"+currentSource.BucketName, nil)
	metrics.url.Set("gs://" + currentSource.BucketName)
	defer metrics.Close()

	var cp *Checkpoint
	if !cursor.IsNew() {
		if err := cursor.Unpack(&cp); err != nil {
			metrics.errorsTotal.Inc()
			return err
		}

		st.setCheckpoint(cp)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-inputCtx.Cancelation.Done()
		cancel()
	}()

	client, err := fetchStorageClient(ctx, input.config)
	if err != nil {
		metrics.errorsTotal.Inc()
		return err
	}

	bucket := client.Bucket(currentSource.BucketName).Retryer(
		// Use WithMaxAttempts to change the maximum number of attempts.
		storage.WithMaxAttempts(currentSource.Retry.MaxAttempts),
		// Use WithBackoff to change the timing of the exponential backoff.
		storage.WithBackoff(gax.Backoff{
			Initial:    currentSource.Retry.InitialBackOffDuration,
			Max:        currentSource.Retry.MaxBackOffDuration,
			Multiplier: currentSource.Retry.BackOffMultiplier,
		}),
		// RetryAlways will retry the operation even if it is non-idempotent.
		// Since we are only reading, the operation is always idempotent
		storage.WithPolicy(storage.RetryAlways),
	)
	scheduler := newScheduler(publisher, bucket, currentSource, &input.config, st, metrics, log)

	return scheduler.schedule(ctx)
}
