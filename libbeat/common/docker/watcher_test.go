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
// +build linux darwin windows

package docker

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"

	"github.com/elastic/beats/v8/libbeat/logp"
)

type MockClient struct {
	// containers to return on ContainerList call
	containers [][]types.Container
	// event list to send on Events call
	events []interface{}
	// done channel is closed when the client has sent all events
	done chan interface{}
}

func (m *MockClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	res := m.containers[0]
	m.containers = m.containers[1:]
	return res, nil
}

func (m *MockClient) Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
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

	return eventsC, errorsC
}

func (m *MockClient) ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, errors.New("unimplemented")
}

func TestWatcherInitialization(t *testing.T) {
	watcher := runAndWait(testWatcher(t, true,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
				types.Container{
					ID:              "6ac6ee8df5d4",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		nil,
	))

	assert.Equal(t, map[string]*Container{
		"0332dbd79e20": &Container{
			ID:     "0332dbd79e20",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"6ac6ee8df5d4": &Container{
			ID:     "6ac6ee8df5d4",
			Name:   "other",
			Image:  "nginx",
			Labels: map[string]string{},
		},
	}, watcher.Containers())
}

func TestWatcherInitializationShortID(t *testing.T) {
	watcher := runAndWait(testWatcherShortID(t, true,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
				types.Container{
					ID:              "2345678901234",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		nil,
		true,
	))

	assert.Equal(t, map[string]*Container{
		"1234567890123": &Container{
			ID:     "1234567890123",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"2345678901234": &Container{
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
	watcher := runAndWait(testWatcher(t, true,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
			[]types.Container{
				types.Container{
					ID:              "6ac6ee8df5d4",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{"label": "value"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		[]interface{}{
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
		"0332dbd79e20": &Container{
			ID:     "0332dbd79e20",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"6ac6ee8df5d4": &Container{
			ID:     "6ac6ee8df5d4",
			Name:   "other",
			Image:  "nginx",
			Labels: map[string]string{"label": "value"},
		},
	}, watcher.Containers())
}

func TestWatcherAddEventsShortID(t *testing.T) {
	watcher := runAndWait(testWatcherShortID(t, true,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"foo": "bar"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
			[]types.Container{
				types.Container{
					ID:              "2345678901234",
					Names:           []string{"/other"},
					Image:           "nginx",
					Labels:          map[string]string{"label": "value"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		[]interface{}{
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
		"1234567890123": &Container{
			ID:     "1234567890123",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"foo": "bar"},
		},
		"2345678901234": &Container{
			ID:     "2345678901234",
			Name:   "other",
			Image:  "nginx",
			Labels: map[string]string{"label": "value"},
		},
	}, watcher.Containers())
}

func TestWatcherUpdateEvent(t *testing.T) {
	watcher := runAndWait(testWatcher(t, true,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
			[]types.Container{
				types.Container{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "bar"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		[]interface{}{
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
		"0332dbd79e20": &Container{
			ID:     "0332dbd79e20",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"label": "bar"},
		},
	}, watcher.Containers())
	assert.Equal(t, 0, len(watcher.deleted))
}

func TestWatcherUpdateEventShortID(t *testing.T) {
	watcher := runAndWait(testWatcherShortID(t, true,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
			[]types.Container{
				types.Container{
					ID:              "1234567890123",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "bar"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		[]interface{}{
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
		"1234567890123": &Container{
			ID:     "1234567890123",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"label": "bar"},
		},
	}, watcher.Containers())
	assert.Equal(t, 0, len(watcher.deleted))
}

func TestWatcherDie(t *testing.T) {
	watcher, clientDone := testWatcher(t, false,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "0332dbd79e20",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		[]interface{}{
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

	watcher.Start()
	defer watcher.Stop()

	// Check it doesn't get removed while we request meta for the container
	for i := 0; i < 18; i++ {
		watcher.Container("0332dbd79e20")
		clock.Sleep(watcher.cleanupTimeout / 2)
		watcher.runCleanup()
		if !assert.Equal(t, 1, len(watcher.Containers())) {
			break
		}
	}

	// Wait to be sure that the delete event has been processed
	<-clientDone
	<-stopListener.Events()

	// Check that after the cleanup period the container is removed
	clock.Sleep(watcher.cleanupTimeout + 1*time.Second)
	watcher.runCleanup()
	assert.Equal(t, 0, len(watcher.Containers()))
}

func TestWatcherDieShortID(t *testing.T) {
	watcher, clientDone := testWatcherShortID(t, false,
		[][]types.Container{
			[]types.Container{
				types.Container{
					ID:              "0332dbd79e20aaa",
					Names:           []string{"/containername", "othername"},
					Image:           "busybox",
					Labels:          map[string]string{"label": "foo"},
					NetworkSettings: &types.SummaryNetworkSettings{},
				},
			},
		},
		[]interface{}{
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

	watcher.Start()
	defer watcher.Stop()

	// Check it doesn't get removed while we request meta for the container
	for i := 0; i < 18; i++ {
		watcher.Container("0332dbd79e20")
		clock.Sleep(watcher.cleanupTimeout / 2)
		watcher.runCleanup()
		if !assert.Equal(t, 1, len(watcher.Containers())) {
			break
		}
	}

	// Wait to be sure that the delete event has been processed
	<-clientDone
	<-stopListener.Events()

	// Check that after the cleanup period the container is removed
	clock.Sleep(watcher.cleanupTimeout + 1*time.Second)
	watcher.runCleanup()
	assert.Equal(t, 0, len(watcher.Containers()))
}

func testWatcher(t *testing.T, kill bool, containers [][]types.Container, events []interface{}) (*watcher, chan interface{}) {
	return testWatcherShortID(t, kill, containers, events, false)
}

func testWatcherShortID(t *testing.T, kill bool, containers [][]types.Container, events []interface{}, enable bool) (*watcher, chan interface{}) {
	logp.TestingSetup()

	client := &MockClient{
		containers: containers,
		events:     events,
		done:       make(chan interface{}),
	}

	w, err := NewWatcherWithClient(logp.L(), client, 200*time.Millisecond, enable)
	if err != nil {
		t.Fatal(err)
	}
	watcher, ok := w.(*watcher)
	if !ok {
		t.Fatal("'watcher' was supposed to be pointer to the watcher structure")
	}

	return watcher, client.done
}

func runAndWait(w *watcher, done chan interface{}) *watcher {
	w.Start()
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
