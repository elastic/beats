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

//go:build windows
// +build windows

package wineventlog

import (
	"strconv"
	"testing"

	"github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestEventIterator(t *testing.T) {
	logp.TestingSetup()

	writer, tearDown := createLog(t)
	defer tearDown()

	const eventCount = 1500
	for i := 0; i < eventCount; i++ {
		safeWriteEvent(t, writer, eventlog.Info, 1, []string{"Test message " + strconv.Itoa(i+1)})
	}

	// Validate the assumption that 1024 is the max number of handles supported
	// by EvtNext.
	t.Run("max_handles_assumption", func(t *testing.T) {
		log := openLog(t, winlogbeatTestLogName)
		defer log.Close()

		var (
			numReturned uint32
			handles     = [evtNextMaxHandles + 1]EvtHandle{}
		)

		// Too many handles.
		err := _EvtNext(log, uint32(len(handles)), &handles[0], 0, 0, &numReturned)
		assert.Equal(t, windows.ERROR_INVALID_PARAMETER, err)

		// The max number of handles.
		err = _EvtNext(log, evtNextMaxHandles, &handles[0], 0, 0, &numReturned)
		if assert.NoError(t, err) {
			for _, h := range handles[:numReturned] {
				h.Close()
			}
		}
	})

	t.Run("no_subscription", func(t *testing.T) {
		_, err := NewEventIterator()
		assert.Error(t, err)
	})

	t.Run("with_subscription", func(t *testing.T) {
		log := openLog(t, winlogbeatTestLogName)
		defer log.Close()

		itr, err := NewEventIterator(WithSubscription(log))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { assert.NoError(t, itr.Close()) }()

		assert.Nil(t, itr.subscriptionFactory)
		assert.NotEqual(t, NilHandle, itr.subscription)
	})

	t.Run("with_subscription_factory", func(t *testing.T) {
		factory := func() (handle EvtHandle, err error) {
			return openLog(t, winlogbeatTestLogName), nil
		}
		itr, err := NewEventIterator(WithSubscriptionFactory(factory))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { assert.NoError(t, itr.Close()) }()

		assert.NotNil(t, itr.subscriptionFactory)
		assert.NotEqual(t, NilHandle, itr.subscription)
	})

	t.Run("with_batch_size", func(t *testing.T) {
		log := openLog(t, winlogbeatTestLogName)
		defer log.Close()

		t.Run("default", func(t *testing.T) {
			itr, err := NewEventIterator(WithSubscription(log))
			if err != nil {
				t.Fatal(err)
			}
			assert.EqualValues(t, evtNextDefaultHandles, itr.batchSize)
		})

		t.Run("custom", func(t *testing.T) {
			itr, err := NewEventIterator(WithSubscription(log), WithBatchSize(128))
			if err != nil {
				t.Fatal(err)
			}
			assert.EqualValues(t, 128, itr.batchSize)
		})

		t.Run("too_small", func(t *testing.T) {
			itr, err := NewEventIterator(WithSubscription(log), WithBatchSize(0))
			if err != nil {
				t.Fatal(err)
			}
			assert.EqualValues(t, evtNextDefaultHandles, itr.batchSize)
		})

		t.Run("too_big", func(t *testing.T) {
			itr, err := NewEventIterator(WithSubscription(log), WithBatchSize(evtNextMaxHandles+1))
			if err != nil {
				t.Fatal(err)
			}
			assert.EqualValues(t, evtNextMaxHandles, itr.batchSize)
		})
	})

	t.Run("iterate", func(t *testing.T) {
		log := openLog(t, winlogbeatTestLogName)
		defer log.Close()

		itr, err := NewEventIterator(WithSubscription(log), WithBatchSize(13))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { assert.NoError(t, itr.Close()) }()

		var iterateCount int
		for h, ok := itr.Next(); ok; h, ok = itr.Next() {
			h.Close()

			if !assert.NotZero(t, h) {
				return
			}

			iterateCount++
		}
		if err := itr.Err(); err != nil {
			t.Fatal(err)
		}

		assert.EqualValues(t, eventCount, iterateCount)
	})

	// Check for regressions of https://github.com/elastic/beats/issues/3076
	// where EvtNext fails reading batch of large events.
	//
	// Note: As of 2020-03 Windows 2019 no longer exhibits this behavior.
	// Instead EvtNext simply returns fewer handles that the requested size.
	t.Run("rpc_error", func(t *testing.T) {
		log := openLog(t, winlogbeatTestLogName)
		defer log.Close()

		// Mock the behavior to simplify testing since it's not reproducible
		// on all Windows versions.
		mockEvtNext := func(resultSet EvtHandle, eventArraySize uint32, eventArray *EvtHandle, timeout uint32, flags uint32, numReturned *uint32) (err error) {
			if eventArraySize > 3 {
				return windows.RPC_S_INVALID_BOUND
			}
			return _EvtNext(resultSet, eventArraySize, eventArray, timeout, flags, numReturned)
		}

		// If you create the iterator with only a subscription handle then
		// no recovery is possible without data loss.
		t.Run("no_recovery", func(t *testing.T) {
			itr, err := NewEventIterator(WithSubscription(log))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { assert.NoError(t, itr.Close()) }()

			itr.evtNext = mockEvtNext

			h, ok := itr.Next()
			assert.False(t, ok)
			assert.Zero(t, h)
			if assert.Error(t, itr.Err()) {
				assert.Contains(t, itr.Err().Error(), "try reducing the batch size")
				assert.Equal(t, windows.RPC_S_INVALID_BOUND, errors.Cause(itr.Err()))
			}
		})

		t.Run("automated_recovery", func(t *testing.T) {
			var numFactoryInvocations int
			var bookmark Bookmark

			// Create a proper subscription factor that resumes from the last
			// read position by using bookmarks.
			factory := func() (handle EvtHandle, err error) {
				numFactoryInvocations++
				log := openLog(t, winlogbeatTestLogName)

				if bookmark != 0 {
					// Seek to bookmark location.
					err := EvtSeek(log, 0, EvtHandle(bookmark), EvtSeekRelativeToBookmark|EvtSeekStrict)
					if err != nil {
						t.Fatal(err)
					}

					// Seek to one event after bookmark (unread position).
					if err = EvtSeek(log, 1, NilHandle, EvtSeekRelativeToCurrent); err != nil {
						t.Fatal(err)
					}
				}

				return log, err
			}

			itr, err := NewEventIterator(WithSubscriptionFactory(factory), WithBatchSize(10))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { assert.NoError(t, itr.Close()) }()

			// Mock the EvtNext to cause the the RPC_S_INVALID_BOUND error.
			itr.evtNext = mockEvtNext

			var iterateCount int
			for h, ok := itr.Next(); ok; h, ok = itr.Next() {
				func() {
					defer h.Close()

					if !assert.NotZero(t, h) {
						t.FailNow()
					}

					// Store last read position.
					if bookmark != 0 {
						bookmark.Close()
					}
					bookmark, err = NewBookmarkFromEvent(h)
					if err != nil {
						t.Fatal(err)
					}

					iterateCount++
				}()
			}
			if err := itr.Err(); err != nil {
				t.Fatal(err)
			}

			// Validate that the factory has been used to recover and
			// that we received all the events.
			assert.Greater(t, numFactoryInvocations, 1)
			assert.EqualValues(t, eventCount, iterateCount)
		})
	})
}
