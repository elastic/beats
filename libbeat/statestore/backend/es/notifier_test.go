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

package es

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func createTestConfigs(t *testing.T, n int) []*conf.C {
	var res []*conf.C
	for i := 0; i < n; i++ {
		c, err := conf.NewConfigFrom(map[string]any{
			"id": i,
		})
		require.NoError(t, err)
		require.NotNil(t, c)
		id, err := c.Int("id", -1)
		require.NoError(t, err, "sanity check: id is stored")
		require.Equal(t, int64(i), id, "sanity check: id is correct")
		res = append(res, c)
	}
	return res
}

func wgWait(t *testing.T, wg *sync.WaitGroup) {
	const timeout = 1 * time.Second
	t.Helper()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		require.Fail(t, "timeout waiting for WaitGroup")
	}
}

func TestSanity(t *testing.T) {
	assert.Equal(t, createTestConfigs(t, 5), createTestConfigs(t, 5))
	assert.NotEqual(t, createTestConfigs(t, 4), createTestConfigs(t, 5))
	assert.NotEqual(t, createTestConfigs(t, 5)[3], createTestConfigs(t, 5)[4])
}

func TestSubscribeAndNotify(t *testing.T) {
	notifier := NewNotifier()

	var (
		wg             sync.WaitGroup
		mx             sync.Mutex
		receivedFirst  []*conf.C
		receivedSecond []*conf.C
	)

	unsubFirst := notifier.Subscribe(func(c *conf.C) {
		defer wg.Done()
		mx.Lock()
		defer mx.Unlock()
		receivedFirst = append(receivedFirst, c)
	})
	defer unsubFirst()

	unsubSecond := notifier.Subscribe(func(c *conf.C) {
		defer wg.Done()
		mx.Lock()
		defer mx.Unlock()
		receivedSecond = append(receivedSecond, c)
	})
	defer unsubSecond()

	const totalNotifications = 3

	configs := createTestConfigs(t, totalNotifications)

	wg.Add(totalNotifications * 2)
	for _, config := range configs {
		notifier.Notify(config)
	}

	wgWait(t, &wg)
	assert.ElementsMatch(t, configs, receivedFirst)
	assert.ElementsMatch(t, configs, receivedSecond)

	// Receive old config
	wg.Add(1)
	notifier.Subscribe(func(c *conf.C) {
		defer wg.Done()
		mx.Lock()
		defer mx.Unlock()
	})
	wgWait(t, &wg)
}

func TestUnsubscribe(t *testing.T) {
	var (
		wg                            sync.WaitGroup
		mx                            sync.Mutex
		receivedFirst, receivedSecond []*conf.C
	)

	notifier := NewNotifier()

	unsubFirst := notifier.Subscribe(func(c *conf.C) {
		defer wg.Done()
		mx.Lock()
		defer mx.Unlock()
		receivedFirst = append(receivedFirst, c)
	})
	defer unsubFirst()

	unsubSecond := notifier.Subscribe(func(c *conf.C) {
		defer wg.Done()
		mx.Lock()
		defer mx.Unlock()
		receivedSecond = append(receivedSecond, c)
	})
	defer unsubSecond()

	const totalNotifications = 3

	configs := createTestConfigs(t, totalNotifications)

	// Unsubscribe first
	unsubFirst()

	// Notify
	wg.Add(totalNotifications)
	for _, config := range configs {
		notifier.Notify(config)
	}

	wgWait(t, &wg)
	assert.Empty(t, receivedFirst)
	assert.ElementsMatch(t, configs, receivedSecond)
}

func TestConcurrentSubscribeAndNotify(t *testing.T) {
	notifier := NewNotifier()

	var (
		wg, wgSub sync.WaitGroup
		mx, mxSub sync.Mutex
		received  []*conf.C
		unsubFns  []UnsubscribeFunc
	)

	// Concurrent subscribers
	const count = 10
	wgSub.Add(count)
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wgSub.Done()
			mxSub.Lock()
			defer mxSub.Unlock()
			unsub := notifier.Subscribe(func(c *conf.C) {
				defer wg.Done()
				mx.Lock()
				defer mx.Unlock()
				received = append(received, c)
			})
			unsubFns = append(unsubFns, unsub)
		}()
	}
	defer func() {
		for _, unsubFn := range unsubFns {
			unsubFn()
		}
	}()

	// Wait for all subscribers to be added
	wgWait(t, &wgSub)

	// Notify
	c := createTestConfigs(t, 1)[0]
	notifier.Notify(c)

	// Wait for all
	wgWait(t, &wg)
	expected := make([]*conf.C, count)
	for i := 0; i < count; i++ {
		expected[i] = c
	}
	assert.Equal(t, expected, received)
}
