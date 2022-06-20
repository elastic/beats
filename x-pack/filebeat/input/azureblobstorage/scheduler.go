// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/elastic-agent-libs/logp"
)

type scheduler interface {
	createJobs(ctx context.Context) ([]*Job, bool, error)
	schedule() error
	scheduleWithPoll() error
}

type azureInputScheduler struct {
	publisher  *cursor.Publisher
	client     *azblob.ContainerClient
	credential *azblob.SharedKeyCredential
	src        *source
	cfg        *config
	state      *state
	log        *logp.Logger
	serviceURL string
}

func newAzureInputScheduler(publisher *cursor.Publisher, client *azblob.ContainerClient,
	credential *azblob.SharedKeyCredential, src *source, cfg *config,
	state *state, serviceURL string, log *logp.Logger) scheduler {

	return &azureInputScheduler{
		publisher:  publisher,
		client:     client,
		credential: credential,
		src:        src,
		cfg:        cfg,
		state:      state,
		log:        log,
		serviceURL: serviceURL,
	}
}

func (ais *azureInputScheduler) schedule() error {
	var wg sync.WaitGroup
	var failuers []error
	errchan := make(chan error)
	return nil
}

func (ais *azureInputScheduler) scheduleWithPoll() error {
	return nil
}

func (ais *azureInputScheduler) createJobs(ctx context.Context) ([]*Job, bool, error) {
	var jobs []*Job

	pager := ais.client.ListBlobsHierarchy("/", &azblob.ContainerListBlobsHierarchyOptions{
		Include: []azblob.ListBlobsIncludeItem{
			azblob.ListBlobsIncludeItemMetadata,
			azblob.ListBlobsIncludeItemTags,
		},
		Marker:     ais.state.marker,
		MaxResults: &ais.src.batchSize,
	})

	for _, v := range pager.PageResponse().Segment.BlobItems {
		blobURL := fmt.Sprintf("%s%s/%s", ais.serviceURL, ais.src.containerName, *v.Name)
		blobClient, err := fetchBlobClients(blobURL, ais.credential, ais.log)
		if err != nil {
			return nil, false, err
		}

		job := newAzureInputJob(blobClient, v, ais.state, ais.publisher)
		jobs = append(jobs, &job)
	}

	return jobs, pager.NextPage(ctx), nil
}
