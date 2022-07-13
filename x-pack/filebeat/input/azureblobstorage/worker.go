// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

type Worker interface {
	Process(work Job)
	Start()
	Stop()
}

type worker struct {
	id        int
	ctx       context.Context
	wg        *sync.WaitGroup
	errChan   chan<- error
	job       chan Job
	readyPool chan chan Job
	quit      chan bool
}

func NewWorker(ctx context.Context, id int, readyPool chan chan Job, wg *sync.WaitGroup, errChan chan<- error) Worker {
	return &worker{
		id:        id,
		ctx:       ctx,
		wg:        wg,
		errChan:   errChan,
		readyPool: readyPool,
		job:       make(chan Job),
		quit:      make(chan bool),
	}
}

func (w *worker) Process(work Job) {
	// do the work
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			w.errChan <- fmt.Errorf("worker %d panicked, but recovered, in running process: %v\n%s\n", w.id, r, buf)
		}
	}()

	jobID := fetchJobID(w.id, work.Source().containerName, work.Name())
	fmt.Printf("JOB WITH ID %v and timeStamp %v EXECUTED\n", jobID, work.Timestamp().String())
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
