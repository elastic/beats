// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Job interface {
	Do(ctx context.Context, log *logp.Logger) error
}

type azureInputJob struct {
	client    *azblob.BlockBlobClient
	blob      *azblob.BlobItemInternal
	state     *state
	publisher *cursor.Publisher
}

func newAzureInputJob(client *azblob.BlockBlobClient, blob *azblob.BlobItemInternal, state *state, publisher *cursor.Publisher) Job {

	return &azureInputJob{
		client:    client,
		blob:      blob,
		state:     state,
		publisher: publisher,
	}
}

func (aij *azureInputJob) Do(ctx context.Context, log *logp.Logger) error {
	return nil
}
