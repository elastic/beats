// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

// inputV2 is the V2 implementation of the aws-s3 input, gated behind
// the features.AwsS3V2 flag. It is a drop-in replacement for the legacy
// SQS reader and S3 poller inputs.
type inputV2 struct {
	config config
	store  statestore.States
	path   *paths.Path
	log    *logp.Logger
}

func newInputV2(cfg config, store statestore.States, path *paths.Path, log *logp.Logger) (*inputV2, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid aws-s3 v2 config: %w", err)
	}
	return &inputV2{
		config: cfg,
		store:  store,
		path:   path,
		log:    log,
	}, nil
}

func (*inputV2) Name() string { return inputName }

func (*inputV2) Test(v2.TestContext) error { return nil }

func (in *inputV2) Run(ctx v2.Context, pipeline beat.Pipeline) error {
	log := ctx.Logger.With("queue_url", in.config.QueueURL, "bucket_arn", in.config.getBucketARN())
	log.Info("aws-s3 V2 input starting")

	// TODO(phase 2+): wire object processing pipeline, discovery, flow control.

	<-ctx.Cancelation.Done()
	log.Info("aws-s3 V2 input stopping")
	return nil
}
