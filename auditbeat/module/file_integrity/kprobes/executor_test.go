// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package kprobes

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_executor(t *testing.T) {
	// parent context is cancelled at creation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	exec := newFixedThreadExecutor(ctx)
	require.Nil(t, exec)

	// parent context is cancelled
	ctx, cancel = context.WithCancel(context.Background())
	exec = newFixedThreadExecutor(ctx)
	require.NotNil(t, exec)

	err := exec.Run(func() error {
		cancel()
		time.Sleep(10 * time.Second)
		return nil
	})
	require.ErrorIs(t, err, ctx.Err())
	require.ErrorIs(t, exec.Run(func() error {
		return nil
	}), ctx.Err())

	// executor is closed while running cancelled
	exec = newFixedThreadExecutor(context.Background())
	require.NotNil(t, exec)

	err = exec.Run(func() error {
		exec.Close()
		time.Sleep(10 * time.Second)
		return nil
	})
	require.ErrorIs(t, err, exec.ctx.Err())

	// normal exec no error
	exec = newFixedThreadExecutor(context.Background())
	require.NotNil(t, exec)

	err = exec.Run(func() error {
		time.Sleep(1 * time.Second)
		return nil
	})
	require.NoError(t, err)
	exec.Close()

	// exec with error
	exec = newFixedThreadExecutor(context.Background())
	require.NotNil(t, exec)
	retErr := errors.New("test error")

	err = exec.Run(func() error {
		return retErr
	})
	require.ErrorIs(t, err, retErr)
	exec.Close()

	// check that runs are indeed sequential
	// as pathTraverser depends on it
	err = nil
	atomicInt := uint32(0)
	atomicCheck := func() error {
		swapped := atomic.CompareAndSwapUint32(&atomicInt, 0, 1)
		if !swapped {
			return errors.New("parallel runs")
		}
		time.Sleep(1 * time.Second)
		swapped = atomic.CompareAndSwapUint32(&atomicInt, 1, 0)
		if !swapped {
			return errors.New("parallel runs")
		}
		return nil
	}
	exec = newFixedThreadExecutor(context.Background())
	require.NotNil(t, exec)
	errChannel := make(chan error, 1)
	wg := sync.WaitGroup{}
	start := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if runErr := exec.Run(atomicCheck); runErr != nil {
				select {
				case errChannel <- runErr:
				default:
				}
			}
		}()
	}
	time.Sleep(1 * time.Second)
	close(start)
	wg.Wait()
	select {
	case err = <-errChannel:
	default:

	}
	require.Nil(t, err)
}
