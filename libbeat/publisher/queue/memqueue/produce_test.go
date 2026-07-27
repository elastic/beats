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

package memqueue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestPublishReusesResponseChannel(t *testing.T) {
	q := NewQueue[int](logp.NewNopLogger(), nil, Settings{
		Events:        64,
		MaxGetRequest: 64,
		FlushTimeout:  time.Millisecond,
	}, 0, nil)
	defer q.Close(true)

	t.Run("forgetful", func(t *testing.T) {
		p := q.Producer(queue.ProducerConfig{})
		defer p.Close()
		fp, ok := p.(*forgetfulProducer[int])
		require.True(t, ok, "producer should be of type forgetfulProducer")
		r1 := fp.makePushRequest(1)
		r2 := fp.makePushRequest(2)
		require.Equal(t, r1.resp, r2.resp,
			"forgetfulProducer should reuse the same response channel across requests")
	})

	t.Run("ack", func(t *testing.T) {
		p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})
		defer p.Close()
		ap, ok := p.(*ackProducer[int])
		require.True(t, ok, "producer should be of type ackProducer")
		r1 := ap.makePushRequest(1)
		r2 := ap.makePushRequest(2)
		require.Equal(t, r1.resp, r2.resp,
			"ackProducer should reuse the same response channel across requests")
	})
}

func TestPublishSequentialEntryIDs(t *testing.T) {
	const n = 1000

	q := NewQueue[int](logp.NewNopLogger(), nil, Settings{
		Events:        256,
		MaxGetRequest: 256,
		FlushTimeout:  time.Millisecond,
	}, 0, nil)
	defer q.Close(true)

	ctx := t.Context()
	go func() {
		for ctx.Err() == nil {
			batch, err := q.Get(64)
			if err != nil {
				return
			}
			batch.Done()
		}
	}()

	for _, tc := range []struct {
		name string
		cfg  queue.ProducerConfig
	}{
		{"forgetful", queue.ProducerConfig{}},
		{"ack", queue.ProducerConfig{ACK: func(int) {}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := q.Producer(tc.cfg)
			defer p.Close()

			var prev queue.EntryID
			for i := range n {
				id, ok := p.Publish(i)
				require.True(t, ok, "publish %d must succeed", i)
				if i > 0 {
					require.Greater(t, id, prev,
						"EntryID must increase monotonically (publish %d)", i)
				}
				prev = id
			}
		})
	}
}
