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

//go:build linux || darwin || windows

package docker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type MockClient struct {
	// containers to return on ContainerList call
	containers    [][]container.Summary
	containersErr error
	// event list to send on Events call
	events []any
	// done channel is closed when the client has sent all events
	done chan any
}

func (m *MockClient) ContainerList(ctx context.Context, options dockerclient.ContainerListOptions) (dockerclient.ContainerListResult, error) {
	if m.containersErr != nil {
		return dockerclient.ContainerListResult{}, m.containersErr
	}
	res := m.containers[0]
	m.containers = m.containers[1:]
	return dockerclient.ContainerListResult{Items: res}, nil
}

func (m *MockClient) Events(ctx context.Context, options dockerclient.EventsListOptions) dockerclient.EventsResult {
	eventsC := make(chan events.Message)
	errorsC := make(chan error)

	go func() {
		for _, event := range m.events {
			switch e := event.(type) {
			case events.Message:
				eventsC <- e
			case error:
				errorsC <- e
			}
		}
		close(m.done)
	}()

	return dockerclient.EventsResult{Messages: eventsC, Err: errorsC}
}

func (m *MockClient) ContainerInspect(ctx context.Context, containerID string, options dockerclient.ContainerInspectOptions) (dockerclient.ContainerInspectResult, error) {
	return dockerclient.ContainerInspectResult{}, errors.New("unimplemented")
}

