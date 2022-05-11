// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// See _meta/terraform/README.md for integration test usage instructions.

//go:build integration && aws
// +build integration,aws

package awss3

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	inputID = "test_id"
)

const (
	terraformOutputYML = "_meta/terraform/outputs.yml"
)

type terraformOutputData struct {
	AWSRegion        string `yaml:"aws_region"`
	BucketName       string `yaml:"bucket_name"`
	QueueURL         string `yaml:"queue_url"`
	BucketNameForSNS string `yaml:"bucket_name_for_sns"`
	QueueURLForSNS   string `yaml:"queue_url_for_sns"`
}

func getTerraformOutputs(t *testing.T) terraformOutputData {
	t.Helper()

	ymlData, err := ioutil.ReadFile(terraformOutputYML)
	if os.IsNotExist(err) {
		t.Skipf("Run 'terraform apply' in %v to setup S3 and SQS for the test.", filepath.Dir(terraformOutputYML))
	}
	if err != nil {
		t.Fatalf("failed reading terraform output data: %v", err)
	}

	var rtn terraformOutputData
	dec := yaml.NewDecoder(bytes.NewReader(ymlData))
	dec.SetStrict(true)
	if err = dec.Decode(&rtn); err != nil {
		t.Fatal(err)
	}

	return rtn
}

func makeTestConfigS3(s3bucket string) *conf.C {
	return conf.MustNewConfigFrom(fmt.Sprintf(`---
bucket_arn: aws:s3:::%s
number_of_workers: 1
file_selectors:
-
  regex: 'events-array.json$'
  expand_event_list_from_field: Events
  content_type: application/json
  include_s3_metadata:
    - last-modified
    - x-amz-version-id
    - x-amz-storage-class
    - Content-Length
    - Content-Type
-
  regex: '\.(?:nd)?json(\.gz)?$'
  content_type: application/json
-
  regex: 'multiline.txt$'
  parsers:
    - multiline:
        pattern: "^<Event"
        negate:  true
        match:   after
`, s3bucket))
}

func makeTestConfigSQS(queueURL string) *conf.C {
	return conf.MustNewConfigFrom(fmt.Sprintf(`---
queue_url: %s
max_number_of_messages: 1
visibility_timeout: 30s
file_selectors:
-
  regex: 'events-array.json$'
  expand_event_list_from_field: Events
  content_type: application/json
  include_s3_metadata:
    - last-modified
    - x-amz-version-id
    - x-amz-storage-class
    - Content-Length
    - Content-Type
-
  regex: '\.(?:nd)?json(\.gz)?$'
  content_type: application/json
-
  regex: 'multiline.txt$'
  parsers:
    - multiline:
        pattern: "^<Event"
        negate:  true
        match:   after
`, queueURL))
}

type testInputStore struct {
	registry *statestore.Registry
}

func openTestStatestore() beater.StateStore {
	return &testInputStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testInputStore) Close() {
	s.registry.Close()
}

func (s *testInputStore) Access() (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}

func (s *testInputStore) CleanupInterval() time.Duration {
	return 24 * time.Hour
}

func createInput(t *testing.T, cfg *conf.C) *s3Input {
	inputV2, err := Plugin(openTestStatestore()).Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return inputV2.(*s3Input)
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger(inputName).With("id", inputID),
		ID:          inputID,
		Cancelation: ctx,
	}, cancel
}

func TestInputRunSQS(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and SQS and must be executed manually.
	tfConfig := getTerraformOutputs(t)

	// Ensure SQS is empty before testing.
	drainSQS(t, tfConfig.AWSRegion, tfConfig.QueueURL)

	// Ensure metrics are removed before testing.
	monitoring.GetNamespace("dataset").GetRegistry().Remove(inputID)

	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketName,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt", // Skipped (no match).
	)

	s3Input := createInput(t, makeTestConfigSQS(tfConfig.QueueURL))

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	client := pubtest.NewChanClient(0)
	defer close(client.Channel)
	go func() {
		for event := range client.Channel {
			// Fake the ACK handling that's not implemented in pubtest.
			event.Private.(*awscommon.EventACKTracker).ACK()
		}
	}()

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		pipeline := pubtest.PublisherWithClient(client)
		return s3Input.Run(inputCtx, pipeline)
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	snap := mapstr.M(monitoring.CollectStructSnapshot(
		monitoring.GetNamespace("dataset").GetRegistry(),
		monitoring.Full,
		false))
	t.Log(snap.StringToPrint())

	assertMetric(t, snap, "sqs_messages_received_total", 8) // S3 could batch notifications.
	assertMetric(t, snap, "sqs_messages_inflight_gauge", 0)
	assertMetric(t, snap, "sqs_messages_deleted_total", 7)
	assertMetric(t, snap, "sqs_messages_returned_total", 1) // Invalid JSON is returned so that it can eventually be DLQed.
	assertMetric(t, snap, "sqs_visibility_timeout_extensions_total", 0)
	assertMetric(t, snap, "s3_objects_inflight_gauge", 0)
	assertMetric(t, snap, "s3_objects_requested_total", 7)
	assertMetric(t, snap, "s3_events_created_total", 12)
}

