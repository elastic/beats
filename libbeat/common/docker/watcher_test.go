package docker

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"

	"github.com/elastic/beats/libbeat/logp"
)

type MockClient struct {
	// containers to return on ContainerList call
	containers [][]types.Container
	// event list to send on Events call
	events []interface{}

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

func TestWatcherInitialization(t *testing.T) {
	watcher := runWatcher(t, true,
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
		nil)

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
	watcher := runWatcherShortID(t, true,
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
		nil, true)

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
	watcher := runWatcher(t, true,
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
	)

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
	watcher := runWatcherShortID(t, true,
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
	)

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
	watcher := runWatcher(t, true,
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
	)

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
	watcher := runWatcherShortID(t, true,
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
	)

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
	watcher := runWatcher(t, false,
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
	defer watcher.Stop()

	// Check it doesn't get removed while we request meta for the container
	for i := 0; i < 18; i++ {
		watcher.Container("0332dbd79e20")
		assert.Equal(t, 1, len(watcher.Containers()))
		time.Sleep(50 * time.Millisecond)
	}

	// Checks a max of 10s for the watcher containers to be updated
	for i := 0; i < 100; i++ {
		// Now it should get removed
		time.Sleep(100 * time.Millisecond)

		if len(watcher.Containers()) == 0 {
			break
		}
	}

	assert.Equal(t, 0, len(watcher.Containers()))
}

func TestWatcherDieShortID(t *testing.T) {
	watcher := runWatcherShortID(t, false,
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
	defer watcher.Stop()

	// Check it doesn't get removed while we request meta for the container
	for i := 0; i < 18; i++ {
		watcher.Container("0332dbd79e20")
		assert.Equal(t, 1, len(watcher.Containers()))
		time.Sleep(50 * time.Millisecond)
	}

	// Checks a max of 10s for the watcher containers to be updated
	for i := 0; i < 100; i++ {
		// Now it should get removed
		time.Sleep(100 * time.Millisecond)

		if len(watcher.Containers()) == 0 {
			break
		}
	}

	assert.Equal(t, 0, len(watcher.Containers()))
}

func runWatcher(t *testing.T, kill bool, containers [][]types.Container, events []interface{}) *watcher {
	return runWatcherShortID(t, kill, containers, events, false)
}

func runWatcherShortID(t *testing.T, kill bool, containers [][]types.Container, events []interface{}, enable bool) *watcher {
	logp.TestingSetup()

	client := &MockClient{
		containers: containers,
		events:     events,
		done:       make(chan interface{}),
	}

	w, err := NewWatcherWithClient(client, 200*time.Millisecond, enable)
	if err != nil {
		t.Fatal(err)
	}
	watcher, ok := w.(*watcher)
	if !ok {
		t.Fatal("'watcher' was supposed to be pointer to the watcher structure")
	}

	err = watcher.Start()
	if err != nil {
		t.Fatal(err)
	}

	<-client.done
	if kill {
		watcher.Stop()
		watcher.stopped.Wait()
	}

	return watcher
}
