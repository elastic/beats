// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// See _meta/terraform/README.md for integration test usage instructions.

//go:build integration && aws

package awss3

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	inputID = "test_id"
)

const (
	terraformOutputYML   = "_meta/terraform/outputs.yml"
	terraformOutputLsYML = "_meta/terraform/outputs-localstack.yml"
)

type terraformOutputData struct {
	AWSRegion        string `yaml:"aws_region"`
	BucketName       string `yaml:"bucket_name"`
	QueueURL         string `yaml:"queue_url"`
	BucketNameForSNS string `yaml:"bucket_name_for_sns"`
	QueueURLForSNS   string `yaml:"queue_url_for_sns"`
	BucketNameForEB  string `yaml:"bucket_name_for_eventbridge"`
	QueueURLForEB    string `yaml:"queue_url_for_eventbridge"`
}

func getTerraformOutputs(t *testing.T, isLocalStack bool) terraformOutputData {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)

	var outputFile string
	if isLocalStack {
		outputFile = terraformOutputLsYML
	} else {
		outputFile = terraformOutputYML
	}

	ymlData, err := ioutil.ReadFile(path.Join(path.Dir(filename), outputFile))
	if os.IsNotExist(err) {
		t.Skipf("Run 'terraform apply' in %v to setup S3 and SQS for the test.", filepath.Dir(outputFile))
	}
	if err != nil {
		t.Fatalf("failed reading terraform output data: %v", err)
	}

	var rtn terraformOutputData
	dec := yaml.NewDecoder(bytes.NewReader(ymlData))
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
  include_s3_metadata:
    - last-modified
    - x-amz-version-id
    - x-amz-storage-class
    - Content-Length
    - Content-Type
-
  regex: '\.(?:nd)?json(\.gz)?$'
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
number_of_workers: 1
visibility_timeout: 30s
region: us-east-1
file_selectors:
-
  regex: 'events-array.json$'
  expand_event_list_from_field: Events
  include_s3_metadata:
    - last-modified
    - x-amz-version-id
    - x-amz-storage-class
    - Content-Length
    - Content-Type
-
  regex: '\.(?:nd)?json(\.gz)?$'
-
  regex: 'multiline.txt$'
  parsers:
    - multiline:
        pattern: "^<Event"
        negate:  true
        match:   after
