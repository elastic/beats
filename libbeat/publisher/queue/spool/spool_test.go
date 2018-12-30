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
	"fmt"
	"math/rand"
	"testing"
	"time"

	humanize "github.com/dustin/go-humanize"

	"github.com/elastic/beats/libbeat/publisher/queue"
	"github.com/elastic/beats/libbeat/publisher/queue/queuetest"
	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/txfiletest"
)

var seed int64
var debug bool

type testQueue struct {
	*Spool
	teardown func()
}

type testLogger struct {
	t *testing.T
}

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
	flag.BoolVar(&debug, "noisy", false, "print test logs to console")
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
	return func(t *testing.T) queue.Queue {
		if debug {
			fmt.Println("Test:", t.Name())
		}

		ok := false
		path, cleanPath := txfiletest.SetupPath(t, "")
		defer func() {
			if !ok {
				cleanPath()
			}
		}()

		spool, err := NewSpool(&testLogger{t}, path, Settings{
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

		tq := &testQueue{Spool: spool, teardown: cleanPath}
		return tq
	}
}

func (t *testQueue) Close() error {
	err := t.Spool.Close()
	t.teardown()
	return err
}

func (l *testLogger) Debug(vs ...interface{})              { l.report("Debug", vs) }
func (l *testLogger) Debugf(fmt string, vs ...interface{}) { l.reportf("Debug: ", fmt, vs) }

func (l *testLogger) Info(vs ...interface{})              { l.report("Info", vs) }
func (l *testLogger) Infof(fmt string, vs ...interface{}) { l.reportf("Info", fmt, vs) }

func (l *testLogger) Error(vs ...interface{})              { l.report("Error", vs) }
func (l *testLogger) Errorf(fmt string, vs ...interface{}) { l.reportf("Error", fmt, vs) }

func (l *testLogger) report(level string, vs []interface{}) {
	args := append([]interface{}{level, ":"}, vs...)
	l.t.Log(args...)
	if debug {
		fmt.Println(args...)
	}
}

func (l *testLogger) reportf(level string, str string, vs []interface{}) {
	str = level + ": " + str
	l.t.Logf(str, vs...)
	if debug {
		fmt.Printf(str, vs...)
	}
}
