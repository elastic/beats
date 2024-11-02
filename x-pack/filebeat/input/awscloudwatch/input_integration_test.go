// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// See _meta/terraform/README.md for integration test usage instructions.

//go:build integration && aws

package awscloudwatch

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	inputID            = "test_id"
	message1           = "test1"
	message2           = "test2"
	terraformOutputYML = "_meta/terraform/outputs.yml"
	logGroupNamePrefix = "filebeat-log-group-integtest-"
)

type terraformOutputData struct {
	AWSRegion  string `yaml:"aws_region"`
	LogGroup1  string `yaml:"log_group_name_1"`
	LogGroup2  string `yaml:"log_group_name_2"`
	LogStream1 string `yaml:"log_stream_name_1"`
	LogStream2 string `yaml:"log_stream_name_2"`
}

func getTerraformOutputs(t *testing.T) terraformOutputData {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)
	ymlData, err := ioutil.ReadFile(path.Join(path.Dir(filename), terraformOutputYML))
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

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger(inputName).With("id", inputID),
		ID:          inputID,
		Cancelation: ctx,
	}, cancel
}

func createInput(t *testing.T, cfg *conf.C) *cloudwatchInput {
	inputV2, err := Plugin().Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return inputV2.(*cloudwatchInput)
}

func makeTestConfigWithLogGroupNamePrefix(regionName string) *conf.C {
	return conf.MustNewConfigFrom(fmt.Sprintf(`---
log_group_name_prefix: %s
region_name: %s
`, logGroupNamePrefix, regionName))
}

func uploadLogMessage(t *testing.T, svc *cloudwatchlogs.Client, message string, timestamp int64, logGroupName string, logStreamName string) {
	describeLogStreamsInput := cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        awssdk.String(logGroupName),
		LogStreamNamePrefix: awssdk.String(logStreamName),
	}

	resp, err := svc.DescribeLogStreams(context.TODO(), &describeLogStreamsInput)
	if err != nil {
		t.Fatalf("Failed to describe log stream %q in log group %q: %v", logStreamName, logGroupName, err)
	}

	if len(resp.LogStreams) != 1 {
		t.Fatalf("Describe log stream %q in log group %q should return 1 and only 1 value", logStreamName, logGroupName)
	}

	inputLogEvent := cloudwatchlogstypes.InputLogEvent{
		Message:   awssdk.String(message),
		Timestamp: awssdk.Int64(timestamp),
	}

	_, err = svc.PutLogEvents(context.TODO(), &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     []cloudwatchlogstypes.InputLogEvent{inputLogEvent},
		LogGroupName:  awssdk.String(logGroupName),
		LogStreamName: awssdk.String(logStreamName),
		SequenceToken: resp.LogStreams[0].UploadSequenceToken,
	})
	if err != nil {
		t.Fatalf("Failed to upload message %q into log stream %q in log group %q: %v", message, logStreamName, logGroupName, err)
	}
}

func TestInputWithLogGroupNamePrefix(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and SQS and must be executed manually.
	tfConfig := getTerraformOutputs(t)

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Region = tfConfig.AWSRegion

	// upload log messages for testing
	svc := cloudwatchlogs.NewFromConfig(cfg)
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

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		pipeline := pubtest.PublisherWithClient(client)
		return cloudwatchInput.Run(inputCtx, pipeline)
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, cloudwatchInput.metrics.logEventsReceivedTotal.Get(), 2)
	assert.EqualValues(t, cloudwatchInput.metrics.logGroupsTotal.Get(), 2)
	assert.EqualValues(t, cloudwatchInput.metrics.cloudwatchEventsCreatedTotal.Get(), 2)
}
