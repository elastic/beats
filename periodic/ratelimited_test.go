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

package periodic

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type syncBuffer struct {
	buff bytes.Buffer
	mu   sync.Mutex
}

func (s *syncBuffer) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buff.Read(p)
}

func (s *syncBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return fmt.Fprintf(&s.buff, "%s", p)
}

func TestRateLimitedLogger(t *testing.T) {
	pattern := "%d occurrences in the last %s"

	newLogger := func() (io.Reader, func(count uint64, d time.Duration)) {
		sbuff := &syncBuffer{}

		logFn := func(count uint64, d time.Duration) {
			fmt.Fprintf(sbuff, pattern, count, d)
		}
		return sbuff, logFn
	}

	now := time.Now()

	t.Run("Start", func(t *testing.T) {
		r := NewDoer(math.MaxInt64, func(count uint64, d time.Duration) {})
		defer r.Stop()
		r.nowFn = func() time.Time { return now }

		r.Start()

		assert.True(t, r.started.Load(),
			"Start() was called, thus 'started' should be true")
		assert.NotEmpty(t, r.lastDone, "lastDone should have been set")
	})

	t.Run("Start twice", func(t *testing.T) {
		r := NewDoer(math.MaxInt64, func(count uint64, d time.Duration) {})
		defer r.Stop()

		r.nowFn = func() time.Time { return now }

		r.Start()
		r.nowFn = func() time.Time { return now.Add(time.Minute) }
		r.Start()

		assert.Equal(t, now, r.lastDone, "lastDone should have been set a second time")
	})

	t.Run("Stop", func(t *testing.T) {
		tcs := []struct {
			name  string
			count int
		}{
			{name: "once", count: 1},
			{name: "twice", count: 2},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				buff, logFn := newLogger()
				r := NewDoer(42*time.Second, logFn)
				r.nowFn = func() time.Time { return now }

				tch := make(chan time.Time)
				r.newTickerFn = func(duration time.Duration) *time.Ticker {
					return &time.Ticker{C: tch}
				}

				r.Start()

				r.nowFn = func() time.Time { return now.Add(42 * time.Second) }

				r.count.Add(1)
				for i := 0; i < tc.count; i++ {
					r.Stop()
				}

				bs, err := io.ReadAll(buff)
				require.NoError(t, err, "failed reading logs")
				logs := string(bs)
				got := strings.TrimSpace(logs)

				assert.False(t, r.started.Load(),
					"Stop() was called, thus 'started' should be false")
				assert.Len(t, strings.Split(got, "\n"), 1)
				assert.Contains(t, logs, fmt.Sprintf(pattern, 1, 42*time.Second))

			})
		}
	})

	t.Run("Add", func(t *testing.T) {
		buff, logFn := newLogger()
		r := NewDoer(42*time.Second, logFn)
		defer r.Stop()

		r.nowFn = func() time.Time { return now }

		tch := make(chan time.Time)
		r.newTickerFn = func(duration time.Duration) *time.Ticker {
			return &time.Ticker{C: tch}
		}

		r.Start()
		r.Add()

		r.nowFn = func() time.Time { return now.Add(42 * time.Second) }
		tch <- now.Add(42 * time.Second)

		var logs string
		assert.Eventually(t, func() bool {
			bs, err := io.ReadAll(buff)
			require.NoError(t, err, "failed reading logs")
			logs = strings.TrimSpace(string(bs))

			return len(strings.Split(logs, "\n")) == 1
		}, time.Second, 100*time.Millisecond, "should have found 1 do")

		assert.Contains(t, logs, fmt.Sprintf(pattern, 1, 42*time.Second))
	})

	t.Run("AddN", func(t *testing.T) {
		buff, logFn := newLogger()
		r := NewDoer(42*time.Second, logFn)
		defer r.Stop()

		r.nowFn = func() time.Time { return now }

		tch := make(chan time.Time)
		r.newTickerFn = func(duration time.Duration) *time.Ticker {
			return &time.Ticker{C: tch}
		}

		r.Start()
		r.AddN(42)

		r.nowFn = func() time.Time { return now.Add(42 * time.Second) }
		tch <- now.Add(42 * time.Second)

		var logs string
		assert.Eventually(t, func() bool {
			bs, err := io.ReadAll(buff)
			require.NoError(t, err, "failed reading logs")
			logs = strings.TrimSpace(string(bs))

			return len(strings.Split(logs, "\n")) == 1
		}, time.Second, 100*time.Millisecond, "should have found 1 do")

		assert.Contains(t, logs, fmt.Sprintf(pattern, 42, 42*time.Second))
	})
}
