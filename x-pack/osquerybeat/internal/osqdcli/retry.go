// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

import (
	"context"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type retry struct {
	maxRetry  int
	retryWait time.Duration
	log       *logp.Logger
}

type tryFunc func(ctx context.Context) error

func (r *retry) Run(ctx context.Context, fn tryFunc) (err error) {
	maxAttempts := r.maxRetry + 1
	for i := 0; i < maxAttempts; i++ {
		attempt := i + 1
		r.log.Debugf("attempt %v out of %v", attempt, maxAttempts)

		err = fn(ctx)

		if err != nil {
			r.log.Debugf("attempt %v out of %v failed, err: %v", attempt, maxAttempts, err)
			if i != maxAttempts {
				if r.retryWait > 0 {
					r.log.Debugf("wait for %v before next retry", r.retryWait)
					err = waitWithContext(ctx, retryWait)
					if err != nil {
						r.log.Debugf("wait returned err: %v", err)
						return err
					}
				}
			} else {
				r.log.Debugf("no more attempts, return err: %v", err)
				return err
			}
		} else {
			r.log.Debugf("attempt %v out of %v succeeded", attempt, maxAttempts)
			return nil
		}
	}
	return err
}

func waitWithContext(ctx context.Context, to time.Duration) error {
	t := time.NewTimer(to)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
	}
	return nil
}
