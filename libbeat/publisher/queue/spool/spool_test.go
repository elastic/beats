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

package spool

import (
	"flag"
	"math/rand"
	"testing"
	"time"

	humanize "github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/txfiletest"
)

var seed int64

type testQueue struct {
	*diskSpool
	teardown func()
}

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
}

func TestProduceConsumer(t *testing.T) {
	maxEvents := 4096
	minEvents := 32

	rand.Seed(seed)
	events := rand.Intn(maxEvents-minEvents) + minEvents
	batchSize := rand.Intn(events-8) + 4

	t.Log("seed: ", seed)
	t.Log("events: ", events)
	t.Log("batchSize: ", batchSize)

	testWith := func(factory queuetest.QueueFactory) func(t *testing.T) {
		return func(test *testing.T) {
			t.Run("single", func(t *testing.T) {
				queuetest.TestSingleProducerConsumer(t, events, batchSize, factory)
			})
			t.Run("multi", func(t *testing.T) {
				queuetest.TestMultiProducerConsumer(t, events, batchSize, factory)
			})
		}
	}

	testWith(makeTestQueue(
		128*humanize.KiByte, 4*humanize.KiByte, 16*humanize.KiByte,
		100*time.Millisecond,
	))(t)
}

func makeTestQueue(
	maxSize, pageSize, writeBuffer uint,
	flushTimeout time.Duration,
) func(*testing.T) queue.Queue {
	logger := defaultLogger()
	return func(t *testing.T) queue.Queue {
		ok := false
		path, cleanPath := txfiletest.SetupPath(t, "")
		defer func() {
			if !ok {
				cleanPath()
			}
		}()

		spool, err := newDiskSpool(logger, path, settings{
			WriteBuffer:       writeBuffer,
			WriteFlushTimeout: flushTimeout,
			Codec:             codecCBORL,
			File: txfile.Options{
				MaxSize:  uint64(maxSize),
				PageSize: uint32(pageSize),
				Prealloc: true,
				Readonly: false,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		tq := &testQueue{diskSpool: spool, teardown: cleanPath}
		return tq
	}
}

func (t *testQueue) Close() error {
	err := t.diskSpool.Close()
	t.teardown()
	return err
}