`, queueURL))
}

func createSQSInput(t *testing.T, cfg *conf.C) *sqsReaderInput {
	inputV2, err := Plugin(openTestStatestore()).Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return inputV2.(*sqsReaderInput)
}

func createS3Input(t *testing.T, cfg *conf.C) *s3PollerInput {
	inputV2, err := Plugin(openTestStatestore()).Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return inputV2.(*s3PollerInput)
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger(inputName).With("id", inputID),
		ID:          inputID,
		Cancelation: ctx,
	}, cancel
}

// Creates a default config for Localstack based tests
func defaultTestConfig(region, queueURL string) config {
	c := config{
		APITimeout:          120 * time.Second,
		VisibilityTimeout:   300 * time.Second,
		BucketListInterval:  120 * time.Second,
		BucketListPrefix:    "",
		SQSWaitTime:         20 * time.Second,
		SQSMaxReceiveCount:  5,
		MaxNumberOfMessages: 5,
		PathStyle:           true,
		RegionName:          region,
		QueueURL:            queueURL,
	}
	c.ReaderConfig.InitDefaults()
	return c
}

// Create an aws config for Localstack based tests
func makeLocalstackConfig(awsRegion string) (aws.Config, error) {
	awsLocalstackEndpoint := "http://localhost:4566" // Default Localstack endpoint

	// Add a custom endpointResolver to the awsConfig so that all the requests are routed to this endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           awsLocalstackEndpoint,
			SigningRegion: awsRegion,
		}, nil
	})

	return awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithRegion(awsRegion),
		awsConfig.WithEndpointResolverWithOptions(customResolver),
	)
}

// Tests reading SQS notifcation via awss3 input when an object is PUT in S3
// and a notification is generated to SQS on Localstack
func TestInputRunSQSOnLocalstack(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3,SQS and must be executed manually.
	tfConfig := getTerraformOutputs(t, true)

	// Read the necessary terraform outputs
	region := tfConfig.AWSRegion
	bucketName := tfConfig.BucketName
	queueUrl := tfConfig.QueueURL

	// Create a default config for the awss3 input
	config := defaultTestConfig(region, queueUrl)
	awsCfg, err := makeLocalstackConfig(region)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure SQS is empty before testing.
	drainSQS(t, region, queueUrl, awsCfg)

	// Upload test files to S3 to generate SQS notification
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

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	// Initialize s3Input with the test config
	s3Input := newSQSReaderInput(config, awsCfg)
	// Run S3 Input with desired context
	var errGroup errgroup.Group
	errGroup.Go(func() error {
		return s3Input.Run(inputCtx, &fakePipeline{})
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 8, s3Input.metrics.sqsMessagesReceivedTotal.Get()) // S3 could batch notifications.
	assert.EqualValues(t, 0, s3Input.metrics.sqsMessagesInflight.Get())
	assert.EqualValues(t, 7, s3Input.metrics.sqsMessagesDeletedTotal.Get())
	assert.EqualValues(t, 1, s3Input.metrics.sqsMessagesReturnedTotal.Get()) // Invalid JSON is returned so that it can eventually be DLQed.
	assert.EqualValues(t, 0, s3Input.metrics.sqsVisibilityTimeoutExtensionsTotal.Get())
	assert.EqualValues(t, 0, s3Input.metrics.s3ObjectsInflight.Get())
	assert.EqualValues(t, 8, s3Input.metrics.s3ObjectsRequestedTotal.Get())
	assert.EqualValues(t, uint64(0x13), s3Input.metrics.s3EventsCreatedTotal.Get())
	assert.Greater(t, 0.0, s3Input.metrics.sqsLagTime.Mean())
	assert.EqualValues(t, 0.0, s3Input.metrics.sqsWorkerUtilization.Get()) // Workers are reset after processing and hence utilization should be 0 at the end
}

func TestInputRunSQSWithConfig(t *testing.T) {
	tests := []struct {
		name           string
		queue_url      string
		endpoint       string
		region         string
		default_region string
		want           string
		wantErr        error
	}{
		{
			name:      "no region",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			want:      "us-east-1",
		},
		{
			name:      "no region but with long endpoint",
			queue_url: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			endpoint:  "https://s3.us-east-1.abc.xyz",
			want:      "us-east-1",
		},
		{
			name:      "no region but with short endpoint",
			queue_url: "https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			endpoint:  "https://abc.xyz",
			want:      "us-east-1",
		},
		{
			name:      "no region custom queue domain",
			queue_url: "https://sqs.us-east-1.xyz.abc/627959692251/test-s3-logs",
			wantErr:   errBadQueueURL,
		},
		{
			name:      "region",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:    "us-west-2",
			want:      "us-west-2",
		},
		{
			name:           "default_region",
			queue_url:      "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			default_region: "us-west-2",
			want:           "us-west-2",
		},
		{
			name:           "region and default_region",
			queue_url:      "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:         "us-east-2",
			default_region: "us-east-3",
			want:           "us-east-2",
		},
		{
			name:      "short_endpoint",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			endpoint:  "https://amazonaws.com",
			want:      "us-east-1",
		},
		{
			name:      "long_endpoint",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			endpoint:  "https://s3.us-east-1.amazonaws.com",
			want:      "us-east-1",
		},
		{
			name:      "region and custom short_endpoint",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:    "us-west-2",
			endpoint:  "https://.elastic.co",
			want:      "us-west-2",
		},
		{
			name:      "region and custom long_endpoint",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:    "us-west-2",
			endpoint:  "https://s3.us-east-1.elastic.co",
			want:      "us-west-2",
		},
		{
			name:      "region and short_endpoint",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:    "us-west-2",
			endpoint:  "https://amazonaws.com",
			want:      "us-west-2",
		},
		{
			name:      "region and long_endpoint",
			queue_url: "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:    "us-west-2",
			endpoint:  "https://s3.us-east-1.amazonaws.com",
			want:      "us-west-2",
		},
		{
			name:           "region and default region and short_endpoint",
			queue_url:      "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:         "us-west-2",
			default_region: "us-east-1",
			endpoint:       "https://amazonaws.com",
			want:           "us-west-2",
		},
		{
			name:           "region and default region and long_endpoint",
			queue_url:      "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			region:         "us-west-2",
			default_region: "us-east-1",
			endpoint:       "https://s3.us-east-1.amazonaws.com",
			want:           "us-west-2",
		},
	}

	for _, test := range tests {
		logp.TestingSetup()

		// Create a filebeat config using the provided test parameters
		config := ""
		if test.queue_url != "" {
			config += fmt.Sprintf("queue_url: %s \n", test.queue_url)
		}
		if test.region != "" {
			config += fmt.Sprintf("region: %s \n", test.region)
		}
		if test.default_region != "" {
			config += fmt.Sprintf("default_region: %s \n", test.default_region)
		}
		if test.endpoint != "" {
			config += fmt.Sprintf("endpoint: %s \n", test.endpoint)
		}

		s3Input := createInput(t, conf.MustNewConfigFrom(config))

		inputCtx, cancel := newV2Context()
		t.Cleanup(cancel)
		time.AfterFunc(5*time.Second, func() {
			cancel()
		})

		var errGroup errgroup.Group
		errGroup.Go(func() error {
			return s3Input.Run(inputCtx, &fakePipeline{})
		})

		if err := errGroup.Wait(); err != nil {
			// assert that err == test.wantErr
			if test.wantErr != nil {
				continue
			}
			// Print the test name to help identify the failing test
			t.Fatal(test.name, err)
		}

		// If the endpoint starts with s3, the endpoint resolver should be null at this point
		// If the endpoint does not start with s3, the endpointresolverwithoptions should be set
		// If the endpoint is not set, the endpoint resolver should be null
		if test.endpoint == "" {
			assert.Nil(t, s3Input.awsConfig.EndpointResolver, test.name)
			assert.Nil(t, s3Input.awsConfig.EndpointResolverWithOptions, test.name)
		} else if strings.HasPrefix(test.endpoint, "https://s3") {
			// S3 resolvers are added later in the code than this integration test covers
			assert.Nil(t, s3Input.awsConfig.EndpointResolver, test.name)
			assert.Nil(t, s3Input.awsConfig.EndpointResolverWithOptions, test.name)
		} else { // If the endpoint is specified but is not s3
			assert.Nil(t, s3Input.awsConfig.EndpointResolver, test.name)
			assert.NotNil(t, s3Input.awsConfig.EndpointResolverWithOptions, test.name)
		}

		assert.EqualValues(t, test.want, s3Input.awsConfig.Region, test.name)
	}
}

func TestInputRunSQS(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and SQS and must be executed manually.
	tfConfig := getTerraformOutputs(t, false)
	awsCfg := makeAWSConfig(t, tfConfig.AWSRegion)

	// Ensure SQS is empty before testing.
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
		"testdata/log.txt", // Skipped (no match).
	)

	sqsInput := createSQSInput(t, makeTestConfigSQS(tfConfig.QueueURL))

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		return sqsInput.Run(inputCtx, &fakePipeline{})
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 8, sqsInput.metrics.sqsMessagesReceivedTotal.Get()) // S3 could batch notifications.
	assert.EqualValues(t, 0, sqsInput.metrics.sqsMessagesInflight.Get())
	assert.EqualValues(t, 7, sqsInput.metrics.sqsMessagesDeletedTotal.Get())
	assert.EqualValues(t, 1, sqsInput.metrics.sqsMessagesReturnedTotal.Get()) // Invalid JSON is returned so that it can eventually be DLQed.
	assert.EqualValues(t, 0, sqsInput.metrics.sqsVisibilityTimeoutExtensionsTotal.Get())
	assert.EqualValues(t, 0, sqsInput.metrics.s3ObjectsInflight.Get())
	assert.EqualValues(t, 7, sqsInput.metrics.s3ObjectsRequestedTotal.Get())
	assert.EqualValues(t, 12, sqsInput.metrics.s3EventsCreatedTotal.Get())
	assert.Greater(t, sqsInput.metrics.sqsLagTime.Mean(), 0.0)
	assert.EqualValues(t, 0.0, sqsInput.metrics.sqsWorkerUtilization.Get()) // Workers are reset after processing and hence utilization should be 0 at the end
}

func TestInputRunS3(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and must be executed manually.
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
		"testdata/log.txt", // Skipped (no match).
	)

	s3Input := createS3Input(t, makeTestConfigS3(tfConfig.BucketName))

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		return s3Input.Run(inputCtx, &fakePipeline{})
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 0, s3Input.metrics.s3ObjectsInflight.Get())
	assert.EqualValues(t, 7, s3Input.metrics.s3ObjectsRequestedTotal.Get())
	assert.EqualValues(t, 8, s3Input.metrics.s3ObjectsListedTotal.Get())
	assert.EqualValues(t, 7, s3Input.metrics.s3ObjectsProcessedTotal.Get())
	assert.EqualValues(t, 7, s3Input.metrics.s3ObjectsAckedTotal.Get())
	assert.EqualValues(t, 12, s3Input.metrics.s3EventsCreatedTotal.Get())
}

func uploadS3TestFiles(t *testing.T, region, bucket string, s3Client *s3.Client, filenames ...string) {
	t.Helper()

	uploader := s3manager.NewUploader(s3Client)

	_, basefile, _, _ := runtime.Caller(0)
	basedir := path.Dir(basefile)
	for _, filename := range filenames {
		data, err := ioutil.ReadFile(path.Join(basedir, filename))
		if err != nil {
			t.Fatalf("Failed to open file %q, %v", filename, err)
		}

		contentType := ""
		if strings.HasSuffix(filename, "ndjson") || strings.HasSuffix(filename, "ndjson.gz") {
			contentType = contentTypeNDJSON + "; charset=UTF-8"
		} else if strings.HasSuffix(filename, "json") || strings.HasSuffix(filename, "json.gz") {
			contentType = contentTypeJSON + "; charset=UTF-8"
		}

		// Upload the file to S3.
		result, err := uploader.Upload(context.Background(), &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(filepath.Base(filename)),
			Body:        bytes.NewReader(data),
			ContentType: aws.String(contentType),
		})
		if err != nil {
			t.Fatalf("Failed to upload file %q: %v", filename, err)
		}
		t.Logf("File uploaded to %s", result.Location)
	}
}

func makeAWSConfig(t *testing.T, region string) aws.Config {
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Region = region
	return cfg
}

func drainSQS(t *testing.T, region string, queueURL string, cfg aws.Config) {
	sqs := &awsSQSAPI{
		client: sqs.NewFromConfig(cfg, func(options *sqs.Options) {
			//options.ClientLogMode = aws.LogResponseWithBody
			options.Region = region
		}),
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

func TestGetRegionFromAccessPointARN(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		arn      string
		expected string
	}{
		{
			name:     "Valid Access Point ARN",
			arn:      "arn:aws:s3:us-east-1:123456789:accesspoint/my-access-point",
			expected: "us-east-1",
		},
		{
			name:     "Invalid ARN with missing region",
			arn:      "arn:aws:s3::123456789:accesspoint/my-access-point",
			expected: "",
		},
		{
			name:     "Invalid ARN with too few parts",
			arn:      "arn:aws:s3",
			expected: "",
		},
		{
			name:     "Standard bucket ARN (not an Access Point)",
			arn:      "arn:aws:s3:::my_corporate_bucket",
			expected: "",
		},
		{
			name:     "Malformed ARN with extra colons",
			arn:      "arn:aws:s3:::us-west-2:123456789:accesspoint/my-access-point",
			expected: "",
		},
		{
			name:     "Access Point ARN with additional elements",
			arn:      "arn:aws:s3:us-east-1:123456789:accesspoint/my-access-point/extra",
			expected: "us-east-1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			region := getRegionFromAccessPointARN(tc.arn)
			assert.Equal(t, tc.expected, region)
		})
	}
}

func TestGetBucketNameFromARN(t *testing.T) {
	testCases := []struct {
		name      string
		bucketARN string
		expected  string
	}{
		{
			name:      "Standard bucket ARN",
			bucketARN: "arn:aws:s3:::my_corporate_bucket",
			expected:  "my_corporate_bucket",
		},
		{
			name:      "Access Point ARN",
			bucketARN: "arn:aws:s3:us-east-1:123456789:accesspoint/my-access-point",
			expected:  "arn:aws:s3:us-east-1:123456789:accesspoint/my-access-point",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bucketName := getBucketNameFromARN(tc.bucketARN)
			assert.Equal(t, tc.expected, bucketName)
		})
	}
}

func TestGetRegionForBucketARN(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and must be executed manually.
	tfConfig := getTerraformOutputs(t, false)

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	s3Client := s3.NewFromConfig(cfg)

	regionName, err := getRegionForBucket(context.Background(), s3Client, getBucketNameFromARN(tfConfig.BucketName))
	assert.NoError(t, err)
	assert.Equal(t, tfConfig.AWSRegion, regionName)
}

func TestPaginatorListPrefix(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and must be executed manually.
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
		"testdata/log.txt", // Skipped (no match).
	)

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	cfg.Region = tfConfig.AWSRegion
	if err != nil {
		t.Fatal(err)
	}

	s3API := &awsS3API{
		client: s3Client,
	}

	var objects []string
	paginator := s3API.ListObjectsPaginator(tfConfig.BucketName, "log")
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		assert.NoError(t, err)
		for _, object := range page.Contents {
			objects = append(objects, *object.Key)
		}
	}

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
	tfConfig := getTerraformOutputs(t, false)
	awsCfg := makeAWSConfig(t, tfConfig.AWSRegion)

	// Ensure SQS is empty before testing.
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
		"testdata/log.txt", // Skipped (no match).
	)

	sqsInput := createSQSInput(t, makeTestConfigSQS(tfConfig.QueueURLForSNS))

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		return sqsInput.Run(inputCtx, &fakePipeline{})
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 8, sqsInput.metrics.sqsMessagesReceivedTotal.Get()) // S3 could batch notifications.
	assert.EqualValues(t, 0, sqsInput.metrics.sqsMessagesInflight.Get())
	assert.EqualValues(t, 7, sqsInput.metrics.sqsMessagesDeletedTotal.Get())
	assert.EqualValues(t, 1, sqsInput.metrics.sqsMessagesReturnedTotal.Get()) // Invalid JSON is returned so that it can eventually be DLQed.
	assert.EqualValues(t, 0, sqsInput.metrics.sqsVisibilityTimeoutExtensionsTotal.Get())
	assert.EqualValues(t, 0, sqsInput.metrics.s3ObjectsInflight.Get())
	assert.EqualValues(t, 7, sqsInput.metrics.s3ObjectsRequestedTotal.Get())
	assert.EqualValues(t, 12, sqsInput.metrics.s3EventsCreatedTotal.Get())
	assert.Greater(t, sqsInput.metrics.sqsLagTime.Mean(), 0.0)
	assert.EqualValues(t, 0.0, sqsInput.metrics.sqsWorkerUtilization.Get()) // Workers are reset after processing and hence utilization should be 0 at the end
}

func TestInputRunEventbridgeSQS(t *testing.T) {
	logp.TestingSetup()

	// Terraform is used to set up S3 and SQS and must be executed manually.
	tfConfig := getTerraformOutputs(t, false)
	awsCfg := makeAWSConfig(t, tfConfig.AWSRegion)

	// Ensure SQS is empty before testing.
	drainSQS(t, tfConfig.AWSRegion, tfConfig.BucketNameForEB, awsCfg)

	s3Client := s3.NewFromConfig(awsCfg)
	uploadS3TestFiles(t, tfConfig.AWSRegion, tfConfig.BucketNameForEB, s3Client,
		"testdata/events-array.json",
		"testdata/invalid.json",
		"testdata/log.json",
		"testdata/log.ndjson",
		"testdata/multiline.json",
		"testdata/multiline.json.gz",
		"testdata/multiline.txt",
		"testdata/log.txt", // Skipped (no match).
	)

	sqsInput := createSQSInput(t, makeTestConfigSQS(tfConfig.QueueURLForEB))

	inputCtx, cancel := newV2Context()
	t.Cleanup(cancel)
	time.AfterFunc(15*time.Second, func() {
		cancel()
	})

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		return sqsInput.Run(inputCtx, &fakePipeline{})
	})

	if err := errGroup.Wait(); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 8, sqsInput.metrics.sqsMessagesReceivedTotal.Get()) // S3 could batch notifications.
	assert.EqualValues(t, 0, sqsInput.metrics.sqsMessagesInflight.Get())
	assert.EqualValues(t, 7, sqsInput.metrics.sqsMessagesDeletedTotal.Get())
	assert.EqualValues(t, 1, sqsInput.metrics.sqsMessagesReturnedTotal.Get()) // Invalid JSON is returned so that it can eventually be DLQed.
	assert.EqualValues(t, 0, sqsInput.metrics.sqsVisibilityTimeoutExtensionsTotal.Get())
	assert.EqualValues(t, 0, sqsInput.metrics.s3ObjectsInflight.Get())
	assert.EqualValues(t, 7, sqsInput.metrics.s3ObjectsRequestedTotal.Get())
	assert.EqualValues(t, 12, sqsInput.metrics.s3EventsCreatedTotal.Get())
	assert.Greater(t, sqsInput.metrics.sqsLagTime.Mean(), 0.0)
	assert.EqualValues(t, 0.0, sqsInput.metrics.sqsWorkerUtilization.Get()) // Workers are reset after processing and hence utilization should be 0 at the end
}
