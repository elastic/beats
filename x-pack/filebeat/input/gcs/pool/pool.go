// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pool

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/job"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Pool is a dynamic & configurable worker pool that performs work distribution similar to a job pool.
// The number of workers a pool can have is configured by the user and based on that number and available free workers,
// jobs are distributed. The workers if free send a signal to the ready pool saying "I'm free to do some work".
// They do this by making their job channel available on the ready pool. When a new job is submitted to the job queue & a free worker is
// available in the ready pool, this job is then sent over the worker channel available in the ready pool.
// The worker on receiving a new job on it's job channel will now pick it up and execute it. Once the job is complete, the worker is free
// again and will make itself available on the ready pool.
type Pool interface {
	Start()
	Stop()
	Submit(job job.Job)
	AvailableWorkers() int
}

type pool struct {
	ctx           context.Context
	workerErrChan chan error
	jobQueue      chan job.Job
	readyPool     chan chan job.Job
	workers       []Worker
	dispatcherWg  sync.WaitGroup
	workersWg     *sync.WaitGroup
	quit          chan bool
	log           *logp.Logger
}

var poolError = "worker pool error : %w"

// NewWorkerPool returns an instance of a worker pool with 'maxWorkers' ready to accept work
func NewWorkerPool(ctx context.Context, maxWorkers int, log *logp.Logger) Pool {
	workersWg := sync.WaitGroup{}

	readyPool := make(chan chan job.Job, maxWorkers)
	workers := make([]Worker, maxWorkers)
	errChan := make(chan error)

	// creates workers
	for i := 0; i < maxWorkers; i++ {
		workers[i] = NewWorker(ctx, i+1, readyPool, &workersWg, errChan, log)
	}

	return &pool{
		ctx:           ctx,
		workerErrChan: errChan,
		jobQueue:      make(chan job.Job),
		readyPool:     readyPool,
		workers:       workers,
		dispatcherWg:  sync.WaitGroup{},
		workersWg:     &workersWg,
		quit:          make(chan bool),
		log:           log,
	}
}

// Start, starts the worker pool and initializes the workers
func (q *pool) Start() {
	// puts workers in ready state
	for i := 0; i < len(q.workers); i++ {
		q.workers[i].Start()
	}

	// starts dispatcher
	go q.dispatch()
}

// Submit, submits the job to the job queue
// This is a blocking if all workers are busy
func (q *pool) Submit(job job.Job) {
	q.jobQueue <- job
}

// AvailableWorkers returns the number of free workers at any point of time
func (q *pool) AvailableWorkers() int {
	return len(q.readyPool)
}

// Stop, gracefully stops the workers & frees the worker pool
func (q *pool) Stop() {
	q.quit <- true
	q.dispatcherWg.Wait()
}

func (q *pool) dispatch() {
	// starts dispatching jobs
	q.dispatcherWg.Add(1)
	for {
		select {

		case err := <-q.workerErrChan:
			q.log.Errorf(poolError, err)

		case <-q.ctx.Done():
			q.log.Errorf(poolError, q.ctx.Err())
			q.Stop()

		case job := <-q.jobQueue:
			workerXChannel := <-q.readyPool // free worker 'x' found
			workerXChannel <- job           // assigns job to worker 'x'

		case <-q.quit:
			// frees all workers & gracefully closes the worker pool
			for i := 0; i < len(q.workers); i++ {
				q.workers[i].Stop()
			}
			// waits for all workers to finish their jobs
			q.workersWg.Wait()
			// stops dispatcher
			q.dispatcherWg.Done()
			return
		}
	}
}
