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

package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"

	b "github.com/elastic/beats/libbeat/common/backoff"
	"github.com/elastic/beats/libbeat/publisher"
)

type backoffClient struct {
	client *client

	reason failReason

	done    chan struct{}
	backoff b.Backoff
}

// failReason is used to track the cause of an error.
// The redis client forces us to reconnect on any error (even for redis
// internal errors). The backoff timer must not be reset on a successful
// reconnect after publishing failed with a redis internal
// error (e.g. OutOfMemory), so we can still guarantee the backoff duration
// increases exponentially.
type failReason uint8

const (
	failNone failReason = iota
	failRedis
	failOther
)

func newBackoffClient(client *client, init, max time.Duration) *backoffClient {
	done := make(chan struct{})
	backoff := b.NewEqualJitterBackoff(done, init, max)
	return &backoffClient{
		client:  client,
		done:    done,
		backoff: backoff,
	}
}

func (b *backoffClient) Connect() error {
	err := b.client.Connect()
	if err != nil {
		// give the client a chance to promote an internal error to a network error.
		b.updateFailReason(err)
		b.backoff.Wait()
	} else if b.reason != failRedis { // Only reset backoff duration if failure was due to IO errors.
		b.resetFail()
	}

	return err
}

func (b *backoffClient) Close() error {
	err := b.client.Close()
	close(b.done)
	return err
}

func (b *backoffClient) Publish(batch publisher.Batch) error {
	err := b.client.Publish(batch)
	if err != nil {
		b.client.Close()
		b.updateFailReason(err)
		b.backoff.Wait()
	} else {
		b.resetFail()
	}
	return err
}

func (b *backoffClient) updateFailReason(err error) {
	if b.reason == failRedis {
		// we only allow 'Publish' to recover from an redis internal error
		return
	}

	if err == nil {
		b.reason = failNone
		return
	}

	if _, ok := err.(redis.Error); ok {
		b.reason = failRedis
	} else {
		b.reason = failOther
	}
}

func (b *backoffClient) resetFail() {
	b.reason = failNone
	b.backoff.Reset()
}

func (b *backoffClient) String() string {
	return b.client.String()
}
