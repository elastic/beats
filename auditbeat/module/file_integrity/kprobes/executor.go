package kprobes

import (
	"context"
	"golang.org/x/sys/unix"
	"runtime"
)

type executor interface {
	Run(f func() error) error
	GetTID() int
}

// fixedExecutor runs tasks on a fixed OS thread (see runtime.LockOSThread).
type fixedExecutor struct {
	ctx      context.Context
	cancelFn context.CancelFunc
	// tid is the OS identifier for the thread where it is running.
	tid  int
	runC chan func() error
	retC chan error
}

// Run submits new tasks to run on the executor and waits for them to finish returning any error.
func (ex *fixedExecutor) Run(f func() error) error {
	select {
	case ex.runC <- f:
	case <-ex.ctx.Done():
		return ex.ctx.Err()
	}

	select {
	case <-ex.ctx.Done():
		return ex.ctx.Err()
	case err := <-ex.retC:
		return err
	}
}

// RunAsync submits new tasks to run on the executor but doesn't wait for them to finish. It is the
// responsibility of the caller to call Wait and essentially wait for the tasks to finish. Note that
// missing doing the former will result in blocking future invocations of RunAsync and Run.
func (ex *fixedExecutor) RunAsync(f func() error) error {
	select {
	case ex.runC <- f:
	case <-ex.ctx.Done():
		return ex.ctx.Err()
	}

	return nil
}

// GetTID returns the OS identifier for the thread where executor goroutine is locked against.
func (ex *fixedExecutor) GetTID() int {
	return ex.tid
}

// Wait returns the channel to read the results of tasks. This channel is closed
// when the executor terminates.
func (ex *fixedExecutor) Wait() <-chan error {
	return ex.retC
}

// Close terminates the executor. Pending tasks will still be run.
func (ex *fixedExecutor) Close() {
	ex.cancelFn()
	close(ex.runC)
}

// newFixedThreadExecutor returns a new fixedExecutor.
func newFixedThreadExecutor(ctx context.Context) *fixedExecutor {

	mCtx, cancelFn := context.WithCancel(ctx)

	ex := &fixedExecutor{
		ctx:      mCtx,
		cancelFn: cancelFn,
		runC:     make(chan func() error, 1),
		retC:     make(chan error, 1),
	}

	tidC := make(chan int)

	go func() {
		defer close(ex.retC)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		select {
		case <-ctx.Done():
			return
		case tidC <- unix.Gettid():
			close(tidC)
		}

		for {
			select {
			case runF, ok := <-ex.runC:
				if !ok {
					// channel closed
					return
				}

				select {
				case ex.retC <- runF():
				case <-ex.ctx.Done():
					return
				}

			case <-ex.ctx.Done():
				return
			}
		}
	}()

	select {
	case ex.tid = <-tidC:
	case <-ctx.Done():
		return nil
	}

	return ex
}
