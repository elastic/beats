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

package outputs

import (
	"context"
	"errors"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common/backoff"
	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/testing"
)

type backoffClient struct {
	client NetworkClient

	done    chan struct{}
	backoff backoff.Backoff
}

// WithBackoff wraps a NetworkClient, adding exponential backoff support to a network client if connection/publishing failed.
func WithBackoff(client NetworkClient, init, max time.Duration) NetworkClient {
	done := make(chan struct{})
	backoff := backoff.NewEqualJitterBackoff(done, init, max)
	return &backoffClient{
		client:  client,
		done:    done,
		backoff: backoff,
	}
}

func (b *backoffClient) Connect() error {
	err := b.client.Connect()
	backoff.WaitOnError(b.backoff, err)
	return err
}

func (b *backoffClient) Close() error {
	err := b.client.Close()
	close(b.done)
	return err
}

func (b *backoffClient) Publish(ctx context.Context, batch publisher.Batch) error {
	err := b.client.Publish(ctx, batch)
	if err != nil {
		b.client.Close()
	}
	backoff.WaitOnError(b.backoff, err)
	return err
}

func (b *backoffClient) Client() NetworkClient {
	return b.client
}

func (b *backoffClient) Test(d testing.Driver) {
	c, ok := b.client.(testing.Testable)
	if !ok {
		d.Fatal("output", errors.New("client doesn't support testing"))
	}

	c.Test(d)
}

func (b *backoffClient) String() string {
	return "backoff(" + b.client.String() + ")"
}
