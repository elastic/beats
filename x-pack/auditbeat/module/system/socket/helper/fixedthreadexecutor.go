// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package helper

import (
	"runtime"

	"golang.org/x/sys/unix"
)

// Task is a function that returns an arbitrary result and/or an error.
type Task func() (interface{}, error)

// TaskResult encapsulates the results of a Task.
type TaskResult struct {
	Data interface{}
	Err  error
}

// FixedThreadExecutor runs tasks on a fixed OS thread (see runtime.LockOSThread).
type FixedThreadExecutor struct {
	// TID is the OS identifier for the thread where it is running.
	TID int

	runC    chan Task
	resultC chan TaskResult
}

// Run submits new tasks to run on the executor.
func (ex FixedThreadExecutor) Run(task Task) {
	ex.runC <- task
}

// C returns the channel to read the results of tasks. This channel is closed
// when the executor terminates.
func (ex FixedThreadExecutor) C() <-chan TaskResult {
	return ex.resultC
}

// Close terminates the executor. Pending tasks will still be run.
func (ex FixedThreadExecutor) Close() {
	close(ex.runC)
}

// NewFixedThreadExecutor returns a new executor. queueSize sets the capacity
// for the channels. This limits how many tasks can be submitted without
// blocking and also allows the executor to terminate without the result channel
// to be consumed.
func NewFixedThreadExecutor(queueSize int) FixedThreadExecutor {
	ex := FixedThreadExecutor{
		runC:    make(chan Task, queueSize),
		resultC: make(chan TaskResult, queueSize),
	}

	go func() {
		defer close(ex.resultC)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		for task := range ex.runC {
			result, err := task()
			ex.resultC <- TaskResult{Data: result, Err: err}
		}
	}()

	ex.Run(func() (interface{}, error) { return unix.Gettid(), nil })
	res := <-ex.resultC
	ex.TID = res.Data.(int)
	return ex
}
