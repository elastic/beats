// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package gcs

import (
	"context"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/job"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/pool"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/state"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

type scheduler interface {
	createJobs(objects []*storage.ObjectAttrs) []job.Job
	Schedule(ctx context.Context) error
}

type gcsInputScheduler struct {
	publisher cursor.Publisher
	bucket    *storage.BucketHandle
	src       *types.Source
	cfg       *config
	state     *state.State
	log       *logp.Logger
}

// NewGcsInputScheduler, returns a new scheduler instance
func NewGcsInputScheduler(publisher cursor.Publisher, bucket *storage.BucketHandle, src *types.Source, cfg *config,
	state *state.State, log *logp.Logger,
) scheduler {
	return &gcsInputScheduler{
		publisher: publisher,
		bucket:    bucket,
		src:       src,
		cfg:       cfg,
		state:     state,
		log:       log,
	}
}

// Schedule, is responsible for fetching & scheduling jobs using the workerpool model
func (gcsis *gcsInputScheduler) Schedule(ctx context.Context) error {
	var pager *iterator.Pager
	var availableWorkers int

	workerPool := pool.NewWorkerPool(ctx, gcsis.src.MaxWorkers, gcsis.log)
	workerPool.Start()

	if !gcsis.src.Poll {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, gcsis.src.BucketTimeOut)
		defer cancel()
		availableWorkers = workerPool.AvailableWorkers()
		pager = gcsis.fetchObjectPager(ctxWithTimeout, availableWorkers)
		return gcsis.scheduleOnce(ctx, pager, workerPool)
	}

	for {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, gcsis.src.BucketTimeOut)
		defer cancel()

		availableWorkers = workerPool.AvailableWorkers()
		if availableWorkers == 0 {
			continue
		}

		// availableWorkers is used as the batch size for a blob page so that
		// work distribution remains efficient
		pager = gcsis.fetchObjectPager(ctxWithTimeout, availableWorkers)
		err := gcsis.scheduleOnce(ctx, pager, workerPool)
		if err != nil {
			return err
		}

		err = timed.Wait(ctx, gcsis.src.PollInterval)
		if err != nil {
			return err
		}
	}
}

func (gcsis *gcsInputScheduler) scheduleOnce(ctx context.Context, pager *iterator.Pager, workerPool pool.Pool) error {
	for {
		var objects []*storage.ObjectAttrs
		nextPageToken, err := pager.NextPage(&objects)
		if err != nil {
			return err
		}

		jobs := gcsis.createJobs(objects)

		// If previous checkpoint was saved then look up starting point for new jobs
		if gcsis.state.Checkpoint().LatestEntryTime != nil {
			jobs = gcsis.moveToLastSeenJob(jobs)
		}

		// Submits job to worker pool for further processing
		for _, job := range jobs {
			workerPool.Submit(job)
		}

		if nextPageToken == "" {
			break
		}
	}
	return nil
}

func (gcsis *gcsInputScheduler) createJobs(objects []*storage.ObjectAttrs) []job.Job {
	var jobs []job.Job

	for _, obj := range objects {
		// check required to ignore directories & sub folders, since there is no inbuilt option to
		// do so. In gcs all the directories are emulated and represented by a prefix, we can
		// define specific prefix's & delimiters to ignore known directories but there is no generic
		// way to do so.
		file := strings.Split(obj.Name, "/")
		if len(file) > 1 && file[len(file)-1] == "" {
			continue
		}

		objectURI := "gs://" + gcsis.src.BucketName + "/" + obj.Name
		job := job.NewGcsInputJob(gcsis.bucket, obj, objectURI, gcsis.state, gcsis.src, gcsis.publisher)
		jobs = append(jobs, job)
	}

	return jobs
}

// fetchObjectPager fetches the page handler for objects, given a batch size.
// [NOTE] : There are no api's / sdk functions that list blobs via timestamp/latest entry, it's always lexicographical order
func (gcsis *gcsInputScheduler) fetchObjectPager(ctx context.Context, pageSize int) *iterator.Pager {
	bktIt := gcsis.bucket.Objects(ctx, &storage.Query{})
	pager := iterator.NewPager(bktIt, pageSize, "")

	return pager
}

// moveToLastSeenJob, moves to the latest job position past the last seen job
// Jobs are stored in lexicographical order always , hence the latest position can be found either on the basis of job name or timestamp
func (gcsis *gcsInputScheduler) moveToLastSeenJob(jobs []job.Job) []job.Job {
	var latestJobs []job.Job
	jobsToReturn := make([]job.Job, 0)
	counter := 0
	flag := false
	ignore := false

	for _, job := range jobs {
		if job.Timestamp().After(*gcsis.state.Checkpoint().LatestEntryTime) {
			latestJobs = append(latestJobs, job)
		} else if job.Name() == gcsis.state.Checkpoint().ObjectName {
			flag = true
			break
		} else if job.Name() > gcsis.state.Checkpoint().ObjectName {
			flag = true
			counter--
			break
		} else if job.Name() <= gcsis.state.Checkpoint().ObjectName && !ignore {
			ignore = true
		}
		counter++
	}

	if flag && (counter < len(jobs)-1) {
		jobsToReturn = jobs[counter+1:]
	} else if !flag && !ignore {
		jobsToReturn = jobs
	}

	// in a senario where there are some jobs which have a later time stamp
	// but lesser lexicographic order and some jobs have greater lexicographic order
	// than the current checkpoint
	if len(jobsToReturn) != len(jobs) && len(latestJobs) > 0 {
		jobsToReturn = append(latestJobs, jobsToReturn...)
	}

	return jobsToReturn
}
