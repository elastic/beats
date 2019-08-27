// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package guess

import (
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/x-pack/auditbeat/tracing"
)

// Context shared with guesses.
type Context struct {
	// Log is a logger so that guesses can log.
	Log helper.Logger
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
	Probes() ([]helper.ProbeDef, error)
	// Provides must return the list of variables that this guess will provide.
	Provides() []string
	// Requires must return the list of variables that this guess requires to
	// be available before running.
	Requires() []string
	// Prepare performs initializations. Events won't be captured for actions
	// performed during preparation. It runs in the same OS thread than Trigger.
	Prepare(ctx Context) error
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

// Guess is a helper function to easily determine memory layouts of kernel
// structs and similar tasks. It installs the guesser's Probe, starts a perf
// channel and executes the Trigger function. Each record received through the
// channel is passed to the Extract function. Terminates once Extract succeeds
// or the timeout expires.
func Guess(guesser Guesser, installer helper.ProbeInstaller, ctx Context) (result common.MapStr, err error) {
	switch v := guesser.(type) {
	case RepeatGuesser:
		result, err = guessMultiple(v, installer, ctx)
	case EventualGuesser:
		result, err = guessEventually(v, installer, ctx)
	default:
		result, err = guessOnce(guesser, installer, ctx)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "%s failed", guesser.Name())
	}
	return result, nil
}

func guessMultiple(guess RepeatGuesser, installer helper.ProbeInstaller, ctx Context) (result common.MapStr, err error) {
	var results []common.MapStr
	for idx := 1; idx <= guess.NumRepeats(); idx++ {
		r, err := guessOnce(guess, installer, ctx)
		if err != nil {
			return nil, err
		}
		ctx.Log.Debugf(" --- result of %s run #%d: %+v", guess.Name(), idx, r)
		results = append(results, r)
	}
	return guess.Reduce(results)
}

func guessEventually(guess EventualGuesser, installer helper.ProbeInstaller, ctx Context) (result common.MapStr, err error) {
	limit := guess.MaxRepeats()
	for i := 0; i < limit; i++ {
		ctx.Log.Debugf(" --- %s run #%d", guess.Name(), i)
		if result, err = guessOnce(guess, installer, ctx); err != nil {
			return nil, err
		}
		if len(result) != 0 {
			return result, nil
		}
	}
	return nil, fmt.Errorf("guess %s didn't succeed after %d tries", guess.Name(), limit)
}

func guessOnce(guesser Guesser, installer helper.ProbeInstaller, ctx Context) (result common.MapStr, err error) {
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

	decoders := make([]tracing.Decoder, 0, len(probes))
	formats := make([]tracing.ProbeFormat, 0, len(probes))
	defer installer.UninstallInstalled()
	for _, pdesc := range probes {
		format, decoder, err := installer.Install(pdesc)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add kprobe '%s'", pdesc.Probe.String())
		}
		formats = append(formats, format)
		decoders = append(decoders, decoder)
	}

	// channel to receive the TID and sync with the trigger goroutine.
	// Length is zero so that it blocks sender and receiver, making it useful
	// for synchronization.
	tidChan := make(chan int, 0)

	// Trigger goroutine.
	go func() {
		// Lock this goroutine to the current CPU thread. This way we can setup
		// the perf channel to receive events from this thread only, avoiding
		// the guess being contaminated with an externally-generated event.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(tidChan)

		// Send the thread ID
		tidChan <- syscall.Gettid()

		// This blocks until the main goroutine installs the perf channel
		tidChan <- 0

		if err := guesser.Trigger(); err != nil {
			ctx.Log.Errorf("Trigger failed (%s): %v", guesser.Name(), err)
		}
	}()

	// Receive the thread ID from the trigger goroutine.
	tid := <-tidChan
	defer func() {
		// reads the tidChan just in case we return due to an error below and
		// the trigger goroutine is blocking for the fire signal.
		// Otherwise this will be a read from a closed channel.
		select {
		case <-tidChan:
		default:
		}
	}()
	perfchan, err := tracing.NewPerfChannel(
		tracing.WithBufferSize(8),
		tracing.WithErrBufferSize(1),
		tracing.WithLostBufferSize(8),
		tracing.WithRingSizeExponent(2),
		tracing.WithTID(tid),
		tracing.WithPollTimeout(time.Millisecond*10))
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

	// Read the tidchan again to release the trigger.
	<-tidChan

	// This blocks until trigger terminates (channel closed) or a timeout
	select {
	case <-tidChan:
	case <-timer.C:
		return nil, errors.New("timeout while waiting for guess trigger to complete")
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
func GuessAll(installer helper.ProbeInstaller, ctx Context) (err error) {
	list := Registry.GetList()
	start := time.Now()
	ctx.Log.Infof("Running %d guesses ...", len(list))
	// This simple O(N^2) topological sort is enough for the small
	// number of guesses
	for len(list) > 0 {
		var next []Guesser
		for _, guesser := range list {
			if !containsAll(guesser.Requires(), ctx.Vars) {
				next = append(next, guesser)
				continue
			}
			result, err := Guess(guesser, installer, ctx)
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
			return errors.New("no guess can be run")
		}
		list = next
	}
	ctx.Log.Infof("Guessing done after %v", time.Since(start))
	return nil
}
