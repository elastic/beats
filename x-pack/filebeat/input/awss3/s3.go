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
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

// var instead of const so it can be reduced during unit tests (instead of waiting
// through 10 minutes of retry backoff)
var readerLoopMaxCircuitBreaker = 10

type s3ObjectPayload struct {
	s3ObjectHandler s3ObjectHandler
	objectState     state
}

type s3PollerInput struct {
	config    config
	awsConfig awssdk.Config
	store     beater.StateStore
}

type s3Poller struct {
	numberOfWorkers      int
	bucket               string
	listPrefix           string
	region               string
	provider             string
	bucketPollInterval   time.Duration
	s3                   s3API
	log                  *logp.Logger
	metrics              *inputMetrics
	client               beat.Client
	s3ObjectHandler      s3ObjectHandlerFactory
	states               *states
	workersProcessingMap *sync.Map
}

func (in *s3PollerInput) Name() string { return inputName }

func (in *s3PollerInput) Test(ctx v2.TestContext) error {
	return nil
}

func newS3PollerInput(
	config config,
	awsConfig awssdk.Config,
	store beater.StateStore,
) (v2.Input, error) {
	return &s3PollerInput{
		config:    config,
		awsConfig: awsConfig,
		store:     store,
	}, nil
}

func (in *s3PollerInput) Run(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)

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

	states, err := newStates(inputContext.Logger, persistentStore)
	if err != nil {
		return fmt.Errorf("can not start persistent store: %w", err)
	}

	// Create S3 receiver and S3 notification processor.
	poller, err := in.createS3Poller(inputContext, ctx, client, states)
	if err != nil {
		return fmt.Errorf("failed to initialize s3 poller: %w", err)
	}
	defer poller.metrics.Close()

	poller.Poll(ctx)
	return nil
}

func (in *s3PollerInput) createS3Poller(ctx v2.Context, cancelCtx context.Context, client beat.Client, states *states) (*s3Poller, error) {
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
	metrics := newInputMetrics(ctx.ID, nil, in.config.MaxNumberOfMessages)
	s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), metrics, s3API, fileSelectors, in.config.BackupConfig)
	s3Poller := newS3Poller(log.Named("s3_poller"),
		metrics,
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

func newS3Poller(log *logp.Logger,
	metrics *inputMetrics,
	s3 s3API,
	client beat.Client,
	s3ObjectHandler s3ObjectHandlerFactory,
	states *states,
	bucket string,
	listPrefix string,
	awsRegion string,
	provider string,
	numberOfWorkers int,
	bucketPollInterval time.Duration,
) *s3Poller {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	return &s3Poller{
		numberOfWorkers:      numberOfWorkers,
		bucket:               bucket,
		listPrefix:           listPrefix,
		region:               awsRegion,
		provider:             provider,
		bucketPollInterval:   bucketPollInterval,
		s3:                   s3,
		log:                  log,
		metrics:              metrics,
		client:               client,
		s3ObjectHandler:      s3ObjectHandler,
		states:               states,
		workersProcessingMap: new(sync.Map),
	}
}

func (p *s3Poller) createS3ObjectProcessor(ctx context.Context, state state) s3ObjectHandler {
	event := s3EventV2{}
	event.AWSRegion = p.region
	event.Provider = p.provider
	event.S3.Bucket.Name = state.Bucket
	event.S3.Bucket.ARN = p.bucket
	event.S3.Object.Key = state.Key

	acker := awscommon.NewEventACKTracker(ctx)

	return p.s3ObjectHandler.Create(ctx, p.log, p.client, acker, event)
}

func (p *s3Poller) workerLoop(ctx context.Context, s3ObjectPayloadChan <-chan *s3ObjectPayload) {
	rateLimitWaiter := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)

	for s3ObjectPayload := range s3ObjectPayloadChan {
		objHandler := s3ObjectPayload.s3ObjectHandler
		state := s3ObjectPayload.objectState

		// Process S3 object (download, parse, create events).
		err := objHandler.ProcessS3Object()
		if errors.Is(err, errS3DownloadFailed) {
			// Download errors are ephemeral. Add a backoff delay, then skip to the
			// next iteration so we don't mark the object as permanently failed.
			rateLimitWaiter.Wait()
			continue
		}
		// Reset the rate limit delay on results that aren't download errors.
		rateLimitWaiter.Reset()

		// Wait for downloaded objects to be ACKed.
		objHandler.Wait()

		if err != nil {
			p.log.Errorf("failed processing S3 event for object key %q in bucket %q: %v",
				state.Key, state.Bucket, err.Error())

			// Non-retryable error.
			state.Failed = true
		} else {
			state.Stored = true
		}

		// Persist the result
		p.states.AddState(state)

		// Metrics
		p.metrics.s3ObjectsAckedTotal.Inc()
	}
}

