// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestRetryRun(t *testing.T) {
	logp.Configure(logp.Config{
		Level:     logp.DebugLevel,
		ToStderr:  true,
		Selectors: []string{"*"},
	})

	log := logp.NewLogger("retry_test").With("context", "osquery client connect")
	ctx := context.Background()

	type fields struct {
		maxRetry  int
		retryWait time.Duration
		log       *logp.Logger
	}

	type args struct {
		ctx context.Context
		fn  tryFunc
	}

	argsWithFunc := func(fn tryFunc) args {
		return args{
			ctx: ctx,
			fn:  fn,
		}
	}

	funcSucceedsOnNAttempt := func(attempt int) func(context.Context) error {
		curAttempt := 1
		return func(ctx context.Context) error {
			if curAttempt == attempt {
				return nil
			}
			curAttempt++
			return ErrAlreadyConnected
		}
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "no retries, no wait, success",
			fields: fields{
				log: log,
			},
			args: argsWithFunc(func(ctx context.Context) error {
				return nil
			}),
		},
		{
			name: "no retries, no wait, error",
			fields: fields{
				log: log,
			},
			args: argsWithFunc(func(ctx context.Context) error {
				return ErrAlreadyConnected
			}),
			wantErr: ErrAlreadyConnected,
		},
		{
			name: "retries, no wait, no more retries fails",
			fields: fields{
				maxRetry: 3,
				log:      log,
			},
			args:    argsWithFunc(funcSucceedsOnNAttempt(8)),
			wantErr: ErrAlreadyConnected,
		},
		{
			name: "retries, no wait, success",
			fields: fields{
				maxRetry: 3,
				log:      log,
			},
			args: argsWithFunc(funcSucceedsOnNAttempt(4)),
		},
		{
			name: "retries, with wait, success",
			fields: fields{
				maxRetry:  3,
				retryWait: 1 * time.Millisecond,
				log:       log,
			},
			args: argsWithFunc(funcSucceedsOnNAttempt(4)),
		},
		{
			name: "retries, with wait, success sooner",
			fields: fields{
				maxRetry:  3,
				retryWait: 1 * time.Millisecond,
				log:       log,
			},
			args: argsWithFunc(funcSucceedsOnNAttempt(2)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &retry{
				maxRetry:  tt.fields.maxRetry,
				retryWait: tt.fields.retryWait,
				log:       tt.fields.log,
			}
			err := r.Run(tt.args.ctx, tt.args.fn)
			if err != nil {
				if tt.wantErr != nil {
					diff := cmp.Diff(tt.wantErr, err, cmpopts.EquateErrors())
					if diff != "" {
						t.Error(diff)
					}
				} else {
					t.Errorf("got err: %v, wantErr: nil", err)
				}
			} else if tt.wantErr != nil {
				t.Errorf("got err: nil, wantErr: %v", tt.wantErr)
			}
		})
	}
}
