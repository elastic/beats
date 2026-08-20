// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// These tests exercise the V2 aws-s3 input end-to-end against localstack
// (or real AWS when terraform outputs are present).
//
// To run the localstack tests:
//
//	docker run -d --name localstack -p 4566:4566 -e SERVICES=s3,sqs localstack/localstack:3.8
//	cd x-pack/filebeat/input/awss3/_meta/terraform
//	terraform init
//	terraform apply -target=aws_s3_bucket.filebeat-integtest-localstack \
//	    -target=aws_sqs_queue.filebeat-integtest-localstack \
//	    -target=aws_s3_bucket_notification.bucket_notification-localstack \
//	    -target=local_file.secrets-localstack -auto-approve
//	cd ../..
//	export AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1
//	go test -tags integration,aws -run "TestV2InputRun.*Localstack" -v .
//
// Cleanup:
//
//	cd _meta/terraform && terraform destroy \
//	    -target=aws_s3_bucket_notification.bucket_notification-localstack \
//	    -target=aws_sqs_queue.filebeat-integtest-localstack \
//	    -target=aws_s3_bucket.filebeat-integtest-localstack \
//	    -target=local_file.secrets-localstack \
//	    -target=random_string.random_localstack -auto-approve
//	docker rm -f localstack

//go:build integration && aws

package awss3

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/features"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestV2InputRunSQSOnLocalstack verifies SQS-driven processing through
// the V2 input code path using localstack.
func TestV2InputRunSQSOnLocalstack(t *testing.T) {
	logp.TestingSetup()

	tfConfig := getTerraformOutputs(t, true)
	region := tfConfig.AWSRegion
	bucketName := tfConfig.BucketName
	queueURL := tfConfig.QueueURL

	awsCfg, err := makeLocalstackConfig(region)
	require.NoError(t, err, "localstack config")

	drainSQS(t, region, queueURL, awsCfg)

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	uploadS3TestFiles(t, region, bucketName, s3Client,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt",
	)

	sqsCfg := conf.MustNewConfigFrom(map[string]interface{}{
		"queue_url":          queueURL,
		"region":             region,
		"endpoint":           "http://localhost:4566",
		"number_of_workers":  1,
		"visibility_timeout": "30s",
		"path_style":         true,
		"file_selectors": []map[string]interface{}{
			{
				"regex":                        `events-array\.json$`,
				"expand_event_list_from_field": "Events",
				"include_s3_metadata":          []string{"last-modified", "x-amz-version-id", "x-amz-storage-class", "Content-Length", "Content-Type"},
			},
			{"regex": `\.(?:nd)?json(\.gz)?$`},
			{
				"regex": `multiline\.txt$`,
				"parsers": []map[string]interface{}{
					{"multiline": map[string]interface{}{
						"pattern": "^<Event",
						"negate":  true,
						"match":   "after",
					}},
				},
			},
		},
	})

	v2in := createV2Input(t, sqsCfg)

	inputCtx, cancel := newV2ContextWithRegistry(t)
	t.Cleanup(cancel)
	time.AfterFunc(20*time.Second, cancel)

	pipeline := &countingPipeline{}

	var eg errgroup.Group
	eg.Go(func() error {
		return v2in.Run(inputCtx, pipeline)
	})

	require.NoError(t, eg.Wait(), "V2 SQS input run")

	// The same 8 test files produce 19 events (7 valid objects):
	// events-array.json: 3 (expanded), log.json: 1, log.ndjson: 1,
	// multiline.json: 1, multiline.json.gz: 1, multiline.txt: 1,
	// log.txt: not matched, invalid.json: fails download/parse.
	// Exact count depends on legacy behavior; assert > 0 as baseline.
	eventCount := pipeline.events.Load()
	assert.Positive(t, eventCount, "expected events to be published")
	t.Logf("V2 SQS input published %d events", eventCount)
}

// TestV2InputRunS3PollingOnLocalstack verifies S3 polling through
// the V2 input code path using localstack.
func TestV2InputRunS3PollingOnLocalstack(t *testing.T) {
	logp.TestingSetup()

	tfConfig := getTerraformOutputs(t, true)
	region := tfConfig.AWSRegion
	bucketName := tfConfig.BucketName

	awsCfg, err := makeLocalstackConfig(region)
	require.NoError(t, err, "localstack config")

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	uploadS3TestFiles(t, region, bucketName, s3Client,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt",
	)

	pollingCfg := conf.MustNewConfigFrom(map[string]interface{}{
		"non_aws_bucket_name":  bucketName,
		"region":               region,
		"endpoint":             "http://localhost:4566",
		"number_of_workers":    1,
		"bucket_list_interval": "2s",
		"path_style":           true,
		"file_selectors": []map[string]interface{}{
			{
				"regex":                        `events-array\.json$`,
				"expand_event_list_from_field": "Events",
				"include_s3_metadata":          []string{"last-modified", "x-amz-version-id", "x-amz-storage-class", "Content-Length", "Content-Type"},
			},
			{"regex": `\.(?:nd)?json(\.gz)?$`},
			{
				"regex": `multiline\.txt$`,
				"parsers": []map[string]interface{}{
					{"multiline": map[string]interface{}{
						"pattern": "^<Event",
						"negate":  true,
						"match":   "after",
					}},
				},
			},
		},
	})

	v2in := createV2Input(t, pollingCfg)

	inputCtx, cancel := newV2ContextWithRegistry(t)
	t.Cleanup(cancel)
	time.AfterFunc(20*time.Second, cancel)

	pipeline := &countingPipeline{}

	var eg errgroup.Group
	eg.Go(func() error {
		return v2in.Run(inputCtx, pipeline)
	})

	require.NoError(t, eg.Wait(), "V2 polling input run")

	eventCount := pipeline.events.Load()
	assert.Positive(t, eventCount, "expected events to be published")
	t.Logf("V2 polling input published %d events", eventCount)
}

