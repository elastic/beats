// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"time"

	"github.com/urso/sderr"

	"github.com/elastic/beats/v7/libbeat/statestore"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName        = "aws-s3"
	awsS3StatePrefix = "filebeat::aws-s3::"
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
	s3Input              *s3Input
	store                beater.StateStore
	persistentStore      *statestore.Store
	states               *States
	grp                  unison.Group
	storeCleanupInterval time.Duration
}

// s3Input is a input for s3
type s3Input struct {
	config               config
	store                *statestore.Store
	states               *States
	grp                  unison.Group
	storeCleanupInterval time.Duration
}

func (im *s3InputManager) Init(grp unison.Group, mode v2.Mode) error {
	if mode != v2.ModeRun {
		return nil
	}

	ok := false
	persistentStore, err := im.store.Access()
	if err != nil {
		return sderr.Wrap(err, "Can not access persistent store")
	}

	defer cleanup.IfNot(&ok, func() { persistentStore.Close() })

	states := NewStates()
	err = states.readStatesFrom(persistentStore)
	if err != nil {
		return sderr.Wrap(err, "Can not start persistent store")
	}

	// We should close the persistent store when filebeat stops:
	// we have to manage it at Init time because the input could not run
	err = grp.Go(func(canceler unison.Canceler) error {
	cancelerLoop:
		for {
			select {
			case <-canceler.Done():
				persistentStore.Close()
				break cancelerLoop
			default:
				if canceler.Err() != nil {
					persistentStore.Close()
					break cancelerLoop
				}
			}
		}

		return nil
	})

	im.grp = grp
	im.persistentStore = persistentStore
	im.states = states
	im.storeCleanupInterval = im.store.CleanupInterval()
	ok = true

	return nil
}

func (im *s3InputManager) Create(cfg *common.Config) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	im.s3Input = newInput(config, im.persistentStore, im.states, im.grp, im.storeCleanupInterval)

	return im.s3Input, nil
}

func newInput(config config, store *statestore.Store, states *States, grp unison.Group, storeCleanupInterval time.Duration) *s3Input {
	return &s3Input{
		config:               config,
		store:                store,
		states:               states,
		grp:                  grp,
		storeCleanupInterval: storeCleanupInterval,
	}
}

func (in *s3Input) Name() string { return inputName }

func (in *s3Input) Test(ctx v2.TestContext) error {
	_, err := awscommon.GetAWSCredentials(in.config.AWSConfig)
	if err != nil {
		return fmt.Errorf("getAWSCredentials failed: %w", err)
	}
	return nil
}

func (in *s3Input) Run(ctx v2.Context, pipeline beat.Pipeline) error {
	var err error
	var collector s3Collector

	metricRegistry := monitoring.GetNamespace("dataset").GetRegistry()
	inputMetrics := newInputMetrics(metricRegistry, ctx.ID)

	if in.config.QueueURL != "" {
		collector, err = in.createSQSCollector(ctx, pipeline, inputMetrics)
		if err != nil {
			return fmt.Errorf("cannot create SQS collector: %w", err)
		}
	}

	if in.config.S3Bucket != "" {
		collector, err = in.createS3BucketCollector(ctx, pipeline, inputMetrics, in.states, in.store)
		if err != nil {
			return fmt.Errorf("cannot create S3 bucket collector: %w", err)
		}
	}

	err = in.grp.Go(func(canceler unison.Canceler) error {
		interval := in.storeCleanupInterval
		if interval <= 0 {
			interval = 5 * time.Minute
		}
		cleanStore(canceler, ctx.Logger, in.store, in.states, interval, in.config.S3BucketObjectExpiration)
		return nil
	})

	if err != nil {
		return sderr.Wrap(err, "Can not start cleanup process")
	}

	defer collector.getMetrics().Close()
	defer collector.getPublisher().Close()
	collector.run()

	if ctx.Cancelation.Err() == context.Canceled {
		return nil
	} else {
		return ctx.Cancelation.Err()
	}
}

