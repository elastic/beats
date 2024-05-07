// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName                = "aws-s3"
	sqsAccessDeniedErrorCode = "AccessDeniedException"
)

func Plugin(store beater.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from s3",
		Manager:    &s3InputManager{store: store},
	}
}

type s3InputManager struct {
	store beater.StateStore
}

func (im *s3InputManager) Init(grp unison.Group) error {
	return nil
}

func (im *s3InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config, im.store)
}

// s3Input is a input for reading logs from S3 when triggered by an SQS message.
type s3Input struct {
	config    config
	awsConfig awssdk.Config
	store     beater.StateStore
	metrics   *inputMetrics
}

func newInput(config config, store beater.StateStore) (*s3Input, error) {
	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)

	if config.AWSConfig.Endpoint != "" {
		// Add a custom endpointResolver to the awsConfig so that all the requests are routed to this endpoint
		awsConfig.EndpointResolverWithOptions = awssdk.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (awssdk.Endpoint, error) {
			return awssdk.Endpoint{
				PartitionID:   "aws",
				URL:           config.AWSConfig.Endpoint,
				SigningRegion: awsConfig.Region,
			}, nil
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS credentials: %w", err)
	}

	return &s3Input{
		config:    config,
		awsConfig: awsConfig,
		store:     store,
	}, nil
}

func (in *s3Input) Name() string { return inputName }

func (in *s3Input) Test(ctx v2.TestContext) error {
	return nil
}

func (in *s3Input) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)

	if in.config.QueueURL != "" {
		return in.runQueueReader(ctx, inputContext, pipeline)
	}

	if in.config.BucketARN != "" || in.config.NonAWSBucketName != "" {
		return in.runS3Poller(ctx, inputContext, pipeline)
	}

	return nil
}

func (in *s3Input) runQueueReader(
	ctx context.Context,
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	// Set awsConfig.Region based on the config and queue URL
	region, err := chooseRegion(inputContext.Logger, in.config)
	if err != nil {
		return err
	}
	in.awsConfig.Region = region

	// Create SQS receiver and S3 notification processor.
	receiver, err := in.createSQSReceiver(inputContext, pipeline)
	if err != nil {
		return fmt.Errorf("failed to initialize sqs receiver: %w", err)
	}
	defer receiver.metrics.Close()

	// Poll metrics periodically in the background
	go pollSqsWaitingMetric(ctx, receiver)

	return receiver.Receive(ctx)
}

func (in *s3Input) runS3Poller(
	ctx context.Context,
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	// Create client for publishing events and receive notification of their ACKs.
	client, err := pipeline.ConnectWith(beat.ClientConfig{
		EventListener: awscommon.NewEventACKHandler(),
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: boolPtr(false),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	// Connect to the registry and create our states lookup
	persistentStore, err := in.store.Access()
	if err != nil {
		return fmt.Errorf("can not access persistent store: %w", err)
	}
	defer persistentStore.Close()

	states, err := newStates(inputContext, persistentStore)
	if err != nil {
		return fmt.Errorf("can not start persistent store: %w", err)
	}

	// Create S3 receiver and S3 notification processor.
	poller, err := in.createS3Poller(inputContext, ctx, client, states)
	if err != nil {
		return fmt.Errorf("failed to initialize s3 poller: %w", err)
	}
	defer poller.metrics.Close()

	return poller.Poll(ctx)
}

func (in *s3Input) createSQSReceiver(ctx v2.Context, pipeline beat.Pipeline) (*sqsReader, error) {
	sqsAPI := &awsSQSAPI{
		client: sqs.NewFromConfig(in.awsConfig, func(o *sqs.Options) {
			if in.config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		}),
		queueURL:          in.config.QueueURL,
		apiTimeout:        in.config.APITimeout,
		visibilityTimeout: in.config.VisibilityTimeout,
		longPollWaitTime:  in.config.SQSWaitTime,
	}

	s3API := &awsS3API{
		client: s3.NewFromConfig(in.awsConfig, func(o *s3.Options) {
			if in.config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
			o.UsePathStyle = in.config.PathStyle
		}),
	}

	log := ctx.Logger.With("queue_url", in.config.QueueURL)
	log.Infof("AWS api_timeout is set to %v.", in.config.APITimeout)
	log.Infof("AWS region is set to %v.", in.awsConfig.Region)
	log.Infof("AWS SQS visibility_timeout is set to %v.", in.config.VisibilityTimeout)
	log.Infof("AWS SQS max_number_of_messages is set to %v.", in.config.MaxNumberOfMessages)

	if in.config.BackupConfig.GetBucketName() != "" {
		log.Warnf("You have the backup_to_bucket functionality activated with SQS. Please make sure to set appropriate destination buckets" +
			"or prefixes to avoid an infinite loop.")
	}

	fileSelectors := in.config.FileSelectors
	if len(in.config.FileSelectors) == 0 {
		fileSelectors = []fileSelectorConfig{{ReaderConfig: in.config.ReaderConfig}}
	}
	script, err := newScriptFromConfig(log.Named("sqs_script"), in.config.SQSScript)
	if err != nil {
		return nil, err
	}
	in.metrics = newInputMetrics(ctx.ID, nil, in.config.MaxNumberOfMessages)

	s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), in.metrics, s3API, fileSelectors, in.config.BackupConfig, in.config.MaxNumberOfMessages)

	sqsMessageHandler := newSQSS3EventProcessor(log.Named("sqs_s3_event"), in.metrics, sqsAPI, script, in.config.VisibilityTimeout, in.config.SQSMaxReceiveCount, pipeline, s3EventHandlerFactory, in.config.MaxNumberOfMessages)

	sqsReader := newSQSReader(log.Named("sqs"), in.metrics, sqsAPI, in.config.MaxNumberOfMessages, sqsMessageHandler)

	return sqsReader, nil
}

