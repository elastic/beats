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
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/state"
	"github.com/elastic/elastic-agent-libs/logp"
)

type scheduler interface {
	createJobs(pager *azblob.ContainerListBlobFlatPager) ([]Job, error)
	schedule(ctx context.Context) error
	scheduleWithPoll() error
}

type azureInputScheduler struct {
	publisher  cursor.Publisher
	client     *azblob.ContainerClient
	credential *azblob.SharedKeyCredential
	src        *source
	cfg        *config
	state      *state.State
	log        *logp.Logger
	serviceURL string
}

func newAzureInputScheduler(publisher cursor.Publisher, client *azblob.ContainerClient,
	credential *azblob.SharedKeyCredential, src *source, cfg *config,
	state *state.State, serviceURL string, log *logp.Logger) scheduler {

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

func (ais *azureInputScheduler) schedule(ctx context.Context) error {
	var wg sync.WaitGroup
	errs := make(chan error)
	jobCounter := 0
	pager := ais.fetchBlobPager()

	// Iterate over the error channel in a go routine and print errors as and when they come
	go func() {
		for e := range errs {
			ais.log.Errorf("Scheduler error : %v ", e)
		}
	}()
	page := 0
	for pager.NextPage(ctx) {
		jobs, err := ais.createJobs(pager)
		if err != nil {
			ais.log.Errorf("Job creation failed for container %s with error %v", ais.src.containerName, err)
			return err
		}

		pageMarker := pager.PageResponse().Marker
		for _, job := range jobs {
			wg.Add(1)
			jobID := fetchJobID(jobCounter, ais.src.containerName, job.Name())
			go job.Do(ctx, jobID, pageMarker, &wg, errs)
			fmt.Printf("PAGE : %d --- JOB WITH ID %v EXECUTED\n", page, jobID)
			jobCounter++
		}
		wg.Wait()
		page++
	}
	close(errs)

	return nil
}

func (ais *azureInputScheduler) scheduleWithPoll() error {
	return nil
}

func (ais *azureInputScheduler) createJobs(pager *azblob.ContainerListBlobFlatPager) ([]Job, error) {
	var jobs []Job

	for _, v := range pager.PageResponse().Segment.BlobItems {
		blobURL := fmt.Sprintf("%s%s/%s", ais.serviceURL, ais.src.containerName, *v.Name)
		blobClient, err := fetchBlobClients(blobURL, ais.credential, ais.log)
		if err != nil {
			return nil, err
		}

		job := newAzureInputJob(blobClient, v, ais.state, ais.src, ais.publisher)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (ais *azureInputScheduler) fetchBlobPager() *azblob.ContainerListBlobFlatPager {
	if ais.state.Checkpoint().Marker != nil {
		fmt.Printf("\nPAGER MARKER : %v\n", *ais.state.Checkpoint().Marker)
	} else {
		fmt.Printf("\nPAGER MARKER : %v\n", ais.state.Checkpoint().Marker)
	}
	pager := ais.client.ListBlobsFlat(&azblob.ContainerListBlobsFlatOptions{
		Include: []azblob.ListBlobsIncludeItem{
			azblob.ListBlobsIncludeItemMetadata,
			azblob.ListBlobsIncludeItemTags,
		},
		Marker:     ais.state.Checkpoint().Marker,
		MaxResults: &ais.src.batchSize,
	})

	return pager
}
