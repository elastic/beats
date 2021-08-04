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

package input_logfile

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testScheduledOp struct {
	key  string
	exec func(n uint)
}

func (t *testScheduledOp) Key() string { return t.key }
func (t *testScheduledOp) Execute(_ *store, n uint) {
	if t.exec != nil {
		t.exec(n)
	}
}

func TestUpdateWriter(t *testing.T) {
	t.Run("single op is executed", func(t *testing.T) {
		ch := newUpdateChan()
		w := newUpdateWriter(nil, ch)
		defer w.Close()

		var wg sync.WaitGroup
		wg.Add(1)

		ch.Send(scheduledUpdate{
			op: &testScheduledOp{
				key:  "test",
				exec: func(n uint) { wg.Add(-int(n)) },
			},
			n: 1,
		})

		wg.Wait()
	})

	t.Run("multiple ops sum for single key", func(t *testing.T) {
		// small stress test to check that all ACKs will be handled or combined

		const N = 100

		ch := newUpdateChan()
		w := newUpdateWriter(nil, ch)
		defer w.Close()

		var wg sync.WaitGroup
		wg.Add(N)

		for i := 0; i < N; i++ {
			ch.Send(scheduledUpdate{
				op: &testScheduledOp{
					key: "test",
					exec: func(n uint) {
						t.Logf("ACK %v events", n)
						wg.Add(-int(n))
					},
				},
				n: 1,
			})
			runtime.Gosched()
		}
		wg.Wait()
	})
}

func TestUpdateChan_SendRecv(t *testing.T) {
	t.Run("read does not block if events are available", func(t *testing.T) {
		ch := newUpdateChan()

		op := makeTestUpdateOp("test")
		ch.Send(scheduledUpdate{op: op, n: 5})

		got, err := ch.Recv(context.TODO())
		require.NoError(t, err)
		assert.Equal(t, []scheduledUpdate{{op: op, n: 5}}, got)
	})

	t.Run("cancel read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()

		ch := newUpdateChan()
		_, err := ch.Recv(ctx)
		assert.Equal(t, ctx.Err(), err)
	})

	t.Run("wait for send", func(t *testing.T) {
		ch := newUpdateChan()

		op := makeTestUpdateOp("test")
		go func() {
			time.Sleep(100 * time.Millisecond)
			ch.Send(scheduledUpdate{op: op, n: 5})
		}()

		got, err := ch.Recv(context.TODO())
		require.NoError(t, err)
		assert.Equal(t, []scheduledUpdate{{op: op, n: 5}}, got)
	})
}

func TestUpdateChan_TryRecv(t *testing.T) {
	t.Run("return empty list if channel is empty", func(t *testing.T) {
		ch := newUpdateChan()
		assert.Empty(t, ch.TryRecv())
	})

	t.Run("read update", func(t *testing.T) {
		ch := newUpdateChan()

		op := makeTestUpdateOp("test")
		ch.Send(scheduledUpdate{op: op, n: 5})

		got := ch.TryRecv()
		assert.Equal(t, []scheduledUpdate{{op: op, n: 5}}, got)
	})

	t.Run("reading updates consumes all pending update", func(t *testing.T) {
		ch := newUpdateChan()

		op := makeTestUpdateOp("test")
		ch.Send(scheduledUpdate{op: op, n: 5})

		assert.NotEmpty(t, ch.TryRecv())
		assert.Empty(t, ch.TryRecv())
	})
}

func makeTestUpdateOp(key string) scheduledOp {
	return &testScheduledOp{key: key}
}
