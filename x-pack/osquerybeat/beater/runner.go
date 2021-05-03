// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"sync"
	"time"
)

type runner struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (r *runner) stop() {
	r.cancel()
	r.wg.Wait()
}

func startRunner(pctx context.Context, q interface{}, interval time.Duration, query func(context.Context, interface{}) error) *runner {
	ctx, cancel := context.WithCancel(pctx)
	r := &runner{
		cancel: cancel,
	}

	r.wg.Add(1)
	go func() {
		defer cancel()
		defer r.wg.Done()

		// Run query right away
		query(ctx, q)

		if interval == 0 {
			return
		}

		// Schedule with interval
		t := time.NewTimer(interval)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				query(ctx, q)
				t.Reset(interval)
			case <-ctx.Done():
				return
			}
		}
	}()

	return r
}
