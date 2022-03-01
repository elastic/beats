// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// See _meta/terraform/README.md for integration test usage instructions.

//go:build integration && aws
// +build integration,aws

package awscloudwatch

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const (
	inputID            = "test_id"
	message1           = "test1"
	message2           = "test2"
	terraformOutputYML = "_meta/terraform/outputs.yml"
	logGroupNamePrefix = "filebeat-log-group-integtest-"
)

var cloudwatchConfig = common.MapStr{
	"start_position":    "beginning",
	"scan_frequency":    10 * time.Second,
	"api_timeout":       120 * time.Second,
	"number_of_workers": 1,
}

type terraformOutputData struct {
	AWSRegion  string `yaml:"aws_region"`
	LogGroup1  string `yaml:"log_group_name_1"`
	LogGroup2  string `yaml:"log_group_name_2"`
	LogStream1 string `yaml:"log_stream_name_1"`
	LogStream2 string `yaml:"log_stream_name_2"`
}

func getTerraformOutputs(t *testing.T) terraformOutputData {
	t.Helper()

	ymlData, err := ioutil.ReadFile(terraformOutputYML)
	if os.IsNotExist(err) {
		t.Skipf("Run 'terraform apply' in %v to setup CloudWatch log groups and log streams for the test.", filepath.Dir(terraformOutputYML))
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

func assertMetric(t *testing.T, snapshot common.MapStr, name string, value interface{}) {
	n, _ := snapshot.GetValue(inputID + "." + name)
	assert.EqualValues(t, value, n, name)
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger(inputName).With("id", inputID),
		ID:          inputID,
		Cancelation: ctx,
	}, cancel
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

func createInput(t *testing.T, cfg *common.Config) *cloudwatchInput {
	inputV2, err := Plugin(openTestStatestore()).Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return inputV2.(*cloudwatchInput)
}

func makeTestConfigWithLogGroupNamePrefix(regionName string) *common.Config {
	return common.MustNewConfigFrom(fmt.Sprintf(`---
log_group_name_prefix: %s
region_name: %s
`, logGroupNamePrefix, regionName))
}

func uploadLogMessage(t *testing.T, svc cloudwatchlogsiface.ClientAPI, message string, timestamp int64, logGroupName string, logStreamName string) {
	describeLogStreamsInput := cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        awssdk.String(logGroupName),
		LogStreamNamePrefix: awssdk.String(logStreamName),
	}

	reqDescribeLogStreams := svc.DescribeLogStreamsRequest(&describeLogStreamsInput)
	resp, err := reqDescribeLogStreams.Send(context.TODO())
	if err != nil {
		t.Fatalf("Failed to describe log stream %q in log group %q: %v", logStreamName, logGroupName, err)
	}

	if len(resp.LogStreams) != 1 {
		t.Fatalf("Describe log stream %q in log group %q should return 1 and only 1 value", logStreamName, logGroupName)
	}

	inputLogEvent := cloudwatchlogs.InputLogEvent{
		Message:   awssdk.String(message),
		Timestamp: awssdk.Int64(timestamp),
	}

	reqPutLogEvents := svc.PutLogEventsRequest(
		&cloudwatchlogs.PutLogEventsInput{
			LogEvents:     []cloudwatchlogs.InputLogEvent{inputLogEvent},
			LogGroupName:  awssdk.String(logGroupName),
			LogStreamName: awssdk.String(logStreamName),
			SequenceToken: resp.LogStreams[0].UploadSequenceToken,
		})
	_, err = reqPutLogEvents.Send(context.TODO())
	if err != nil {
		t.Fatalf("Failed to upload message %q into log stream %q in log group %q: %v", message, logStreamName, logGroupName, err)
	}
}

func TestInputWithLogGroupNamePrefix(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and SQS and must be executed manually.
	tfConfig := getTerraformOutputs(t)

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Region = tfConfig.AWSRegion

	// upload log messages for testing
	svc := cloudwatchlogs.New(cfg)
	currentTime := time.Now()
	timestamp := currentTime.UnixNano() / int64(time.Millisecond)

	uploadLogMessage(t, svc, message1, timestamp, tfConfig.LogGroup1, tfConfig.LogStream1)
	uploadLogMessage(t, svc, message2, timestamp, tfConfig.LogGroup2, tfConfig.LogStream2)

	// sleep for 30 seconds to wait for the log messages to show up
	time.Sleep(30 * time.Second)

	cloudwatchInput := createInput(t, makeTestConfigWithLogGroupNamePrefix(tfConfig.AWSRegion))
	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(30*time.Second, func() {
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
		return cloudwatchInput.Run(inputCtx, pipeline)
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	snap := common.MapStr(monitoring.CollectStructSnapshot(
		monitoring.GetNamespace("dataset").GetRegistry(),
		monitoring.Full,
		false))
	t.Log(snap.StringToPrint())

	assertMetric(t, snap, "log_events_received_total", 2)
	assertMetric(t, snap, "log_groups_total", 2)
	assertMetric(t, snap, "cloudwatch_events_created_total", 2)
}