type nonAWSBucketResolver struct {
	endpoint string
}

func (n nonAWSBucketResolver) ResolveEndpoint(region string, options s3.EndpointResolverOptions) (awssdk.Endpoint, error) {
	return awssdk.Endpoint{URL: n.endpoint, SigningRegion: region, HostnameImmutable: true, Source: awssdk.EndpointSourceCustom}, nil
}

func (in *s3Input) createS3Poller(ctx v2.Context, cancelCtx context.Context, client beat.Client, states *states) (*s3Poller, error) {
	var bucketName string
	var bucketID string
	if in.config.NonAWSBucketName != "" {
		bucketName = in.config.NonAWSBucketName
		bucketID = bucketName
	} else if in.config.BucketARN != "" {
		bucketName = getBucketNameFromARN(in.config.BucketARN)
		bucketID = in.config.BucketARN
	}

	s3Client := s3.NewFromConfig(in.awsConfig, func(o *s3.Options) {
		if in.config.NonAWSBucketName != "" {
			o.EndpointResolver = nonAWSBucketResolver{endpoint: in.config.AWSConfig.Endpoint}
		}

		if in.config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
		o.UsePathStyle = in.config.PathStyle

		o.Retryer = retry.NewStandard(func(so *retry.StandardOptions) {
			so.MaxAttempts = 5
			// Recover quickly when requests start working again
			so.NoRetryIncrement = 100
		})
	})
	regionName, err := getRegionForBucket(cancelCtx, s3Client, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS region for bucket: %w", err)
	}

	originalAwsConfigRegion := in.awsConfig.Region

	in.awsConfig.Region = regionName

	if regionName != originalAwsConfigRegion {
		s3Client = s3.NewFromConfig(in.awsConfig, func(o *s3.Options) {
			if in.config.NonAWSBucketName != "" {
				o.EndpointResolver = nonAWSBucketResolver{endpoint: in.config.AWSConfig.Endpoint}
			}

			if in.config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
			o.UsePathStyle = in.config.PathStyle
		})
	}

	s3API := &awsS3API{
		client: s3Client,
	}

	log := ctx.Logger.With("bucket", bucketID)
	log.Infof("number_of_workers is set to %v.", in.config.NumberOfWorkers)
	log.Infof("bucket_list_interval is set to %v.", in.config.BucketListInterval)
	log.Infof("bucket_list_prefix is set to %v.", in.config.BucketListPrefix)
	log.Infof("AWS region is set to %v.", in.awsConfig.Region)

	fileSelectors := in.config.FileSelectors
	if len(in.config.FileSelectors) == 0 {
		fileSelectors = []fileSelectorConfig{{ReaderConfig: in.config.ReaderConfig}}
	}
	in.metrics = newInputMetrics(ctx.ID, nil, in.config.MaxNumberOfMessages)
	s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), in.metrics, s3API, fileSelectors, in.config.BackupConfig, in.config.MaxNumberOfMessages)
	s3Poller := newS3Poller(log.Named("s3_poller"),
		in.metrics,
		s3API,
		client,
		s3EventHandlerFactory,
		states,
		bucketID,
		in.config.BucketListPrefix,
		in.awsConfig.Region,
		getProviderFromDomain(in.config.AWSConfig.Endpoint, in.config.ProviderOverride),
		in.config.NumberOfWorkers,
		in.config.BucketListInterval)

	return s3Poller, nil
}

var errBadQueueURL = errors.New("QueueURL is not in format: https://sqs.{REGION_ENDPOINT}.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME} or https://{VPC_ENDPOINT}.sqs.{REGION_ENDPOINT}.vpce.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME}")

func chooseRegion(log *logp.Logger, config config) (string, error) {
	urlRegion := getRegionFromQueueURL(config.QueueURL, config.AWSConfig.Endpoint)
	if config.RegionName != "" {
		// If a region is configured, that takes precedence over the URL.
		if log != nil && config.RegionName != urlRegion {
			log.Warnf("configured region disagrees with queue_url region (%q != %q): using %q", config.RegionName, urlRegion, config.RegionName)
		}
		return config.RegionName, nil
	}
	if urlRegion != "" {
		// If no region is configured, fall back on the URL.
		return urlRegion, nil
	}
	// If we can't get the region from the config or the URL, report an error.
	return "", fmt.Errorf("failed to get AWS region from queue_url: %w", errBadQueueURL)
}