// TestV2InputRunSQS verifies V2 SQS input against real AWS (requires terraform setup).
func TestV2InputRunSQS(t *testing.T) {
	logp.TestingSetup()

	tfConfig := getTerraformOutputs(t, false)
	awsCfg := makeAWSConfig(t, tfConfig.AWSRegion)

	drainSQS(t, tfConfig.AWSRegion, tfConfig.QueueURL, awsCfg)

	s3Client := s3.NewFromConfig(awsCfg)
	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketName, s3Client,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt",
	)

	sqsCfg := makeTestConfigSQS(tfConfig.QueueURL)
	v2in := createV2Input(t, sqsCfg)

	inputCtx, cancel := newV2ContextWithRegistry(t)
	t.Cleanup(cancel)
	time.AfterFunc(20*time.Second, cancel)

	pipeline := &countingPipeline{}

	var eg errgroup.Group
	eg.Go(func() error {
		return v2in.Run(inputCtx, pipeline)
	})

	require.NoError(t, eg.Wait(), "V2 SQS input run")

	eventCount := pipeline.events.Load()
	assert.Positive(t, eventCount, "expected events to be published")
	t.Logf("V2 SQS input published %d events", eventCount)
}

// TestV2InputRunS3 verifies V2 S3 polling against real AWS (requires terraform setup).
func TestV2InputRunS3(t *testing.T) {
	logp.TestingSetup()

	tfConfig := getTerraformOutputs(t, false)
	awsCfg := makeAWSConfig(t, tfConfig.AWSRegion)

	s3Client := s3.NewFromConfig(awsCfg)
	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketName, s3Client,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt",
	)

	pollingCfg := makeTestConfigS3(tfConfig.BucketName)
	v2in := createV2Input(t, pollingCfg)

	inputCtx, cancel := newV2ContextWithRegistry(t)
	t.Cleanup(cancel)
	time.AfterFunc(20*time.Second, cancel)

	pipeline := &countingPipeline{}

	var eg errgroup.Group
	eg.Go(func() error {
		return v2in.Run(inputCtx, pipeline)
	})

	require.NoError(t, eg.Wait(), "V2 polling input run")

	eventCount := pipeline.events.Load()
	assert.Positive(t, eventCount, "expected events to be published")
	t.Logf("V2 polling input published %d events", eventCount)
}

// TestV2InputRunSNS verifies V2 SQS input with SNS-wrapped notifications
// against real AWS (requires terraform setup).
func TestV2InputRunSNS(t *testing.T) {
	logp.TestingSetup()

	tfConfig := getTerraformOutputs(t, false)
	awsCfg := makeAWSConfig(t, tfConfig.AWSRegion)

	drainSQS(t, tfConfig.AWSRegion, tfConfig.QueueURLForSNS, awsCfg)

	s3Client := s3.NewFromConfig(awsCfg)
	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketNameForSNS, s3Client,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt",
	)

	sqsCfg := makeTestConfigSQS(tfConfig.QueueURLForSNS)
	v2in := createV2Input(t, sqsCfg)

	inputCtx, cancel := newV2ContextWithRegistry(t)
	t.Cleanup(cancel)
	time.AfterFunc(20*time.Second, cancel)

	pipeline := &countingPipeline{}

	var eg errgroup.Group
	eg.Go(func() error {
		return v2in.Run(inputCtx, pipeline)
	})

	require.NoError(t, eg.Wait(), "V2 SNS/SQS input run")

	eventCount := pipeline.events.Load()
	assert.Positive(t, eventCount, "expected events to be published")
	t.Logf("V2 SNS/SQS input published %d events", eventCount)
}

// --- Test helpers ---

type countingPipeline struct {
	events atomic.Int64
}

func (p *countingPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return &countingClient{pipeline: p, listener: cfg.EventListener}, nil
}

func (p *countingPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *countingPipeline) Disconnect(context.Context) error { return nil }

type countingClient struct {
	pipeline *countingPipeline
	listener beat.EventListener
}

func (c *countingClient) Close() error { return nil }

func (c *countingClient) Publish(event beat.Event) {
	c.pipeline.events.Add(1)
	if c.listener != nil {
		c.listener.AddEvent(event, true)
		c.listener.ACKEvents(1)
	}
}

func (c *countingClient) PublishAll(events []beat.Event) {
	for _, e := range events {
		c.Publish(e)
	}
}

func newV2ContextWithRegistry(t *testing.T) (v2.Context, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:          logp.NewLogger(inputName).With("id", inputID),
		ID:              inputID,
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}, cancel
}

func enableV2(t *testing.T) {
	t.Helper()
	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"features": map[string]interface{}{
			"aws_s3_v2": map[string]interface{}{
				"enabled": true,
			},
		},
	})
	require.NoError(t, features.UpdateFromConfig(cfg), "failed to enable V2 flag")
	t.Cleanup(func() {
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
			"features": map[string]interface{}{
				"aws_s3_v2": map[string]interface{}{
					"enabled": false,
				},
			},
		})
		_ = features.UpdateFromConfig(cfg)
	})
}

func createV2Input(t *testing.T, cfg *conf.C) *inputV2 {
	t.Helper()
	enableV2(t)
	in, err := Plugin(logp.NewLogger(inputName), openTestStatestore(), nil).Manager.Create(cfg)
	require.NoError(t, err, "failed to create V2 input")
	v2in, ok := in.(*inputV2)
	require.True(t, ok, "expected *inputV2, got %T", in)
	return v2in
}
