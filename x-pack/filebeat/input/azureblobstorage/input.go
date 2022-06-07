// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName = "azureblobstorage"
)

type azurebsInputManager struct {
	store beater.StateStore
	log   *logp.Logger
}

type azurebsInput struct {
	config     config
	store      beater.StateStore
	client     *azblob.ServiceClient
	credential *azblob.SharedKeyCredential
	log        *logp.Logger
	serviceURL string
}

type blobClientObj struct {
	client *azblob.BlockBlobClient
	blobs  []*azblob.BlobItemInternal
}

func Plugin(log *logp.Logger, store beater.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "Collect logs from Azure Blob Storage",
		Manager:    &azurebsInputManager{store: store, log: log},
	}
}

func (im *azurebsInputManager) Init(grp unison.Group, mode v2.Mode) error {
	return nil
}

func (im *azurebsInputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config, im.log, im.store)
}

func newInput(config config, log *logp.Logger, store beater.StateStore) (*azurebsInput, error) {
	url := fmt.Sprintf("https://%s.blob.core.windows.net/", config.AccountName)

	serviceClient, credential, err := fetchServiceClientAndCreds(config, url, log)
	if err != nil {
		return nil, err
	}

	return &azurebsInput{
		config:     config,
		store:      store,
		client:     serviceClient,
		serviceURL: url,
		log:        log,
		credential: credential,
	}, nil
}

func (input *azurebsInput) Name() string {
	return inputName
}

func (input *azurebsInput) Test(ctx v2.TestContext) error {
	return nil
}

func (input *azurebsInput) Run(inputCtx v2.Context, pipeline beat.Pipeline) error {
	var err error
	ctx := context.Background()

	persistentStore, err := input.store.Access()
	if err != nil {
		return fmt.Errorf("cannot connect to persistent storage %v", err)
	}
	defer persistentStore.Close()

	input.collect(ctx, persistentStore)

	ctx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputCtx.Cancelation.Done():
		case <-ctx.Done():
		}
	}()
	defer cancelInputCtx()
	input.log.Info("Running azure blob storage")
	return nil
}
