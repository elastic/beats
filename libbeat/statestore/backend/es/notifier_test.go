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

	"github.com/google/go-cmp/cmp"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func createTestConfigs(n int) ([]*conf.C, error) {
	var res []*conf.C
	for i := 0; i < n; i++ {
		c, err := conf.NewConfigFrom(map[string]any{
			"id": i,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, nil
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func confDiff(t *testing.T, c1, c2 *conf.C) string {
	var m1, m2 map[string]any
	err := c1.Unpack(&m1)
	if err != nil {
		t.Fatal(err)
	}
	err = c2.Unpack(&m2)
	if err != nil {
		t.Fatal(err)
	}

	return cmp.Diff(m1, m2)
}

func getMatchOrdered(t *testing.T, conf1, conf2 []*conf.C) []*conf.C {
	var matchingOrdered []*conf.C
	for _, c1 := range conf1 {
		for _, c2 := range conf2 {
			if confDiff(t, c1, c2) == "" {
				matchingOrdered = append(matchingOrdered, c2)
			}
		}
	}
	return matchingOrdered
}

// Test subscribing and notifying
func TestSubscribeAndNotify(t *testing.T) {
	notifier := NewNotifier()

	var (
		wg             sync.WaitGroup
		mx             sync.Mutex
		receivedFirst  []*conf.C
		receivedSecond []*conf.C
	)

	unsubFirst := notifier.Subscribe(func(c *conf.C) {
		mx.Lock()
		defer mx.Unlock()
		receivedFirst = append(receivedFirst, c)
		wg.Done()
	})
	defer unsubFirst()

	unsubSecond := notifier.Subscribe(func(c *conf.C) {
		mx.Lock()
		defer mx.Unlock()
		receivedSecond = append(receivedSecond, c)
		wg.Done()
	})
	defer unsubSecond()

	const (
		totalNotifications = 3
	)

	configs, err := createTestConfigs(totalNotifications)
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(totalNotifications * 2)
	for i := 0; i < totalNotifications; i++ {
		notifier.Notify(configs[i])
	}

	if !waitWithTimeout(&wg, time.Second) {
		t.Fatal("Wait for notifications failed with timeout")
	}

	receivedFirst = getMatchOrdered(t, configs, receivedFirst)
	diff := cmp.Diff(totalNotifications, len(receivedFirst))
	if diff != "" {
		t.Fatal(diff)
	}

	receivedSecond = getMatchOrdered(t, configs, receivedSecond)
	diff = cmp.Diff(totalNotifications, len(receivedSecond))
	if diff != "" {
		t.Fatal(diff)
	}
}

// Test unsubscribing
func TestUnsubscribe(t *testing.T) {

	var (
		wg                            sync.WaitGroup
		mx                            sync.Mutex
		receivedFirst, receivedSecond []*conf.C
	)

	notifier := NewNotifier()

	unsubFirst := notifier.Subscribe(func(c *conf.C) {
		mx.Lock()
		defer mx.Unlock()
		receivedFirst = append(receivedFirst, c)
		wg.Done()
	})
	defer unsubFirst()

	unsubSecond := notifier.Subscribe(func(c *conf.C) {
		mx.Lock()
		defer mx.Unlock()
		receivedSecond = append(receivedSecond, c)
		wg.Done()
	})
	defer unsubSecond()

	const (
		totalNotifications = 3
	)

	configs, err := createTestConfigs(totalNotifications)
	if err != nil {
		t.Fatal(err)
	}

	// Unsubscribe first
	unsubFirst()

	// Notify
	wg.Add(totalNotifications)
	for i := 0; i < totalNotifications; i++ {
		notifier.Notify(configs[i])
	}

	if !waitWithTimeout(&wg, time.Second) {
		t.Fatal("Wait for notifications failed with timeout")
	}

	diff := cmp.Diff(0, len(receivedFirst))
	if diff != "" {
		t.Fatal(diff)
	}

	receivedSecond = getMatchOrdered(t, configs, receivedSecond)
	diff = cmp.Diff(totalNotifications, len(receivedSecond))
	if diff != "" {
		t.Fatal(diff)
	}
}

// Test concurrency
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
			mxSub.Lock()
			defer mxSub.Unlock()
			unsub := notifier.Subscribe(func(c *conf.C) {
				mx.Lock()
				defer mx.Unlock()
				received = append(received, c)
				wg.Done()
			})
			unsubFns = append(unsubFns, unsub)
			wgSub.Done()
		}()
	}
	defer func() {
		for _, unsubfn := range unsubFns {
			unsubfn()
		}
	}()

	// Wait for all subscribers to be added
	wgSub.Wait()

	// Notify
	c, err := conf.NewConfigFrom(map[string]any{
		"id": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	notifier.Notify(c)

	// Wait for all
	if !waitWithTimeout(&wg, time.Second) {
		t.Fatal("Wait for notifications failed with timeout")
	}

	diff := cmp.Diff(count, len(received))
	if diff != "" {
		t.Fatal(diff)
	}
}
