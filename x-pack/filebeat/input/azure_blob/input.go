// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure_blob

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/statestore"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/unison"
)

const inputName = "azure-blob"

func Plugin(store beater.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from azure blob storage",
		Manager:    &blobInputManager{store: store},
	}
}

type blobInputManager struct {
	store beater.StateStore
}

func (im *blobInputManager) Init(grp unison.Group, mode v2.Mode) error {
	return nil
}

func (im *blobInputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config, im.store)
}

// s3Input is a input for reading logs from S3 when triggered by an SQS message.
type blobInput struct {
	config     config
	credential *azblob.SharedKeyCredential
	store      beater.StateStore
}

func newInput(config config, store beater.StateStore) (*blobInput, error) {
	credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Azure Blob credentials: %w", err)
	}

	return &blobInput{
		config:     config,
		credential: credential,
		store:      store,
	}, nil
}

func (in *blobInput) Name() string { return inputName }

func (in *blobInput) Test(ctx v2.TestContext) error {
	return nil
}

func (in *blobInput) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	var err error

	persistentStore, err := in.store.Access()
	if err != nil {
		return fmt.Errorf("can not access persistent store: %w", err)
	}

	defer persistentStore.Close()

	states := newStates(inputContext)
	err = states.readStatesFrom(persistentStore)
	if err != nil {
		return fmt.Errorf("can not start persistent store: %w", err)
	}

	// Wrap input Context's cancellation Done channel a context.Context. This
	// goroutine stops with the parent closes the Done channel.
	ctx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Cancelation.Done():
		case <-ctx.Done():
		}
	}()
	defer cancelInputCtx()

	// Create client for publishing events and receive notification of their ACKs.
	client, err := pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   inputContext.Cancelation,
		ACKHandler: awscommon.NewEventACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	// Create Blob receiver and Blob notification processor.
	poller, err := in.createBlobLister(inputContext, ctx, client, persistentStore, states)
	if err != nil {
		return fmt.Errorf("failed to initialize Blob poller: %w", err)
	}
	defer poller.metrics.Close()

	if err := poller.Poll(ctx); err != nil {
		return err
	}

	return nil
}

func (in *blobInput) createBlobLister(ctx v2.Context, cancelCtx context.Context, client beat.Client, persistentStore *statestore.Store, states *states) (*blobPoller, error) {
	// s3ServiceName := awscommon.CreateServiceName("s3", in.config.AWSConfig.FIPSEnabled, in.awsConfig.Region)
	u := ParseEndpointUrl(in.config.Endpoint, in.config.AccountName)
	p := azblob.NewPipeline(in.credential, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*u, p)
	containerClient := serviceURL.NewContainerURL(in.config.Container) // Container names require lowercase

	blobAPI := &azureBlobAPI{
		client: containerClient,
	}

	log := ctx.Logger.With("container", in.config.Container)
	// log.Infof("number_of_workers is set to %v.", in.config.NumberOfWorkers)
	// log.Infof("bucket_list_interval is set to %v.", in.config.BucketListInterval)
	// log.Infof("bucket_list_prefix is set to %v.", in.config.BucketListPrefix)
	// log.Infof("AWS region is set to %v.", in.awsConfig.Region)
	// log.Debugf("AWS S3 service name is %v.", s3ServiceName)

	metricRegistry := monitoring.GetNamespace("dataset").GetRegistry()
	metrics := newInputMetrics(metricRegistry, ctx.ID)

	// fileSelectors := in.config.FileSelectors
	// if len(in.config.FileSelectors) == 0 {
	// 	fileSelectors = []fileSelectorConfig{{ReaderConfig: in.config.ReaderConfig}}
	// }
	blobEventHandlerFactory := newBlobObjectProcessorFactory(log.Named("blob"), metrics, blobAPI, client, in.config.ReaderConfig)
	s3Poller := newBlobPoller(log.Named("blob_poller"),
		metrics,
		blobAPI,
		*blobEventHandlerFactory,
		states,
		persistentStore,
		in.config.Container,
		in.config.BlobListPrefix,
		in.config.NumberOfWorkers,
		in.config.BlobListInterval)

	return s3Poller, nil
}

func ParseEndpointUrl(endpoint string, account_name string) *url.URL {
	var u *url.URL
	if endpoint != "" {
		parsedEndpoint, _ := url.Parse(endpoint)
		if parsedEndpoint.Scheme != "" {
			u, _ = url.Parse(fmt.Sprintf("%s/%s", endpoint, account_name))
		} else {
			u, _ = url.Parse(fmt.Sprintf("https://%s.%s", account_name, endpoint))
		}
	}
	return u
}
