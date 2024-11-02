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

package monitors

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncPipelineWrapper(t *testing.T) {
	defineJob := func(fields mapstr.M) func(event *beat.Event) (j []jobs.Job, e error) {
		return func(event *beat.Event) (j []jobs.Job, e error) {
			eventext.MergeEventFields(event, fields)
			return nil, nil
		}
	}
	simpleJob := defineJob(mapstr.M{"foo": "bar"})

	testCases := []struct {
		name string
		job  jobs.Job
		acks int
	}{
		{
			"simple",
			simpleJob,
			1,
		},
		{
			"one cont",
			func(event *beat.Event) (j []jobs.Job, e error) {
				_, _ = simpleJob(event)
				return []jobs.Job{simpleJob}, nil
			},
			2,
		},
		{
			"multiple conts",
			func(event *beat.Event) (j []jobs.Job, e error) {
				_, _ = simpleJob(event)
				return []jobs.Job{
					defineJob(mapstr.M{"baz": "bot"}),
					defineJob(mapstr.M{"blah": "blargh"}),
				}, nil
			},
			3,
		},
		{
			"cancelled cont",
			func(event *beat.Event) (j []jobs.Job, e error) {
				eventext.CancelEvent(event)
				return []jobs.Job{simpleJob}, nil
			},
			1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan bool)
			pipel := &MockPipeline{}
			sync := &SyncPipelineWrapper{}
			wrapped := WithSyncPipelineWrapper(pipel, sync)

			client, err := wrapped.Connect()
			require.NoError(t, err)
			queue := runPublishJob(tc.job, client)
			for {
				if len(queue) == 0 {
					break
				}
				tf := queue[0]
				queue = queue[1:]
				conts := tf(context.Background())
				queue = append(queue, conts...)
			}
			err = client.Close()
			require.NoError(t, err)

			go func() {
				sync.Wait()
				done <- true
			}()

			wait := time.After(1000 * time.Millisecond)
			select {
			case <-done:
				assert.Fail(t, "pipeline exited before events were published")
			case <-wait:
			}

			sync.onACK(tc.acks)

			wait = time.After(1000 * time.Millisecond)
			select {
			case <-done:
			case <-wait:
				assert.Fail(t, "pipeline exceeded timeout after every event acked")
			}
		})
	}
}
