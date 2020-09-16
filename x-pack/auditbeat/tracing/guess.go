// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package tracing

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

// GuessContext shared with guesses.
type GuessContext struct {
	// Log is a logger so that guesses can log.
	Log Logger
	// Vars is the current set of template variables.
	Vars common.MapStr
	// Timeout is the maximum time allowed to wait for a guess to complete.
	Timeout time.Duration
}

// Guesser is the interface that must be fulfilled to perform guesses using
// kprobes.
type Guesser interface {
	// Name returned the name of this guess.
	Name() string
	// Probes returns one or more probes to install.
	Probes() ([]ProbeDef, error)
	// Provides must return the list of variables that this guess will provide.
	Provides() []string
	// Requires must return the list of variables that this guess requires to
	// be available before running.
	Requires() []string
	// Prepare performs initializations. Events won't be captured for actions
	// performed during preparation. It runs in the same OS thread than Trigger.
	Prepare(ctx GuessContext) error
	// Trigger performs the actions necessary to generate events of interest
	// to this guess.
	Trigger() error
	// Extract receives the events generated during trigger.
	// Done is false when it needs to be called with more events. True when
	// the guess has completed and results is a map with the discovered values.
	Extract(event interface{}) (result common.MapStr, done bool)
	// Terminate performs cleanup after the guess is complete.
	Terminate() error
}

// RepeatGuesser is a guess that needs to be repeated multiple times and the
// results consolidated into a single result.
type RepeatGuesser interface {
	Guesser
	// NumRepeats returns how many times the guess is repeated.
	NumRepeats() int
	// Reduce takes the output of every repetition and returns the final result.
	Reduce([]common.MapStr) (common.MapStr, error)
}

// EventualGuesser is a guess that repeats an undetermined amount of times
// until it succeeds. It is re-executed as long as its Extract method returns
// a nil result.
type EventualGuesser interface {
	Guesser
	// MaxRepeats is the maximum number of times to repeat.
	MaxRepeats() int
}

// ConditionalGuesser is a guess that might only need to run under certain
// conditions (i.e. when IPv6 is enabled).
type ConditionalGuesser interface {
	Guesser
	// Condition returns if this guess has to be run.
	// When false, it must set all its Provides() to dummy values to ensure that
	// dependent guesses are run.
	Condition(ctx GuessContext) (bool, error)
}

// guess is a helper function to easily determine memory layouts of kernel
// structs and similar tasks. It installs the guesser's Probe, starts a perf
// channel and executes the Trigger function. Each record received through the
// channel is passed to the Extract function. Terminates once Extract succeeds
// or the timeout expires.
func (e *Engine) guess(guesser Guesser, ctx GuessContext) (result common.MapStr, err error) {
	switch v := guesser.(type) {
	case RepeatGuesser:
		result, err = e.guessMultiple(v, ctx)
	case EventualGuesser:
		result, err = e.guessEventually(v, ctx)
	default:
		result, err = e.guessOnce(guesser, ctx)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "%s failed", guesser.Name())
	}
	return result, nil
}

func (e *Engine) guessMultiple(guess RepeatGuesser, ctx GuessContext) (result common.MapStr, err error) {
	var results []common.MapStr
	for idx := 1; idx <= guess.NumRepeats(); idx++ {
		r, err := e.guessOnce(guess, ctx)
		if err != nil {
			return nil, err
		}
		ctx.Log.Debugf(" --- result of %s run #%d: %+v", guess.Name(), idx, r)
		results = append(results, r)
	}
	return guess.Reduce(results)
}

func (e *Engine) guessEventually(guess EventualGuesser, ctx GuessContext) (result common.MapStr, err error) {
	limit := guess.MaxRepeats()
	for i := 0; i < limit; i++ {
		ctx.Log.Debugf(" --- %s run #%d", guess.Name(), i)
		if result, err = e.guessOnce(guess, ctx); err != nil {
			return nil, err
		}
		if len(result) != 0 {
			return result, nil
		}
	}
	return nil, fmt.Errorf("guess %s didn't succeed after %d tries", guess.Name(), limit)
}

