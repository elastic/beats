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

package discard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

type countingObserver struct {
	newBatches  int
	ackedEvents int
}

func (o *countingObserver) NewBatch(events int)             { o.newBatches += events }
func (o *countingObserver) AckedEvents(events int)          { o.ackedEvents += events }
func (*countingObserver) RetryableErrors(int)               {}
func (*countingObserver) PermanentErrors(int)               {}
func (*countingObserver) DuplicateEvents(int)               {}
func (*countingObserver) DeadLetterEvents(int)              {}
func (*countingObserver) ErrTooMany(int)                    {}
func (*countingObserver) FailureStoreEvents(int)            {}
func (*countingObserver) BatchSplit()                       {}
func (*countingObserver) WriteError(error)                  {}
func (*countingObserver) WriteBytes(int)                    {}
func (*countingObserver) ReadError(error)                   {}
func (*countingObserver) ReadBytes(int)                     {}
func (*countingObserver) ReportLatency(time.Duration)       {}

func TestPublishACKsAndReportsObserver(t *testing.T) {
	observer := &countingObserver{}
	out := &discardOutput{observer: observer}
	batch := outest.NewBatch(beat.Event{}, beat.Event{}, beat.Event{})

	err := out.Publish(context.Background(), batch)
	require.NoError(t, err)
	require.Len(t, batch.Signals, 1)
	assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
	assert.Equal(t, 3, observer.newBatches)
	assert.Equal(t, 3, observer.ackedEvents)
}

func TestMakeDiscardRejectsUnknownQueueType(t *testing.T) {
	cfg := conf.MustNewConfigFrom(mapstr.M{
		"queue": mapstr.M{
			"unknown": mapstr.M{
				"enabled": true,
			},
		},
	})

	_, err := makeDiscard(nil, beat.Info{Logger: logptest.NewTestingLogger(t, "")}, outputs.NewNilObserver(), cfg, paths.New())
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown queue type: unknown")
}

func TestMakeDiscardDisablesBulkMaxSize(t *testing.T) {
	cfg := conf.MustNewConfigFrom(mapstr.M{})
	grp, err := makeDiscard(nil, beat.Info{Logger: logptest.NewTestingLogger(t, "")}, outputs.NewNilObserver(), cfg, paths.New())
	require.NoError(t, err)
	require.Len(t, grp.Clients, 1)
	assert.Equal(t, -1, grp.BatchSize)
	assert.Equal(t, 0, grp.Retry)

	var unpacked struct {
		BulkMaxSize int `config:"bulk_max_size"`
	}
	require.NoError(t, cfg.Unpack(&unpacked))
	assert.Equal(t, -1, unpacked.BulkMaxSize)
}
