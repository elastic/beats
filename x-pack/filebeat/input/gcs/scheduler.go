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
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

// limiter, is used to limit the number of goroutines from blowing up the stack
type limiter struct {
	wg sync.WaitGroup
	// limit specifies the maximum number
	// of concurrent jobs to perform.
	limit chan struct{}
}
type scheduler struct {
	publisher cursor.Publisher
	bucket    *storage.BucketHandle
	src       *Source
	cfg       *config
	state     *state
	log       *logp.Logger
	limiter   *limiter
}

// newScheduler, returns a new scheduler instance
func newScheduler(publisher cursor.Publisher, bucket *storage.BucketHandle, src *Source, cfg *config,
	state *state, log *logp.Logger,
) *scheduler {
	return &scheduler{
		publisher: publisher,
		bucket:    bucket,
		src:       src,
		cfg:       cfg,
		state:     state,
		log:       log,
		limiter:   &limiter{limit: make(chan struct{}, src.MaxWorkers)},
	}
}

// Schedule, is responsible for fetching & scheduling jobs using the workerpool model
func (s *scheduler) schedule(ctx context.Context) error {
	if !s.src.Poll {
		return s.scheduleOnce(ctx)
	}

	for {
		err := s.scheduleOnce(ctx)
		if err != nil {
			return err
		}

		err = timed.Wait(ctx, s.src.PollInterval)
		if err != nil {
			return err
		}
	}
}

// acquire gets an available worker thread.
func (l *limiter) acquire() {
	l.wg.Add(1)
	l.limit <- struct{}{}
}

func (l *limiter) wait() {
	l.wg.Wait()
}

// release puts back a worker thread.
func (l *limiter) release() {
	<-l.limit
	l.wg.Done()
}

func (s *scheduler) scheduleOnce(ctx context.Context) error {
	defer s.limiter.wait()
	pager := s.fetchObjectPager(ctx, s.src.MaxWorkers)
	var numObs, numJobs int
	for {
		var objects []*storage.ObjectAttrs
		nextPageToken, err := pager.NextPage(&objects)
		if err != nil {
			return err
		}
		numObs += len(objects)
		jobs := s.createJobs(objects, s.log)
		s.log.Debugf("scheduler: %d objects fetched for current batch", len(objects))

		// If previous checkpoint was saved then look up starting point for new jobs
		if !s.state.checkpoint().LatestEntryTime.IsZero() {
			jobs = s.moveToLastSeenJob(jobs)
			if len(s.state.checkpoint().FailedJobs) > 0 {
				jobs = s.addFailedJobs(ctx, jobs)
			}
		}
		s.log.Debugf("scheduler: %d jobs scheduled for current batch", len(jobs))

		// distributes jobs among workers with the help of a limiter
		for i, job := range jobs {
			numJobs++
			id := fetchJobID(i, s.src.BucketName, job.Name())
			job := job
			s.limiter.acquire()
			go func() {
				defer s.limiter.release()
				job.do(ctx, id)
			}()
		}

		s.log.Debugf("scheduler: total objects read till now: %d\nscheduler: total jobs scheduled till now: %d", numObs, numJobs)
		if len(jobs) != 0 {
			s.log.Debugf("scheduler: first job in current batch: %s\nscheduler: last job in current batch: %s", jobs[0].Name(), jobs[len(jobs)-1].Name())
		}

		if nextPageToken == "" {
			break
		}
	}

	return nil
}

// fetchJobID returns a job id which is a combination of worker id, bucket name and object name
func fetchJobID(workerId int, bucketName string, objectName string) string {
	jobID := fmt.Sprintf("%s-%s-worker-%d", bucketName, objectName, workerId)

	return jobID
}

func (s *scheduler) createJobs(objects []*storage.ObjectAttrs, log *logp.Logger) []*job {
	//nolint:prealloc // No need to preallocate the slice
	var jobs []*job
	for _, obj := range objects {
		// if file selectors are present, then only select the files that match the regex
		if len(s.src.FileSelectors) != 0 && !s.isFileSelected(obj.Name) {
			continue
		}
		// date filter is applied on last updated time of the object
		if s.src.TimeStampEpoch != nil && obj.Updated.Unix() < *s.src.TimeStampEpoch {
			continue
		}
		// check required to ignore directories & sub folders, since there is no inbuilt option to
		// do so. In gcs all the directories are emulated and represented by a prefix, we can
		// define specific prefix's & delimiters to ignore known directories but there is no generic
		// way to do so.
		file := strings.Split(obj.Name, "/")
		if len(file) > 1 && file[len(file)-1] == "" {
			continue
		}

		objectURI := "gs://" + s.src.BucketName + "/" + obj.Name
		job := newJob(s.bucket, obj, objectURI, s.state, s.src, s.publisher, log, false)
		jobs = append(jobs, job)
	}

	return jobs
}

// fetchObjectPager fetches the page handler for objects, given a batch size.
// [NOTE] : There are no api's / sdk functions that list blobs via timestamp/latest entry, it's always lexicographical order
func (s *scheduler) fetchObjectPager(ctx context.Context, pageSize int) *iterator.Pager {
	bktIt := s.bucket.Objects(ctx, &storage.Query{})
	pager := iterator.NewPager(bktIt, pageSize, "")

	return pager
}

// moveToLastSeenJob, moves to the latest job position past the last seen job
// Jobs are stored in lexicographical order always, hence the latest position can be found either on the basis of job name or timestamp
func (s *scheduler) moveToLastSeenJob(jobs []*job) []*job {
	var latestJobs []*job
	jobsToReturn := make([]*job, 0)
	counter := 0
	flag := false
	ignore := false

	for _, job := range jobs {
		switch {
		case job.Timestamp().After(s.state.checkpoint().LatestEntryTime):
			latestJobs = append(latestJobs, job)
		case job.Name() == s.state.checkpoint().ObjectName:
			flag = true
		case job.Name() > s.state.checkpoint().ObjectName:
			flag = true
			counter--
		case job.Name() <= s.state.checkpoint().ObjectName && (!ignore):
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
	// than the current checkpoint object name, then we append the latest jobs
	if len(jobsToReturn) != len(jobs) && len(latestJobs) > 0 {
		jobsToReturn = append(latestJobs, jobsToReturn...)
	}

	return jobsToReturn
}

func (s *scheduler) addFailedJobs(ctx context.Context, jobs []*job) []*job {
	jobMap := make(map[string]bool)

	for _, j := range jobs {
		jobMap[j.Name()] = true
	}

	failedJobs := s.state.checkpoint().FailedJobs
	s.log.Debugf("scheduler: %d failed jobs found", len(failedJobs))
	fj := 0
	for name := range failedJobs {
		if !jobMap[name] {
			obj, err := s.bucket.Object(name).Attrs(ctx)
			if err != nil {
				s.log.Errorf("adding failed job %s to job list caused an error: %w", err)
				continue
			}

			objectURI := "gs://" + s.src.BucketName + "/" + obj.Name
			job := newJob(s.bucket, obj, objectURI, s.state, s.src, s.publisher, s.log, true)
			jobs = append(jobs, job)
			s.log.Debugf("scheduler: adding failed job number %d with name %s to job current list", fj, job.Name())
			fj++
		}
	}
	return jobs
}

func (s *scheduler) isFileSelected(name string) bool {
	for _, sel := range s.src.FileSelectors {
		if sel.Regex.MatchString(name) {
			return true
		}
	}
	return false
}