func TestWatcherInitialization(t *testing.T) {
	watcher := runAndWait(testWatcher(t,
		[][]container.Summary{
			{
				container.Summary{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
				container.Summary{
					ID:              "6ac6ee8df5d4",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		nil,
	))

	assert.Equal(t, map[string]*Container{
		"0332dbd79e20": {
			ID:     "0332dbd79e20",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"6ac6ee8df5d4": {
			ID:     "6ac6ee8df5d4",
			Name:   "other",
			Image:  "nginx",
			Labels: map[string]string{},
		},
	}, watcher.Containers())
}

func TestWatcherInitializationShortID(t *testing.T) {
	watcher := runAndWait(testWatcherShortID(t,
		[][]container.Summary{
			{
				container.Summary{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
				container.Summary{
					ID:              "2345678901234",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		nil,
		true,
	))

	assert.Equal(t, map[string]*Container{
		"1234567890123": {
			ID:     "1234567890123",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"2345678901234": {
			ID:     "2345678901234",
			Name:   "other",
			Image:  "nginx",
			Labels: map[string]string{},
		},
	}, watcher.Containers())

	assert.Equal(t, &Container{
		ID:     "1234567890123",
		Name:   "containername",
		Image:  "busybox",
		Labels: map[string]string{"foo": "bar"},
	}, watcher.Container("123456789012"))
}

func TestWatcherAddEvents(t *testing.T) {
	watcher := runAndWait(testWatcher(t,
		[][]container.Summary{
			{
				container.Summary{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
			{
				container.Summary{
					ID:              "6ac6ee8df5d4",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{"label": "value"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		[]any{
			events.Message{
				Action: "start",
				Actor: events.Actor{
					ID: "6ac6ee8df5d4",
					Attributes: map[string]string{
						"name":  "other",
						"image": "nginx",
						"label": "value",
					},
				},
			},
		},
	))

	assert.Equal(t, map[string]*Container{
		"0332dbd79e20": {
			ID:     "0332dbd79e20",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"6ac6ee8df5d4": {
			ID:     "6ac6ee8df5d4",
			Name:   "other",
			Image:  "nginx",
			Labels: map[string]string{"label": "value"},
		},
	}, watcher.Containers())
}

func TestWatcherAddEventsShortID(t *testing.T) {
	watcher := runAndWait(testWatcherShortID(t,
		[][]container.Summary{
			{
				container.Summary{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
			{
				container.Summary{
					ID:              "2345678901234",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{"label": "value"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		[]any{
			events.Message{
				Action: "start",
				Actor: events.Actor{
					ID: "2345678901234",
					Attributes: map[string]string{
						"name":  "other",
						"image": "nginx",
						"label": "value",
					},
				},
			},
		},
		true,
	))

	assert.Equal(t, map[string]*Container{
		"1234567890123": {
			ID:     "1234567890123",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"2345678901234": {
			ID:     "2345678901234",
			Name:   "other",
			Image:  "nginx",
			Labels: map[string]string{"label": "value"},
		},
	}, watcher.Containers())
}

func TestWatcherUpdateEvent(t *testing.T) {
	watcher := runAndWait(testWatcher(t,
		[][]container.Summary{
			{
				{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
			{
				container.Summary{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "bar"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		[]any{
			events.Message{
				Action: "update",
				Actor: events.Actor{
					ID: "0332dbd79e20",
					Attributes: map[string]string{
						"name":  "containername",
						"image": "busybox",
						"label": "bar",
					},
				},
			},
		},
	))

	assert.Equal(t, map[string]*Container{
		"0332dbd79e20": {
			ID:     "0332dbd79e20",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"label": "bar"},
		},
	}, watcher.Containers())
	assert.Empty(t, watcher.deleted)
}

func TestWatcherUpdateEventShortID(t *testing.T) {
	watcher := runAndWait(testWatcherShortID(t,
		[][]container.Summary{
			{
				container.Summary{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
			{
				container.Summary{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "bar"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		[]any{
			events.Message{
				Action: "update",
				Actor: events.Actor{
					ID: "1234567890123",
					Attributes: map[string]string{
						"name":  "containername",
						"image": "busybox",
						"label": "bar",
					},
				},
			},
		},
		true,
	))

	assert.Equal(t, map[string]*Container{
		"1234567890123": {
			ID:     "1234567890123",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"label": "bar"},
		},
	}, watcher.Containers())
	assert.Empty(t, watcher.deleted)
}

func TestWatcherDie(t *testing.T) {
	watcher, clientDone := testWatcher(t,
		[][]container.Summary{
			{
				container.Summary{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		[]any{
			events.Message{
				Action: "die",
				Actor: events.Actor{
					ID: "0332dbd79e20",
				},
			},
		},
	)

	clock := newTestClock()
	watcher.clock = clock

	stopListener := watcher.ListenStop()

	err := watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	// Check it doesn't get removed while we request meta for the container
	for range 18 {
		watcher.Container("0332dbd79e20")
		clock.Sleep(watcher.cleanupTimeout / 2)
		watcher.runCleanup()
		if !assert.Len(t, watcher.Containers(), 1) {
			break
		}
	}

	// Wait to be sure that the delete event has been processed
	<-clientDone
	<-stopListener.Events()

	// Check that after the cleanup period the container is removed
	clock.Sleep(watcher.cleanupTimeout + 1*time.Second)
	watcher.runCleanup()
	assert.Empty(t, watcher.Containers())
}

func TestWatcherDieShortID(t *testing.T) {
	watcher, clientDone := testWatcherShortID(t,
		[][]container.Summary{
			{
				container.Summary{
					ID:              "0332dbd79e20aaa",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &container.NetworkSettingsSummary{},
				},
			},
		},
		[]any{
			events.Message{
				Action: "die",
				Actor: events.Actor{
					ID: "0332dbd79e20aaa",
				},
			},
		},
		true,
	)

	clock := newTestClock()
	watcher.clock = clock

	stopListener := watcher.ListenStop()

	err := watcher.Start()
	require.NoError(t, err)
	defer watcher.Stop()

	// Check it doesn't get removed while we request meta for the container
	for range 18 {
		watcher.Container("0332dbd79e20")
		clock.Sleep(watcher.cleanupTimeout / 2)
		watcher.runCleanup()
		if !assert.Len(t, watcher.Containers(), 1) {
			break
		}
	}

	// Wait to be sure that the delete event has been processed
	<-clientDone
	<-stopListener.Events()

	// Check that after the cleanup period the container is removed
	clock.Sleep(watcher.cleanupTimeout + 1*time.Second)
	watcher.runCleanup()
	assert.Empty(t, watcher.Containers())
}

func TestWatcherNoError(t *testing.T) {
	core, obs := observer.New(zapcore.DebugLevel)
	l, err := logp.ConfigureWithCoreLocal(logp.DefaultConfig(logp.DefaultEnvironment), core)
	require.NoError(t, err)
	client := &MockClient{
		containersErr: errors.New("test error"),
		events:        nil,
		done:          make(chan any),
	}
	w, err := NewWatcherWithClient(l, client, 200*time.Millisecond, true)
	if err != nil {
		t.Fatal(err)
	}
	watcher, ok := w.(*watcher)
	if !ok {
		t.Fatal("'watcher' was supposed to be pointer to the watcher structure")
	}
	clock := newTestClock()
	watcher.clock = clock

	err = watcher.Start()
	require.NoError(t, err)
	watcher.Stop()
	require.Len(t, obs.FilterMessageSnippet("Failed to call listContainers").All(), 1)
}

func testWatcher(t *testing.T, containers [][]container.Summary, events []any) (*watcher, chan any) {
	return testWatcherShortID(t, containers, events, false)
}

func testWatcherShortID(t *testing.T, containers [][]container.Summary, events []any, enable bool) (*watcher, chan any) {
	logger := logptest.NewTestingLogger(t, "")

	client := &MockClient{
		containers: containers,
		events:     events,
		done:       make(chan any),
	}

	w, err := NewWatcherWithClient(logger, client, 200*time.Millisecond, enable)
	if err != nil {
		t.Fatal(err)
	}
	watcher, ok := w.(*watcher)
	if !ok {
		t.Fatal("'watcher' was supposed to be pointer to the watcher structure")
	}

	return watcher, client.done
}

func runAndWait(w *watcher, done chan any) *watcher {
	_ = w.Start()
	<-done
	w.Stop()
	return w
}

type testClock struct {
	sync.Mutex

	now time.Time
}

func newTestClock() *testClock {
	return &testClock{now: time.Time{}}
}

func (c *testClock) Now() time.Time {
	c.Lock()
	defer c.Unlock()

	c.now = c.now.Add(1)
	return c.now
}

func (c *testClock) Sleep(d time.Duration) {
	c.Lock()
	defer c.Unlock()

	c.now = c.now.Add(d)
}
