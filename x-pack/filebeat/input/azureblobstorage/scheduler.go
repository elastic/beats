// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"
	"fmt"
	"sync"

	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

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
	client     *azcontainer.Client
	credential *serviceCredentials
	src        *Source
	cfg        *config
	state      *state
	log        *logp.Logger
	limiter    *limiter
	serviceURL string
}

// newScheduler, returns a new scheduler instance
func newScheduler(publisher cursor.Publisher, client *azcontainer.Client,
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
	defer s.limiter.wait()
	pager := s.fetchBlobPager(int32(s.src.MaxWorkers))
	fileSelectorLen := len(s.src.FileSelectors)
	var numBlobs, numJobs int

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}

		numBlobs += len(resp.Segment.BlobItems)
		s.log.Debugf("scheduler: %d blobs fetched for current batch", len(resp.Segment.BlobItems))

		var jobs []*job
		for _, v := range resp.Segment.BlobItems {
			// if file selectors are present, then only select the files that match the regex
			if fileSelectorLen != 0 && !s.isFileSelected(*v.Name) {
				continue
			}
			// date filter is applied on last modified time of the blob
			if s.src.TimeStampEpoch != nil && v.Properties.LastModified.Unix() < *s.src.TimeStampEpoch {
				continue
			}
			blobURL := s.serviceURL + s.src.ContainerName + "/" + *v.Name
			blobCreds := &blobCredentials{
				serviceCreds:  s.credential,
				blobName:      *v.Name,
				containerName: s.src.ContainerName,
			}

			blobClient, err := fetchBlobClient(blobURL, blobCreds, s.log)
			if err != nil {
				s.log.Errorf("Job creation failed for container %s with error %v", s.src.ContainerName, err)
				return err
			}

			job := newJob(blobClient, v, blobURL, s.state, s.src, s.publisher, s.log)
			jobs = append(jobs, job)
		}

		// If previous checkpoint was saved then look up starting point for new jobs
		if !s.state.checkpoint().LatestEntryTime.IsZero() {
			jobs = s.moveToLastSeenJob(jobs)
		}

		s.log.Debugf("scheduler: %d jobs scheduled for current batch", len(jobs))

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

		s.log.Debugf("scheduler: total objects read till now: %d\nscheduler: total jobs scheduled till now: %d", numBlobs, numJobs)
		if len(jobs) != 0 {
			s.log.Debugf("scheduler: first job in current batch: %s\nscheduler: last job in current batch: %s", jobs[0].name(), jobs[len(jobs)-1].name())
		}
	}

	return nil
}

// fetchJobID returns a job id which is a combination of worker id, container name and blob name
func fetchJobID(workerId int, containerName string, blobName string) string {
	jobID := fmt.Sprintf("%s-%s-worker-%d", containerName, blobName, workerId)

	return jobID
}

// fetchBlobPager fetches the current blob page object given a batch size & a page marker.
// The page marker has been disabled since it was found that it operates on the basis of
// lexicographical order, and not on the basis of the latest file uploaded, meaning if a blob with a name
// of lesser lexicographic value is uploaded after a blob with a name of higher value, the latest
// marker stored in the checkpoint will not retrieve that new blob, this distorts the polling logic
// hence disabling it for now, until more feedback is given. Disabling this how ever makes the sheduler loop
// through all the blobs on every poll action to arrive at the latest checkpoint.
// [NOTE] : There are no api's / sdk functions that list blobs via timestamp/latest entry, it's always lexicographical order
func (s *scheduler) fetchBlobPager(batchSize int32) *azruntime.Pager[azblob.ListBlobsFlatResponse] {
	pager := s.client.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
		Include: azcontainer.ListBlobsInclude{
			Metadata: true,
			Tags:     true,
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
		switch {
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

	// in a senario where there are some jobs which have a greater timestamp
	// but lesser alphanumeric order and some jobs have greater alphanumeric order
	// than the current checkpoint blob name, then we append the latest jobs
	if len(jobsToReturn) != len(jobs) && len(latestJobs) > 0 {
		jobsToReturn = append(latestJobs, jobsToReturn...)
	}

	return jobsToReturn
}

func (s *scheduler) isFileSelected(name string) bool {
	for _, sel := range s.src.FileSelectors {
		if sel.Regex == nil || sel.Regex.MatchString(name) {
			return true
		}
	}
	return false
}