func TestInputRunS3(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and must be executed manually.
	tfConfig := getTerraformOutputs(t)

	// Ensure metrics are removed before testing.
	monitoring.GetNamespace("dataset").GetRegistry().Remove(inputID)

	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketName,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt", // Skipped (no match).
	)

	s3Input := createInput(t, makeTestConfigS3(tfConfig.BucketName))

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	client := pubtest.NewChanClient(0)
	defer close(client.Channel)
	go func() {
		for event := range client.Channel {
			// Fake the ACK handling that's not implemented in pubtest.
			event.Private.(*awscommon.EventACKTracker).ACK()
		}
	}()

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		pipeline := pubtest.PublisherWithClient(client)
		return s3Input.Run(inputCtx, pipeline)
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	snap := mapstr.M(monitoring.CollectStructSnapshot(
		monitoring.GetNamespace("dataset").GetRegistry(),
		monitoring.Full,
		false))
	t.Log(snap.StringToPrint())

	assertMetric(t, snap, "s3_objects_inflight_gauge", 0)
	assertMetric(t, snap, "s3_objects_requested_total", 7)
	assertMetric(t, snap, "s3_objects_listed_total", 8)
	assertMetric(t, snap, "s3_objects_processed_total", 7)
	assertMetric(t, snap, "s3_objects_acked_total", 6)
	assertMetric(t, snap, "s3_events_created_total", 12)
}

func assertMetric(t *testing.T, snapshot mapstr.M, name string, value interface{}) {
	n, _ := snapshot.GetValue(inputID + "." + name)
	assert.EqualValues(t, value, n, name)
}

func uploadS3TestFiles(t *testing.T, region, bucket string, filenames ...string) {
	t.Helper()

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Region = region

	uploader := s3manager.NewUploader(cfg)

	for _, filename := range filenames {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("Failed to open file %q, %v", filename, err)
		}

		// Upload the file to S3.
		result, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(filepath.Base(filename)),
			Body:   bytes.NewReader(data),
		})
		if err != nil {
			t.Fatalf("Failed to upload file %q: %v", filename, err)
		}
		t.Logf("File uploaded to %s", result.Location)
	}
}

func drainSQS(t *testing.T, region string, queueURL string) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Region = region

	sqs := &awsSQSAPI{
		client:            sqs.New(cfg),
		queueURL:          queueURL,
		apiTimeout:        1 * time.Minute,
		visibilityTimeout: 30 * time.Second,
		longPollWaitTime:  10,
	}

	ctx := context.Background()
	var deletedCount int
	for {
		msgs, err := sqs.ReceiveMessage(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			if err = sqs.DeleteMessage(ctx, &msg); err != nil {
				t.Fatal(err)
			}
			deletedCount++
		}
	}
	t.Logf("Drained %d SQS messages.", deletedCount)
}

func TestGetBucketNameFromARN(t *testing.T) {
	bucketName := getBucketNameFromARN("arn:aws:s3:::my_corporate_bucket")
	assert.Equal(t, "my_corporate_bucket", bucketName)
}

func TestGetRegionForBucketARN(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and must be executed manually.
	tfConfig := getTerraformOutputs(t)

	awsConfig, err := external.LoadDefaultAWSConfig()
	if err != nil {
		t.Fatal(err)
	}

	s3Client := s3.New(awscommon.EnrichAWSConfigWithEndpoint("", "s3", "", awsConfig))

	regionName, err := getRegionForBucket(context.Background(), s3Client, getBucketNameFromARN(tfConfig.BucketName))
	assert.Equal(t, tfConfig.AWSRegion, regionName)
}

func TestPaginatorListPrefix(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and must be executed manually.
	tfConfig := getTerraformOutputs(t)

	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketName,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt", // Skipped (no match).
	)

	awsConfig, err := external.LoadDefaultAWSConfig()
	awsConfig.Region = tfConfig.AWSRegion
	if err != nil {
		t.Fatal(err)
	}

	s3Client := s3.New(awscommon.EnrichAWSConfigWithEndpoint("", "s3", "", awsConfig))

	s3API := &awsS3API{
		client: s3Client,
	}

	var objects []string
	paginator := s3API.ListObjectsPaginator(tfConfig.BucketName, "log")
	for paginator.Next(context.Background()) {
		page := paginator.CurrentPage()
		for _, object := range page.Contents {
			objects = append(objects, *object.Key)
		}
	}

	assert.NoError(t, paginator.Err())

	expected := []string{
		"log.json",
		"log.ndjson",
		"log.txt",
	}

	assert.Equal(t, expected, objects)
}

func TestInputRunSNS(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3, SNS and SQS and must be executed manually.
	tfConfig := getTerraformOutputs(t)

	// Ensure SQS is empty before testing.
	drainSQS(t, tfConfig.AWSRegion, tfConfig.QueueURLForSNS)

	// Ensure metrics are removed before testing.
	monitoring.GetNamespace("dataset").GetRegistry().Remove(inputID)

	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketNameForSNS,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt", // Skipped (no match).
	)

	s3Input := createInput(t, makeTestConfigSQS(tfConfig.QueueURLForSNS))

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	client := pubtest.NewChanClient(0)
	defer close(client.Channel)
	go func() {
		for event := range client.Channel {
			event.Private.(*awscommon.EventACKTracker).ACK()
		}
	}()

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		pipeline := pubtest.PublisherWithClient(client)
		return s3Input.Run(inputCtx, pipeline)
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	snap := mapstr.M(monitoring.CollectStructSnapshot(
		monitoring.GetNamespace("dataset").GetRegistry(),
		monitoring.Full,
		false))
	t.Log(snap.StringToPrint())

	assertMetric(t, snap, "sqs_messages_received_total", 8) // S3 could batch notifications.
	assertMetric(t, snap, "sqs_messages_inflight_gauge", 0)
	assertMetric(t, snap, "sqs_messages_deleted_total", 7)
	assertMetric(t, snap, "sqs_messages_returned_total", 1) // Invalid JSON is returned so that it can eventually be DLQed.
	assertMetric(t, snap, "sqs_visibility_timeout_extensions_total", 0)
	assertMetric(t, snap, "s3_objects_inflight_gauge", 0)
	assertMetric(t, snap, "s3_objects_requested_total", 7)
	assertMetric(t, snap, "s3_events_created_total", 12)
}
