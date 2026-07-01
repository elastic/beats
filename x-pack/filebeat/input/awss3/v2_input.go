// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v7/x-pack/libbeat/statusreporterhelper"
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

func (in *inputV2) Run(inputCtx v2.Context, pipeline beat.Pipeline) error {
	st := statusreporterhelper.New(inputCtx, inputCtx.Logger, "aws-s3 V2")
	st.UpdateStatus(status.Starting, "Input starting")

	log := inputCtx.Logger.With("queue_url", in.config.QueueURL, "bucket_arn", in.config.getBucketARN())
	log.Info("aws-s3 V2 input starting")

	awsCfg, err := awscommon.InitializeAWSConfig(in.config.AWSConfig, log)
	if err != nil {
		st.UpdateStatus(status.Failed, fmt.Sprintf("AWS config init failed: %s", err))
		return fmt.Errorf("initializing AWS config: %w", err)
	}

	ctx := v2.GoContextFromCanceler(inputCtx.Cancelation)

	if in.config.QueueURL != "" {
		err = in.runSQS(ctx, log, st, awsCfg, pipeline, inputCtx)
	} else {
		err = in.runPolling(ctx, log, st, awsCfg, pipeline, inputCtx)
	}

	log.Info("aws-s3 V2 input stopping")
	if err == nil {
		st.UpdateStatus(status.Stopped, "Input execution ended")
	}
	return err
}

func (in *inputV2) runSQS(ctx context.Context, log *logp.Logger, st status.StatusReporter, awsCfg awssdk.Config, pipeline beat.Pipeline, inputCtx v2.Context) error {
	region := in.resolveSQSRegion(awsCfg)
	if region == "" {
		st.UpdateStatus(status.Failed, "Cannot determine SQS region")
		return fmt.Errorf("region not specified and failed to get AWS region from queue_url: %w", errBadQueueURL)
	}
	awsCfg.Region = region

	sqsAPI := &awsSQSAPI{
		client:            sqs.NewFromConfig(awsCfg, in.config.sqsConfigModifier),
		queueURL:          in.config.QueueURL,
		apiTimeout:        in.config.APITimeout,
		visibilityTimeout: in.config.VisibilityTimeout,
		longPollWaitTime:  in.config.SQSWaitTime,
	}
	s3API := newAWSs3API(s3.NewFromConfig(awsCfg, in.config.s3ConfigModifier), log)

	// Validate the pipeline (including processors) before starting the
	// receive loop so config errors are reported without needing an SQS
	// round-trip. On failure, block until cancelled so the DEGRADED status
	// remains visible to the management framework (which overwrites status
	// when Run returns).
	probeACK := newAWSACKHandler()
	probeClient, err := createPipelineClient(pipeline, probeACK)
	if err != nil {
		st.UpdateStatus(status.Degraded, fmt.Sprintf("Pipeline connection failed: %s", err))
		<-ctx.Done()
		return nil
	}
	probeACK.Close()
	probeClient.Close()

	metrics := newInputMetrics(inputCtx.MetricsRegistry, in.config.NumberOfWorkers, logp.NewNopLogger())
	defer metrics.Close()

	processor := newObjectProcessorV2(s3API, metrics, in.config.getFileSelectors(), in.config.BackupConfig)

	script, err := newScriptFromConfig(log.Named("sqs_script"), in.config.SQSScript, in.path)
	if err != nil {
		st.UpdateStatus(status.Failed, fmt.Sprintf("Script init failed: %s", err))
		return fmt.Errorf("failed to initialize SQS script: %w", err)
	}

	disc := newSQSDiscoveryV2(sqsDiscoveryV2Config{
		SQS:               sqsAPI,
		S3Move:            s3API,
		QueueURL:          in.config.QueueURL,
		VisibilityTimeout: in.config.VisibilityTimeout,
		MaxReceiveCount:   in.config.SQSMaxReceiveCount,
		Script:            script,
		Processor:         processor,
		Metrics:           metrics,
		Log:               log.Named("sqs"),
		Status:            st,
	})

	cc := newConcurrencyController(concurrencyControllerConfig{
		MaxWorkers:     in.config.NumberOfWorkers,
		AdjustCooldown: 5 * time.Second,
		Log:            log.Named("flow"),
		Registry:       inputCtx.MetricsRegistry,
	})

	// Start queue depth monitor.
	go messageCountMonitor{sqs: sqsAPI, metrics: metrics}.run(ctx)

	st.UpdateStatus(status.Running, "Input is running")

	// Worker pool bounded by number_of_workers. The adaptive controller
	// observes per-publish latency and records effective concurrency; actual
	// throttling happens at the publish layer via publishWithBackpressure.
	sem := make(chan struct{}, in.config.NumberOfWorkers)
	var wg sync.WaitGroup

	disc.ReceiveLoop(ctx, in.config.NumberOfWorkers, func(msgCtx context.Context, msg types.Message) {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			in.processSQSMessage(msgCtx, disc, cc, msg, pipeline, metrics)
		}()
	})

	wg.Wait()
	return nil
}