func (e *Engine) guessOnce(guesser Guesser, ctx GuessContext) (result common.MapStr, err error) {
	if err := guesser.Prepare(ctx); err != nil {
		return nil, errors.Wrap(err, "prepare failed")
	}
	defer func() {
		if err := guesser.Terminate(); err != nil {
			ctx.Log.Errorf("Terminate failed: %v", err)
		}
	}()
	probes, err := guesser.Probes()
	if err != nil {
		return nil, errors.Wrap(err, "failed generating probes")
	}

	decoders := make([]Decoder, 0, len(probes))
	formats := make([]ProbeFormat, 0, len(probes))
	defer e.uninstallProbes()
	for _, pdesc := range probes {
		format, decoder, err := e.installProbe(pdesc)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add kprobe '%s'", pdesc.Probe.String())
		}
		formats = append(formats, format)
		decoders = append(decoders, decoder)
	}

	// Separate OS thread to run the guesses.Trigger() function.
	// - This thread is locked to a single CPU thread, so that perf channel
	//   can monitor only that thread ID.
	// - Running the trigger in a separate goroutine allows to timeout
	//   if trigger blocks.
	//
	// executorQueueSize>0 allows the executor to terminate and release the OS
	// thread even if the result of an execution is not consumed because be
	// timeout.
	const executorQueueSize = 1
	thread := NewFixedThreadExecutor(executorQueueSize)
	defer thread.Close()

	perfchan, err := NewPerfChannel(
		WithBufferSize(8),
		WithErrBufferSize(1),
		WithLostBufferSize(8),
		WithRingSizeExponent(2),
		WithTID(thread.TID),
		WithPollTimeout(time.Millisecond*10))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create perfchannel")
	}
	defer perfchan.Close()

	for i := range probes {
		if err := perfchan.MonitorProbe(formats[i], decoders[i]); err != nil {
			return nil, errors.Wrap(err, "failed to monitor probe")
		}
	}

	if err := perfchan.Run(); err != nil {
		return nil, errors.Wrap(err, "failed to run perf channel")
	}

	timer := time.NewTimer(ctx.Timeout)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	thread.Run(func() (interface{}, error) {
		return nil, guesser.Trigger()
	})

	// Blocks until trigger terminates or a timeout.
	// Need to make sure that the trigger has finished before extracting
	// results because there could be data-races between Trigger and Extract.
	select {
	case r := <-thread.C():
		if r.Err != nil {
			return nil, errors.Wrap(r.Err, "trigger execution failed")
		}
	case <-timer.C:
		return nil, errors.New("timeout while waiting for trigger to complete")
	}

	for {
		select {
		case <-timer.C:
			return nil, errors.New("timeout while waiting for event")

		case ev, ok := <-perfchan.C():
			if !ok {
				return nil, errors.New("perf channel closed unexpectedly")
			}
			if result, ok = guesser.Extract(ev); !ok {
				continue
			}
			return result, nil

		case err := <-perfchan.ErrC():
			if err != nil {
				return nil, errors.Wrap(err, "error received from perf channel")
			}

		case <-perfchan.LostC():
			return nil, errors.Wrap(err, "event loss in perf channel")
		}
	}
}

func containsAll(requirements []string, dict common.MapStr) bool {
	for _, req := range requirements {
		if _, found := dict[req]; !found {
			return false
		}
	}
	return true
}

// GuessAll will run all the registered guesses, taking care of doing so in an
// order so that a probe dependencies are available before it runs.
func (e *Engine) guessAll(list []Guesser, ctx GuessContext) (err error) {
	start := time.Now()
	ctx.Log.Infof("Running %d guesses ...", len(list))
	// This simple O(N^2) topological sort is enough for the small
	// number of guesses
	for len(list) > 0 {
		var next []Guesser
		for _, guesser := range list {
			if cond, isCond := guesser.(ConditionalGuesser); isCond {
				mustRun, err := cond.Condition(ctx)
				if err != nil {
					return errors.Wrapf(err, "condition failed for %s", cond.Name())
				}
				if !mustRun {
					ctx.Log.Debugf("Guess %s skipped.", cond.Name())
					continue
				}
			}
			if !containsAll(guesser.Requires(), ctx.Vars) {
				next = append(next, guesser)
				continue
			}
			result, err := e.guess(guesser, ctx)
			if err != nil {
				return err
			}
			if !containsAll(guesser.Provides(), result) {
				ctx.Log.Errorf("Guesser '%s' promised %+v but provided %+v", guesser.Name(), guesser.Provides(), result)
				return errors.New("guesser did not provide all promised variables")
			}
			ctx.Vars.Update(result)
			ctx.Log.Debugf("Guess %s completed: %v", guesser.Name(), result)
		}
		if len(next) == len(list) {
			ctx.Log.Warnf("Internal error: No guess can be run (%d pending):", len(list))
			for _, guess := range list {
				requires := guess.Requires()
				var missing []string
				for _, req := range requires {
					if _, found := ctx.Vars[req]; !found {
						missing = append(missing, req)
					}
				}
				ctx.Log.Warnf("   guess '%s' requires missing vars: %v", guess.Name(), missing)
			}
			return errors.New("no guess can be run")
		}
		list = next
	}
	ctx.Log.Infof("Guessing done after %v", time.Since(start))
	return nil
}