func (p *s3Poller) readerLoop(ctx context.Context, s3ObjectPayloadChan chan<- *s3ObjectPayload) {
	defer close(s3ObjectPayloadChan)

	bucketName := getBucketNameFromARN(p.bucket)

	errorBackoff := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)
	circuitBreaker := 0
	paginator := p.s3.ListObjectsPaginator(bucketName, p.listPrefix)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)

		if err != nil {
			p.log.Warnw("Error when paginating listing.", "error", err)
			// QuotaExceededError is client-side rate limiting in the AWS sdk,
			// don't include it in the circuit breaker count
			if !errors.As(err, &ratelimit.QuotaExceededError{}) {
				circuitBreaker++
				if circuitBreaker >= readerLoopMaxCircuitBreaker {
					p.log.Warnw(fmt.Sprintf("%d consecutive error when paginating listing, breaking the circuit.", circuitBreaker), "error", err)
					break
				}
			}
			// add a backoff delay and try again
			errorBackoff.Wait()
			continue
		}
		// Reset the circuit breaker and the error backoff if a read is successful
		circuitBreaker = 0
		errorBackoff.Reset()

		totListedObjects := len(page.Contents)

		// Metrics
		p.metrics.s3ObjectsListedTotal.Add(uint64(totListedObjects))
		for _, object := range page.Contents {
			state := newState(bucketName, *object.Key, *object.ETag, *object.LastModified)
			if p.states.IsProcessed(state) {
				p.log.Debugw("skipping state.", "state", state)
				continue
			}

			s3Processor := p.createS3ObjectProcessor(ctx, state)
			if s3Processor == nil {
				p.log.Debugw("empty s3 processor.", "state", state)
				continue
			}

			s3ObjectPayloadChan <- &s3ObjectPayload{
				s3ObjectHandler: s3Processor,
				objectState:     state,
			}

			p.metrics.s3ObjectsProcessedTotal.Inc()
		}
	}
}

func (p *s3Poller) Poll(ctx context.Context) {
	for ctx.Err() == nil {
		var workerWg sync.WaitGroup
		workChan := make(chan *s3ObjectPayload)

		// Start the worker goroutines to listen on the work channel
		for i := 0; i < p.numberOfWorkers; i++ {
			workerWg.Add(1)
			go func() {
				defer workerWg.Done()
				p.workerLoop(ctx, workChan)
			}()
		}

		// Start reading data and wait for its processing to be done
		p.readerLoop(ctx, workChan)
		workerWg.Wait()

		_ = timed.Wait(ctx, p.bucketPollInterval)
	}
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

type nonAWSBucketResolver struct {
	endpoint string
}

func (n nonAWSBucketResolver) ResolveEndpoint(region string, options s3.EndpointResolverOptions) (awssdk.Endpoint, error) {
	return awssdk.Endpoint{URL: n.endpoint, SigningRegion: region, HostnameImmutable: true, Source: awssdk.EndpointSourceCustom}, nil
}
