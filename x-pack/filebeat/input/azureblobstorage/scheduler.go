// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

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

// acquire gets an available worker thread.
func (l *limiter) acquire() {
	l.wg.Add(1)
	l.limit <- struct{}{}
}

func (l *limiter) wait() {
	l.wg.Wait()
}

// release puts pack a worker thread.
func (l *limiter) release() {
	<-l.limit
	l.wg.Done()
}

type scheduler struct {
	publisher  cursor.Publisher
	client     *azblob.ContainerClient
	credential *serviceCredentials
	src        *Source
	cfg        *config
	state      *state
	log        *logp.Logger
	limiter    *limiter
	serviceURL string
}

// newScheduler, returns a new scheduler instance
func newScheduler(publisher cursor.Publisher, client *azblob.ContainerClient,
	credential *serviceCredentials, src *Source, cfg *config,
	state *state, serviceURL string, log *logp.Logger,
) *scheduler {
	return &scheduler{
		publisher:  publisher,
		client:     client,
		credential: credential,
		src:        src,
		cfg:        cfg,
		state:      state,
		log:        log,
		limiter:    &limiter{limit: make(chan struct{}, src.MaxWorkers)},
		serviceURL: serviceURL,
	}
}

// schedule, is responsible for fetching & scheduling jobs using the workerpool model
func (s *scheduler) schedule(ctx context.Context) error {
	defer s.limiter.wait()
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

func (s *scheduler) scheduleOnce(ctx context.Context) error {
	pager := s.fetchBlobPager(int32(s.src.MaxWorkers))
	for pager.NextPage(ctx) {
		jobs, err := s.createJobs(pager)
		if err != nil {
			s.log.Errorf("Job creation failed for container %s with error %v", s.src.ContainerName, err)
			return err
		}

		// If previous checkpoint was saved then look up starting point for new jobs
		if !s.state.checkpoint().LatestEntryTime.IsZero() {
			jobs = s.moveToLastSeenJob(jobs)
		}

		// distributes jobs among workers with the help of a limiter
		for i, job := range jobs {
			id := fetchJobID(i, s.src.ContainerName, job.name())
			job := job
			s.limiter.acquire()
			go func() {
				defer s.limiter.release()
				job.do(ctx, id)
			}()
		}
	}

	return pager.Err()
}

// fetchJobID returns a job id which is a combination of worker id, container name and blob name
func fetchJobID(workerId int, containerName string, blobName string) string {
	jobID := fmt.Sprintf("%s-%s-worker-%d", containerName, blobName, workerId)

	return jobID
}

func (s *scheduler) createJobs(pager *azblob.ContainerListBlobFlatPager) ([]*job, error) {
	var jobs []*job

	for _, v := range pager.PageResponse().Segment.BlobItems {
		blobURL := s.serviceURL + s.src.ContainerName + "/" + *v.Name
		blobCreds := &blobCredentials{
			serviceCreds:  s.credential,
			blobName:      *v.Name,
			containerName: s.src.ContainerName,
		}

		blobClient, err := fetchBlobClient(blobURL, blobCreds, s.log)
		if err != nil {
			return nil, err
		}

		job := newJob(blobClient, v, blobURL, s.state, s.src, s.publisher, s.log)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// fetchBlobPager fetches the current blob page object given a batch size & a page marker.
// The page marker has been disabled since it was found that it operates on the basis of
// lexicographical order, and not on the basis of the latest file uploaded, meaning if a blob with a name
// of lesser lexicographic value is uploaded after a blob with a name of higher value, the latest
// marker stored in the checkpoint will not retrieve that new blob, this distorts the polling logic
// hence disabling it for now, until more feedback is given. Disabling this how ever makes the sheduler loop
// through all the blobs on every poll action to arrive at the latest checkpoint.
// [NOTE] : There are no api's / sdk functions that list blobs via timestamp/latest entry, it's always lexicographical order
func (s *scheduler) fetchBlobPager(batchSize int32) *azblob.ContainerListBlobFlatPager {
	pager := s.client.ListBlobsFlat(&azblob.ContainerListBlobsFlatOptions{
		Include: []azblob.ListBlobsIncludeItem{
			azblob.ListBlobsIncludeItemMetadata,
			azblob.ListBlobsIncludeItemTags,
		},
		MaxResults: &batchSize,
	})

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
		switch offset, isPartial := s.state.cp.PartiallyProcessed[*job.blob.Name]; {
		case isPartial:
			job.offset = offset
			latestJobs = append(latestJobs, job)
		case job.timestamp().After(s.state.checkpoint().LatestEntryTime):
			latestJobs = append(latestJobs, job)
		case job.name() == s.state.checkpoint().BlobName:
			flag = true
		case job.name() > s.state.checkpoint().BlobName:
			flag = true
			counter--
		case job.name() <= s.state.checkpoint().BlobName && (!ignore):
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
	// but lesser alphanumeric order and some jobs have greater alphanumeric order
	// than the current checkpoint or partially completed jobs are present
	if len(jobsToReturn) != len(jobs) && len(latestJobs) > 0 {
		jobsToReturn = append(latestJobs, jobsToReturn...)
	}

	return jobsToReturn
}