// getRegionFromQueueURL returns the region from standard queue URLs, or the
// empty string if it couldn't be determined.
func getRegionFromQueueURL(queueURL, endpoint string) string {
	// get region from queueURL
	// Example for sqs queue: https://sqs.us-east-1.amazonaws.com/12345678912/test-s3-logs
	// Example for vpce: https://vpce-test.sqs.us-east-1.vpce.amazonaws.com/12345678912/sqs-queue
	u, err := url.Parse(queueURL)
	if err != nil {
		return ""
	}

	// check for sqs queue url
	host := strings.SplitN(u.Host, ".", 3)
	if len(host) == 3 && host[0] == "sqs" {
		if host[2] == endpoint || (endpoint == "" && strings.HasPrefix(host[2], "amazonaws.")) {
			return host[1]
		}
	}

	// check for vpce url
	host = strings.SplitN(u.Host, ".", 5)
	if len(host) == 5 && host[1] == "sqs" {
		if host[4] == endpoint || (endpoint == "" && strings.HasPrefix(host[4], "amazonaws.")) {
			return host[2]
		}
	}

	return ""
}

func getRegionForBucket(ctx context.Context, s3Client *s3.Client, bucketName string) (string, error) {
	getBucketLocationOutput, err := s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: awssdk.String(bucketName),
	})

	if err != nil {
		return "", err
	}

	// Region us-east-1 have a LocationConstraint of null.
	if len(getBucketLocationOutput.LocationConstraint) == 0 {
		return "us-east-1", nil
	}

	return string(getBucketLocationOutput.LocationConstraint), nil
}

func getBucketNameFromARN(bucketARN string) string {
	bucketMetadata := strings.Split(bucketARN, ":")
	bucketName := bucketMetadata[len(bucketMetadata)-1]
	return bucketName
}

func getProviderFromDomain(endpoint string, ProviderOverride string) string {
	if ProviderOverride != "" {
		return ProviderOverride
	}
	if endpoint == "" {
		return "aws"
	}
	// List of popular S3 SaaS providers
	providers := map[string]string{
		"amazonaws.com":          "aws",
		"c2s.sgov.gov":           "aws",
		"c2s.ic.gov":             "aws",
		"amazonaws.com.cn":       "aws",
		"backblazeb2.com":        "backblaze",
		"cloudflarestorage.com":  "cloudflare",
		"wasabisys.com":          "wasabi",
		"digitaloceanspaces.com": "digitalocean",
		"dream.io":               "dreamhost",
		"scw.cloud":              "scaleway",
		"googleapis.com":         "gcp",
		"cloud.it":               "arubacloud",
		"linodeobjects.com":      "linode",
		"vultrobjects.com":       "vultr",
		"appdomain.cloud":        "ibm",
		"aliyuncs.com":           "alibaba",
		"oraclecloud.com":        "oracle",
		"exo.io":                 "exoscale",
		"upcloudobjects.com":     "upcloud",
		"ilandcloud.com":         "iland",
		"zadarazios.com":         "zadara",
	}

	parsedEndpoint, _ := url.Parse(endpoint)
	for key, provider := range providers {
		// support endpoint with and without scheme (http(s)://abc.xyz, abc.xyz)
		constraint := parsedEndpoint.Hostname()
		if len(parsedEndpoint.Scheme) == 0 {
			constraint = parsedEndpoint.Path
		}
		if strings.HasSuffix(constraint, key) {
			return provider
		}
	}
	return "unknown"
}

func pollSqsWaitingMetric(ctx context.Context, receiver *sqsReader) {
	// Run GetApproximateMessageCount before start of timer to set initial count for sqs waiting metric
	// This is to avoid misleading values in metric when sqs messages are processed before the ticker channel kicks in
	if shouldReturn := updateMessageCount(receiver, ctx); shouldReturn {
		return
	}

	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if shouldReturn := updateMessageCount(receiver, ctx); shouldReturn {
				return
			}
		}
	}
}

// updateMessageCount runs GetApproximateMessageCount for the given context and updates the receiver metric with the count returning false on no error
// If there is an error, the metric is reinitialized to -1 and true is returned
func updateMessageCount(receiver *sqsReader, ctx context.Context) bool {
	count, err := receiver.GetApproximateMessageCount(ctx)

	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case sqsAccessDeniedErrorCode:
			// stop polling if auth error is encountered
			// Set it back to -1 because there is a permission error
			receiver.metrics.sqsMessagesWaiting.Set(int64(-1))
			return true
		}
	}

	receiver.metrics.sqsMessagesWaiting.Set(int64(count))
	return false
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