func (in *s3Input) createS3BucketCollector(ctx v2.Context, pipeline beat.Pipeline, metrics *inputMetrics, states *States, store *statestore.Store) (*s3BucketCollector, error) {
	storedOp := newStoredOp(in.states, in.store)
	publisher, err := pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   ctx.Cancelation,
		ACKHandler: newACKHandler(storedOp),
	})

	if err != nil {
		return nil, err
	}

	log := ctx.Logger.With("s3_bucket", in.config.S3Bucket)

	awsConfig, err := awscommon.GetAWSCredentials(in.config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("getAWSCredentials failed: %w", err)
	}
	s3Servicename := "s3"
	if in.config.FIPSEnabled {
		s3Servicename = "s3-fips"
	}

	log.Debug("s3 service name = ", s3Servicename)
	log.Debug("s3 input config max_number_of_messages = ", in.config.MaxNumberOfMessages)
	log.Debug("s3 input config endpoint = ", in.config.AWSConfig.Endpoint)

	return &s3BucketCollector{
		cancellation: ctxtool.FromCanceller(ctx.Cancelation),
		logger:       log,
		config:       &in.config,
		publisher:    publisher,
		s3:           s3.New(awscommon.EnrichAWSConfigWithEndpoint(in.config.AWSConfig.Endpoint, s3Servicename, awsConfig.Region, awsConfig)),
		metrics:      metrics,
		states:       states,
		store:        store,
	}, nil
}

func (in *s3Input) createSQSCollector(ctx v2.Context, pipeline beat.Pipeline, metrics *inputMetrics) (*s3SQSCollector, error) {
	publisher, err := pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   ctx.Cancelation,
		ACKHandler: newACKHandler(nil),
	})

	if err != nil {
		return nil, err
	}

	log := ctx.Logger.With("queue_url", in.config.QueueURL)

	regionName, err := getRegionFromQueueURL(in.config.QueueURL, in.config.AWSConfig.Endpoint)
	if err != nil {
		err := fmt.Errorf("getRegionFromQueueURL failed: %w", err)
		log.Error(err)
		return nil, err
	} else {
		log = log.With("region", regionName)
	}

	awsConfig, err := awscommon.GetAWSCredentials(in.config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("getAWSCredentials failed: %w", err)
	}
	awsConfig.Region = regionName

	visibilityTimeout := int64(in.config.VisibilityTimeout.Seconds())
	log.Infof("visibility timeout is set to %v seconds", visibilityTimeout)
	log.Infof("aws api timeout is set to %v", in.config.APITimeout)

	s3Servicename := "s3"
	if in.config.FIPSEnabled {
		s3Servicename = "s3-fips"
	}

	log.Debug("s3 service name = ", s3Servicename)
	log.Debug("s3 input config max_number_of_messages = ", in.config.MaxNumberOfMessages)
	log.Debug("s3 input config endpoint = ", in.config.AWSConfig.Endpoint)
	return &s3SQSCollector{
		cancellation:      ctxtool.FromCanceller(ctx.Cancelation),
		logger:            log,
		config:            &in.config,
		publisher:         publisher,
		visibilityTimeout: visibilityTimeout,
		sqs:               sqs.New(awscommon.EnrichAWSConfigWithEndpoint(in.config.AWSConfig.Endpoint, "sqs", regionName, awsConfig)),
		s3:                s3.New(awscommon.EnrichAWSConfigWithEndpoint(in.config.AWSConfig.Endpoint, s3Servicename, regionName, awsConfig)),
		metrics:           metrics,
	}, nil
}

func newACKHandler(storedOp *storedOp) beat.ACKer {
	return acker.ConnectionOnly(
		acker.EventPrivateReporter(func(_ int, privates []interface{}) {
			for _, private := range privates {
				if private, ok := private.([]interface{}); ok {
					for _, currentPrivate := range private {
						if s3Context, ok := currentPrivate.(*s3Context); ok {
							s3Context.done()
						}
						if info, ok := currentPrivate.(s3Info); ok && storedOp != nil {
							storedOp.execute(info)
						}
					}
				}
			}
		}),
	)
}