func (in *inputV2) processSQSMessage(ctx context.Context, disc *sqsDiscoveryV2, cc *concurrencyController, msg types.Message, pipeline beat.Pipeline, metrics *inputMetrics) {
	id := metrics.beginSQSWorker()
	defer metrics.endSQSWorker(id)

	acks := newAWSACKHandler()
	client, err := createPipelineClient(pipeline, acks)
	if err != nil {
		disc.log.Errorf("failed to create pipeline client: %v", err)
		return
	}
	defer func() {
		acks.Close()
		client.Close()
	}()

	publishCount := 0
	result := disc.ProcessMessage(ctx, &msg, func(e beat.Event) {
		publishWithBackpressure(cc, 50*time.Millisecond, func() {
			client.Publish(e)
		})
		publishCount++
	})

	if publishCount == 0 {
		result.Done()
	} else {
		acks.Add(publishCount, result.Done)
	}
}

func (in *inputV2) runPolling(ctx context.Context, log *logp.Logger, st status.StatusReporter, awsCfg awssdk.Config, pipeline beat.Pipeline, inputCtx v2.Context) error {
	// Validate the pipeline (including processors) early. Block on failure
	// so the DEGRADED status stays visible (the compat runner overwrites
	// status when Run returns).
	probeACK := newAWSACKHandler()
	probeClient, err := createPipelineClient(pipeline, probeACK)
	if err != nil {
		st.UpdateStatus(status.Degraded, fmt.Sprintf("Pipeline connection failed: %s", err))
		<-ctx.Done()
		return nil
	}
	probeACK.Close()
	probeClient.Close()

	if in.config.RegionName != "" {
		awsCfg.Region = in.config.RegionName
	}

	s3Client := s3.NewFromConfig(awsCfg, in.config.s3ConfigModifier)

	// Detect bucket region for AWS buckets.
	if in.config.NonAWSBucketName == "" {
		regionName, err := getRegionForBucket(ctx, s3Client, in.config.getBucketName())
		if err != nil {
			st.UpdateStatus(status.Failed, fmt.Sprintf("Failed to get bucket region: %s", err))
			return fmt.Errorf("failed to get AWS region for bucket: %w", err)
		}
		if regionName != awsCfg.Region {
			awsCfg.Region = regionName
			s3Client = s3.NewFromConfig(awsCfg, in.config.s3ConfigModifier)
		}
	}

	s3API := newAWSs3API(s3Client, log)

	metrics := newInputMetrics(inputCtx.MetricsRegistry, in.config.NumberOfWorkers, logp.NewNopLogger())
	defer metrics.Close()

	processor := newObjectProcessorV2(s3API, metrics, in.config.getFileSelectors(), in.config.BackupConfig)

	capacity := 0
	if in.config.LexicographicalOrdering {
		capacity = in.config.LexicographicalLookbackKeys
	}
	registry, err := newStateRegistryV2(stateRegistryV2Config{
		Log:       log,
		Store:     in.store,
		KeyPrefix: in.config.BucketListPrefix,
		Capacity:  capacity,
	})
	if err != nil {
		st.UpdateStatus(status.Failed, fmt.Sprintf("State registry init failed: %s", err))
		return fmt.Errorf("creating state registry: %w", err)
	}
	defer registry.Close()

	poller := newPollingDiscoveryV2(pollingDiscoveryV2Config{
		S3:              s3API,
		Processor:       processor,
		Registry:        registry,
		Metrics:         metrics,
		Log:             log.Named("s3"),
		Status:          st,
		BucketARN:       in.config.getBucketARN(),
		ListPrefix:      in.config.BucketListPrefix,
		ListInterval:    in.config.BucketListInterval,
		NumWorkers:      in.config.NumberOfWorkers,
		Region:          awsCfg.Region,
		Provider:        in.config.ProviderOverride,
		Lexicographical: in.config.LexicographicalOrdering,
		FilterProvider:  newFilterProvider(&in.config),
	})

	st.UpdateStatus(status.Running, "Input is running")
	poller.Run(ctx, pipeline)
	return nil
}

func (in *inputV2) resolveSQSRegion(awsCfg awssdk.Config) string {
	if in.config.RegionName != "" {
		return in.config.RegionName
	}
	if r := getRegionFromQueueURL(in.config.QueueURL); r != "" {
		return r
	}
	if awsCfg.Region != "" {
		return awsCfg.Region
	}
	if in.config.AWSConfig.DefaultRegion != "" {
		return in.config.AWSConfig.DefaultRegion
	}
	return ""
}
