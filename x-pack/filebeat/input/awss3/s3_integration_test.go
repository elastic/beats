// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package awss3

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/go-concert/unison"
)

const (
	fileName1         = "sample1.txt"
	fileName2         = "sample2.txt"
	visibilityTimeout = 300 * time.Second
)

// GetConfigForTest function gets aws credentials for integration tests.
func getConfigForTest(t *testing.T) config {
	t.Helper()

	awsConfig := awscommon.ConfigAWS{}
	queueURL := os.Getenv("QUEUE_URL")
	profileName := os.Getenv("AWS_PROFILE_NAME")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	config := config{
		VisibilityTimeout: visibilityTimeout,
	}
	switch {
	case queueURL == "":
		t.Fatal("$QUEUE_URL is not set in environment")
	case profileName == "" && accessKeyID == "":
		t.Fatal("$AWS_ACCESS_KEY_ID or $AWS_PROFILE_NAME not set or set to empty")
	case profileName != "":
		awsConfig.ProfileName = profileName
		config.QueueURL = queueURL
		config.AwsConfig = awsConfig
		return config
	case secretAccessKey == "":
		t.Fatal("$AWS_SECRET_ACCESS_KEY not set or set to empty")
	}

	awsConfig.AccessKeyID = accessKeyID
	awsConfig.SecretAccessKey = secretAccessKey
	if sessionToken != "" {
		awsConfig.SessionToken = sessionToken
	}
	config.AwsConfig = awsConfig
	return config
}

func defaultTestConfig() *common.Config {
	return common.MustNewConfigFrom(common.MapStr{
		"queue_url": os.Getenv("QUEUE_URL"),
		"file_selectors": []common.MapStr{
			{
				"regex":     strings.Replace(fileName1, ".", "\\.", -1),
				"max_bytes": 4096,
			},
			{
				"regex":     strings.Replace(fileName2, ".", "\\.", -1),
				"max_bytes": 4096,
				"multiline": common.MapStr{
					"pattern": "^<Event",
					"negate":  true,
					"match":   "after",
				},
			},
		},
	})
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("s3_test"),
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
}

func setupInput(t *testing.T, cfg *common.Config) (*s3Collector, chan beat.Event) {
	inp, err := Plugin().Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := newV2Context()
	t.Cleanup(cancel)

	client := pubtest.NewChanClient(0)
	pipeline := pubtest.ConstClient(client)
	collector, err := inp.(*s3Input).createCollector(ctx, pipeline)
	if err != nil {
		t.Fatal(err)
	}
	return collector, client.Channel
}

func setupCollector(t *testing.T, cfg *common.Config, mock bool) (*s3Collector, chan beat.Event) {
	collector, receiver := setupInput(t, cfg)
	if mock {
		svcS3 := &MockS3Client{}
		svcSQS := &MockSQSClient{}
		collector.sqs = svcSQS
		collector.s3 = svcS3
		return collector, receiver
	}

	config := getConfigForTest(t)
	awsConfig, err := awscommon.GetAWSCredentials(config.AwsConfig)
	if err != nil {
		t.Fatal("failed GetAWSCredentials with AWS Config: ", err)
	}

	s3BucketRegion := os.Getenv("S3_BUCKET_REGION")
	if s3BucketRegion == "" {
		t.Log("S3_BUCKET_REGION is not set, default to us-west-1")
		s3BucketRegion = "us-west-1"
	}
	awsConfig.Region = s3BucketRegion
	awsConfig = awsConfig.Copy()
	collector.sqs = sqs.New(awsConfig)
	collector.s3 = s3.New(awsConfig)
	return collector, receiver
}

func runTest(t *testing.T, cfg *common.Config, mock bool, run func(t *testing.T, collector *s3Collector, receiver chan beat.Event)) {
	collector, receiver := setupCollector(t, cfg, mock)
	run(t, collector, receiver)
}

func TestS3Input(t *testing.T) {
	runTest(t, defaultTestConfig(), false, func(t *testing.T, collector *s3Collector, receiver chan beat.Event) {
		// upload a sample log file for testing
		s3BucketNameEnv := os.Getenv("S3_BUCKET_NAME")
		if s3BucketNameEnv == "" {
			t.Fatal("failed to get S3_BUCKET_NAME")
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.run()
		}()

		for i := 0; i < 4; i++ {
			event := <-receiver
			bucketName, err := event.GetValue("aws.s3.bucket.name")
			assert.NoError(t, err)
			assert.Equal(t, s3BucketNameEnv, bucketName)

			objectKey, err := event.GetValue("aws.s3.object.key")
			assert.NoError(t, err)

			switch objectKey {
			case fileName1:
				message, err := event.GetValue("message")
				assert.NoError(t, err)
				assert.Contains(t, message, "logline")
			case fileName2:
				message, err := event.GetValue("message")
				assert.NoError(t, err)
				assert.Contains(t, message, "<Event>")
				assert.Contains(t, message, "</Event>")
			default:
				t.Fatalf("object key %s is unknown", objectKey)
			}
		}
	})
}

// MockSQSClient struct is used for unit tests.
type MockSQSClient struct {
	sqsiface.ClientAPI
}

var (
	sqsMessageTest = "{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\"," +
		"\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\"," +
		"\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\"," +
		"\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54\"}}}]}"
)

func (m *MockSQSClient) ReceiveMessageRequest(input *sqs.ReceiveMessageInput) sqs.ReceiveMessageRequest {
	httpReq, _ := http.NewRequest("", "", nil)
	return sqs.ReceiveMessageRequest{
		Request: &awssdk.Request{
			Data: &sqs.ReceiveMessageOutput{
				Messages: []sqs.Message{
					{Body: awssdk.String(sqsMessageTest)},
				},
			},
			HTTPRequest: httpReq,
		},
	}
}

func (m *MockSQSClient) DeleteMessageRequest(input *sqs.DeleteMessageInput) sqs.DeleteMessageRequest {
	httpReq, _ := http.NewRequest("", "", nil)
	return sqs.DeleteMessageRequest{
		Request: &awssdk.Request{
			Data:        &sqs.DeleteMessageOutput{},
			HTTPRequest: httpReq,
		},
	}
}

func (m *MockSQSClient) ChangeMessageVisibilityRequest(input *sqs.ChangeMessageVisibilityInput) sqs.ChangeMessageVisibilityRequest {
	httpReq, _ := http.NewRequest("", "", nil)
	return sqs.ChangeMessageVisibilityRequest{
		Request: &awssdk.Request{
			Data:        &sqs.ChangeMessageVisibilityOutput{},
			HTTPRequest: httpReq,
		},
	}
}

func TestMockS3Input(t *testing.T) {
	defer resources.NewGoroutinesChecker().Check(t)
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"queue_url": "https://sqs.ap-southeast-1.amazonaws.com/123456/test",
	})

	runTest(t, cfg, true, func(t *testing.T, collector *s3Collector, receiver chan beat.Event) {
		defer collector.cancellation.Done()
		defer collector.publisher.Close()

		output, err := collector.receiveMessage(collector.sqs, collector.visibilityTimeout)
		assert.NoError(t, err)

		var grp unison.MultiErrGroup
		errC := make(chan error)
		defer close(errC)
		grp.Go(func() (err error) {
			return collector.processMessage(collector.s3, output.Messages[0], errC)
		})

		event := <-receiver
		bucketName, err := event.GetValue("aws.s3.bucket.name")
		assert.NoError(t, err)
		assert.Equal(t, "test-s3-ks-2", bucketName)
	})
}
