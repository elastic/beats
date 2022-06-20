// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/pkg/errors"
)

const (
	inputName = "azureblobstorage"
)

type source struct {
	containerName string
	accountName   string
	batchSize     int32
	batchIntrval  int32
	poll          bool
}

type azurebsInput struct {
	config     config
	client     *azblob.ServiceClient
	credential *azblob.SharedKeyCredential
	serviceURL string
}

type blobClientObj struct {
	client *azblob.BlockBlobClient
	blob   *azblob.BlobItemInternal
}

func Plugin(log *logp.Logger, store cursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "Azure Blob Storage logs",
		Doc:        "Collect logs from Azure Blob Storage Service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       inputName,
			Configure:  configure,
		},
	}
}

func configure(cfg *conf.C) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, errors.Wrap(err, "reading config")
	}

	var sources []cursor.Source
	for _, c := range config.Containers {
		sources = append(sources, &source{
			accountName:   config.AccountName,
			containerName: c.Name,
			batchSize:     c.BatchSize,
			batchIntrval:  c.BatchIntervalMs,
		})
	}

	url := fmt.Sprintf("https://%s.blob.core.windows.net/", config.AccountName)
	return sources, &azurebsInput{config: config, serviceURL: url}, nil
}

func (s *source) Name() string {
	return s.accountName + "::" + s.containerName
}

func (input *azurebsInput) Name() string {
	return inputName
}

func (input *azurebsInput) Test(src cursor.Source, ctx v2.TestContext) error {
	return nil
}

func (input *azurebsInput) Run(inputCtx v2.Context, src cursor.Source, cursor cursor.Cursor, publisher cursor.Publisher) error {
	var err error
	var st *state

	currentSource := src.(*source)
	ctx := context.Background()
	log := inputCtx.Logger.With("account_name", currentSource.accountName).With("container", currentSource.containerName)
	log.Info("Running azure blob storage for account %s", input.config.AccountName)

	serviceClient, credential, err := fetchServiceClientAndCreds(input.config, input.serviceURL, log)
	if err != nil {
		return err
	}
	containerClient, err := fetchContainerClient(serviceClient, currentSource.containerName, log)
	if err != nil {
		return err
	}
	input.client = serviceClient
	input.credential = credential

	if cursor.IsNew() {
		st = newState()
	} else {
		cursor.Unpack(&st)
	}

	scheduler := newAzureInputScheduler(&publisher, containerClient, input.credential, currentSource, &input.config, st, input.serviceURL, log)
	scheduler.schedule()

	ctx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputCtx.Cancelation.Done():
		case <-ctx.Done():
		}
	}()
	defer cancelInputCtx()
	return nil
}
