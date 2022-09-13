// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/job"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/state"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

type scheduler interface {
	createJobs(objects []*storage.ObjectAttrs, log *logp.Logger) []job.Job
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

// limiter, is used to limit the number of goroutines from blowing up the stack
type limiter struct {
	ctx context.Context
	wg  sync.WaitGroup
	// limit specifies the maximum number
	// of concurrent jobs to perform.
	limit chan struct{}
}

// acquire gets an available worker thread.
func (c *limiter) acquire() {
	c.wg.Add(1)
	c.limit <- struct{}{}
}

// release puts pack a worker thread.
func (c *limiter) release() {
	<-c.limit
	c.wg.Done()
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
func (s *gcsInputScheduler) Schedule(ctx context.Context) error {
	var pager *iterator.Pager
	var availableWorkers int

	// workerPool := pool.NewWorkerPool(ctx, s.src.MaxWorkers, s.log)
	// workerPool.Start()

	lmtr := &limiter{limit: make(chan struct{}, s.src.MaxWorkers), ctx: ctx}

	if !s.src.Poll {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, s.src.BucketTimeOut)
		defer cancel()
		for {
			availableWorkers = s.src.MaxWorkers - len(lmtr.limit)
			if availableWorkers == 0 {
				continue
			} else {
				break
			}
		}
		pager = s.fetchObjectPager(ctxWithTimeout, availableWorkers)
		return lmtr.scheduleOnce(ctxWithTimeout, pager, s)
	}

	for {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, s.src.BucketTimeOut)
		defer cancel()

		availableWorkers = s.src.MaxWorkers - len(lmtr.limit)
		if availableWorkers == 0 {
			continue
		}

		// availableWorkers is used as the batch size for a blob page so that
		// work distribution remains efficient
		pager = s.fetchObjectPager(ctxWithTimeout, availableWorkers)
		err := lmtr.scheduleOnce(ctxWithTimeout, pager, s)
		if err != nil {
			return err
		}

		err = timed.Wait(ctx, s.src.PollInterval)
		if err != nil {
			return err
		}
	}
}

func (l *limiter) scheduleOnce(ctx context.Context, pager *iterator.Pager, s *gcsInputScheduler) error {
	for {
		var objects []*storage.ObjectAttrs
		nextPageToken, err := pager.NextPage(&objects)
		if err != nil {
			return err
		}

		jobs := s.createJobs(objects, s.log)

		// If previous checkpoint was saved then look up starting point for new jobs
		if s.state.Checkpoint().LatestEntryTime != nil {
			jobs = s.moveToLastSeenJob(jobs)
			if len(s.state.Checkpoint().FailedJobs) > 0 {
				jobs = s.addFailedJobs(ctx, jobs)
			}
		}

		// distributes jobs among workers with the help of a limiter
		for i, job := range jobs {
			id := fetchJobID(i, s.src.BucketName, job.Name())
			job := job
			l.acquire()
			go func() {
				defer l.release()
				job.Do(l.ctx, id)
			}()
		}

		if nextPageToken == "" {
			break
		}
	}
	return nil
}

func (s *gcsInputScheduler) createJobs(objects []*storage.ObjectAttrs, log *logp.Logger) []job.Job {
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

		objectURI := "gs://" + s.src.BucketName + "/" + obj.Name
		job := job.NewGcsInputJob(s.bucket, obj, objectURI, s.state, s.src, s.publisher, log, false)
		jobs = append(jobs, job)
	}

	return jobs
}

// fetchObjectPager fetches the page handler for objects, given a batch size.
// [NOTE] : There are no api's / sdk functions that list blobs via timestamp/latest entry, it's always lexicographical order
func (s *gcsInputScheduler) fetchObjectPager(ctx context.Context, pageSize int) *iterator.Pager {
	bktIt := s.bucket.Objects(ctx, &storage.Query{})
	pager := iterator.NewPager(bktIt, pageSize, "")

	return pager
}

// moveToLastSeenJob, moves to the latest job position past the last seen job
// Jobs are stored in lexicographical order always , hence the latest position can be found either on the basis of job name or timestamp
func (s *gcsInputScheduler) moveToLastSeenJob(jobs []job.Job) []job.Job {
	var latestJobs []job.Job
	jobsToReturn := make([]job.Job, 0)
	counter := 0
	flag := false
	ignore := false

	for _, job := range jobs {
		if job.Timestamp().After(*s.state.Checkpoint().LatestEntryTime) {
			latestJobs = append(latestJobs, job)
		} else if job.Name() == s.state.Checkpoint().ObjectName {
			flag = true
			break
		} else if job.Name() > s.state.Checkpoint().ObjectName {
			flag = true
			counter--
			break
		} else if job.Name() <= s.state.Checkpoint().ObjectName && !ignore {
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

func (s *gcsInputScheduler) addFailedJobs(ctx context.Context, jobs []job.Job) []job.Job {
	jobMap := make(map[string]bool)

	for _, j := range jobs {
		jobMap[j.Name()] = true
	}

	for name := range s.state.Checkpoint().FailedJobs {
		if !jobMap[name] {
			obj, err := s.bucket.Object(name).Attrs(ctx)
			if err != nil {
				s.log.Errorf("adding failed job %s to job list caused an error : %w", err)
			}

			objectURI := "gs://" + s.src.BucketName + "/" + obj.Name
			job := job.NewGcsInputJob(s.bucket, obj, objectURI, s.state, s.src, s.publisher, s.log, true)
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// fetchJobID returns a job id which is a combination of worker id, bucket name and object name
func fetchJobID(workerId int, bucketName string, objectName string) string {
	jobID := fmt.Sprintf("%s-%s-worker-%d", bucketName, objectName, workerId)

	return jobID
}
