// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pool

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/job"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Worker interface {
	Process(work job.Job)
	Start()
	Stop()
}

type worker struct {
	id        int
	ctx       context.Context
	wg        *sync.WaitGroup
	errChan   chan<- error
	job       chan job.Job
	readyPool chan chan job.Job
	quit      chan bool
	log       *logp.Logger
}

func NewWorker(ctx context.Context, id int, readyPool chan chan job.Job, wg *sync.WaitGroup, errChan chan<- error, log *logp.Logger) Worker {
	return &worker{
		id:        id,
		ctx:       ctx,
		wg:        wg,
		errChan:   errChan,
		readyPool: readyPool,
		job:       make(chan job.Job),
		quit:      make(chan bool),
		log:       log,
	}
}

func (w *worker) Process(work job.Job) {
	// do the work
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			w.errChan <- fmt.Errorf("worker %d panicked, but recovered, in running process: %v\n%s", w.id, r, buf)
		}
	}()

	jobID := fetchJobID(w.id, work.Source().BucketName, work.Name())
	err := work.Do(w.ctx, jobID)
	if err != nil {
		w.errChan <- fmt.Errorf("worker %d encountered an error : %w", w.id, err)
	}
}

func (w *worker) Start() {
	w.wg.Add(1)
	go func() {
		for {
			w.readyPool <- w.job // worker ready for job
			select {
			case job := <-w.job: // worker waiting for new job
				w.Process(job)
			case <-w.quit:
				w.wg.Done()
				return
			}
		}
	}()
}

func (w *worker) Stop() {
	// tells worker to stop
	w.quit <- true
}

// fetchJobID returns a job id which is a combination of worker id, container name and blob name
func fetchJobID(workerId int, containerName string, blobName string) string {
	jobID := fmt.Sprintf("%s-%s-worker-%d", containerName, blobName, workerId)

	return jobID
}
