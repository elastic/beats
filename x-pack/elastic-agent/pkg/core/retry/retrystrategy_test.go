// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/backoff"
)

func TestRetry(t *testing.T) {
	type testCase struct {
		Fn                 func(context.Context) error
		ExpectedExecutions int64
		IsErrExpected      bool
		Enabled            bool
		Exponential        bool
	}

	errFatal := errors.New("fatal")
	var executions int64

	testCases := map[string]testCase{
		"not-failing":        testCase{Fn: func(_ context.Context) error { executions++; return nil }, ExpectedExecutions: 1, Enabled: true},
		"failing":            testCase{Fn: func(_ context.Context) error { executions++; return errors.New("fail") }, ExpectedExecutions: 4, IsErrExpected: true, Enabled: true},
		"fatal-by-enum":      testCase{Fn: func(_ context.Context) error { executions++; return errFatal }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: true},
		"fatal-by-iface":     testCase{Fn: func(_ context.Context) error { executions++; return ErrFatal{} }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: true},
		"not-fatal-by-iface": testCase{Fn: func(_ context.Context) error { executions++; return ErrNotFatal{} }, ExpectedExecutions: 4, IsErrExpected: true, Enabled: true},

		"dis-not-failing":        testCase{Fn: func(_ context.Context) error { executions++; return nil }, ExpectedExecutions: 1, Enabled: false},
		"dis-failing":            testCase{Fn: func(_ context.Context) error { executions++; return errors.New("fail") }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: false},
		"dis-fatal-by-enum":      testCase{Fn: func(_ context.Context) error { executions++; return errFatal }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: false},
		"dis-fatal-by-iface":     testCase{Fn: func(_ context.Context) error { executions++; return ErrFatal{} }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: false},
		"dis-not-fatal-by-iface": testCase{Fn: func(_ context.Context) error { executions++; return ErrNotFatal{} }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: false},

		"failing-exp": testCase{Fn: func(_ context.Context) error { executions++; return errors.New("fail") }, ExpectedExecutions: 4, IsErrExpected: true, Enabled: true, Exponential: true},
	}

	config := &Config{
		RetriesCount: 3,
		Delay:        500 * time.Millisecond,
	}

	for n, tc := range testCases {
		testFn := tc.Fn
		executions = 0
		config.Enabled = tc.Enabled
		config.Exponential = tc.Exponential

		startTime := time.Now()
		err := Do(context.Background(), config, testFn, errFatal)

		executionTime := time.Since(startTime)
		minExecutionTime := getMinExecutionTime(config.Delay, tc.ExpectedExecutions, tc.Exponential)
		maxExecutionTime := getMaxExecutionTime(config.Delay, tc.ExpectedExecutions, tc.Exponential)
		if tc.ExpectedExecutions > 1 && (executionTime < minExecutionTime || executionTime > maxExecutionTime) {
			t.Fatalf("[%s]: expecting execution time between %d-%d ns, got: %v", n, minExecutionTime, maxExecutionTime, executionTime)
		}

		if (err == nil) == tc.IsErrExpected {
			t.Fatalf("[%s]: expecting error, got: %v", n, err)
		}

		if executions != tc.ExpectedExecutions {
			t.Fatalf("[%s]: expecting %d executions, got: %d", n, tc.ExpectedExecutions, executions)
		}
	}
}

func TestRetryWithBackoff(t *testing.T) {
	type testCase struct {
		Fn                 func() error
		ExpectedExecutions int
		IsErrExpected      bool
		Enabled            bool
	}

	errFatal := errors.New("fatal")
	executions := 0

	testCases := map[string]testCase{
		"not-failing":        testCase{Fn: func() error { executions++; return nil }, ExpectedExecutions: 1, Enabled: true},
		"failing":            testCase{Fn: func() error { executions++; return errors.New("fail") }, ExpectedExecutions: 4, IsErrExpected: true, Enabled: true},
		"fatal-by-enum":      testCase{Fn: func() error { executions++; return errFatal }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: true},
		"fatal-by-iface":     testCase{Fn: func() error { executions++; return ErrFatal{} }, ExpectedExecutions: 1, IsErrExpected: true, Enabled: true},
		"not-fatal-by-iface": testCase{Fn: func() error { executions++; return ErrNotFatal{} }, ExpectedExecutions: 4, IsErrExpected: true, Enabled: true},
	}

	config := &Config{
		RetriesCount: 3,
		Delay:        5000,
	}
	maxDelay := time.Duration(config.Delay) * time.Millisecond

	done := make(chan struct{})
	maxWaitTime := 200 * time.Millisecond
	minWaitTime := 50 * time.Millisecond
	backoff := backoff.NewEqualJitterBackoff(done, minWaitTime, maxWaitTime)

	for n, tc := range testCases {
		testFn := tc.Fn
		executions = 0
		config.Enabled = tc.Enabled

		startTime := time.Now()
		err := DoWithBackoff(config, backoff, testFn, errFatal)

		executionTime := time.Since(startTime)
		minExecTime := getBackoffMinTime(minWaitTime, maxWaitTime, tc.ExpectedExecutions)
		if tc.ExpectedExecutions > 1 && (executionTime < minExecTime || executionTime > maxDelay) {
			t.Fatalf("[%s]: expecting execution time between %d-%d ns, got: %v", n, minExecTime, maxDelay, executionTime)
		}

		if (err == nil) == tc.IsErrExpected {
			t.Fatalf("[%s]: expecting error, got: %v", n, err)
		}

		if executions != tc.ExpectedExecutions {
			t.Fatalf("[%s]: expecting %d executions, got: %d", n, tc.ExpectedExecutions, executions)
		}
	}
}

type ErrFatal struct{ error }

func (ErrFatal) Fatal() bool {
	return true
}

type ErrNotFatal struct{ error }

func (ErrNotFatal) Fatal() bool {
	return false
}

func getMaxExecutionTime(delayDuration time.Duration, executions int64, exponential bool) time.Duration {
	delay := delayDuration.Nanoseconds()
	execTime := (executions)*delay + (delay / 2)
	if exponential {
		execTime = 0
		for i := int64(0); i < executions; i++ {
			execTime += i * delay
		}
		execTime += (delay / 2) * executions
	}

	return time.Duration(execTime)
}

func getMinExecutionTime(delayDuration time.Duration, executions int64, exponential bool) time.Duration {
	delay := delayDuration.Nanoseconds()
	execTime := (executions-1)*delay - (delay / 2)
	if exponential {
		execTime = 0
		for i := int64(0); i < executions; i++ {
			execTime += i * delay
		}
		execTime -= (delay / 2)
	}

	if execTime < 0 {
		execTime = 0
	}
	return time.Duration(execTime)
}

func getBackoffMinTime(delay time.Duration, maxWaitTime time.Duration, executions int) time.Duration {
	var duration time.Duration
	for i := 1; i < executions; i++ {
		duration += delay
		delay *= 2
		if delay > maxWaitTime {
			delay = maxWaitTime
		}
	}

	return duration
}
