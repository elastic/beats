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

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/state"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

type scheduler interface {
	createJobs(pager *azblob.ContainerListBlobFlatPager) ([]Job, error)
	schedule(ctx context.Context) error
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
	var pager *azblob.ContainerListBlobFlatPager
	var availableWorkers int32

	workerPool := NewWorkerPool(ctx, ais.src.maxWorkers, ais.log)
	availableWorkers = workerPool.AvailableWorkers()

	if !ais.src.poll {
		pager = ais.fetchBlobPager(availableWorkers)
		return ais.scheduleOnce(ctx, pager, workerPool)
	}

	for {

		availableWorkers = workerPool.AvailableWorkers()
		if availableWorkers == 0 {
			continue
		}
		pager = ais.fetchBlobPager(availableWorkers)
		err := ais.scheduleOnce(ctx, pager, workerPool)
		if err != nil {
			return err
		}

		err = timed.Wait(ctx, ais.src.pollInterval)
		if err != nil {
			return err
		}

	}

}

func (ais *azureInputScheduler) scheduleOnce(ctx context.Context, pager *azblob.ContainerListBlobFlatPager, workerPool Pool) error {

	for pager.NextPage(ctx) {
		jobs, err := ais.createJobs(pager)
		if err != nil {
			ais.log.Errorf("Job creation failed for container %s with error %v", ais.src.containerName, err)
			return err
		}

		// If previous checkpoint was saved then look up starting point for new jobs
		if ais.state.Checkpoint().LatestEntryTime != nil {
			jobs = ais.moveToLastSeenJob(jobs)
		}

		// Submits job to worker pool for further processing
		for _, job := range jobs {
			workerPool.Submit(job)
		}
	}
	return nil
}

func (ais *azureInputScheduler) createJobs(pager *azblob.ContainerListBlobFlatPager) ([]Job, error) {
	var jobs []Job
	pageMarker := pager.PageResponse().Marker

	for _, v := range pager.PageResponse().Segment.BlobItems {
		blobURL := fmt.Sprintf("%s%s/%s", ais.serviceURL, ais.src.containerName, *v.Name)
		blobClient, err := fetchBlobClients(blobURL, ais.credential, ais.log)
		if err != nil {
			return nil, err
		}

		job := newAzureInputJobV2(blobClient, v, pageMarker, ais.state, ais.src, ais.publisher)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (ais *azureInputScheduler) fetchBlobPager(batchSize int32) *azblob.ContainerListBlobFlatPager {
	pager := ais.client.ListBlobsFlat(&azblob.ContainerListBlobsFlatOptions{
		Include: []azblob.ListBlobsIncludeItem{
			azblob.ListBlobsIncludeItemMetadata,
			azblob.ListBlobsIncludeItemTags,
		},
		Marker:     ais.state.Checkpoint().Marker,
		MaxResults: &batchSize,
	})

	return pager
}

func (ais *azureInputScheduler) moveToLastSeenJob(jobs []Job) []Job {
	// Jobs are stored in alphabedical order always , hence the latest position can be found on the basis of job name
	var latestJobs []Job
	var jobsToReturn []Job
	counter := 0
	flag := false

	for _, job := range jobs {
		if job.Timestamp().After(*ais.state.Checkpoint().LatestEntryTime) {
			latestJobs = append(latestJobs, job)
		} else if job.Name() == ais.state.Checkpoint().BlobName {
			flag = true
			break
		}
		counter++
	}

	if flag {
		if counter < len(jobs)-1 {
			jobsToReturn = jobs[counter+1:]
		} else {
			jobsToReturn = make([]Job, 0)
		}
	} else {
		jobsToReturn = jobs
	}

	latestJobs = append(latestJobs, jobsToReturn...)

	return latestJobs
}
