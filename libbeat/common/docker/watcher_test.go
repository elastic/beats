package docker

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
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

	assert.Equal(t, watcher.Containers(), map[string]*Container{
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
	})
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

	assert.Equal(t, watcher.Containers(), map[string]*Container{
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
	})
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

	assert.Equal(t, watcher.Containers(), map[string]*Container{
		"0332dbd79e20": &Container{
			ID:     "0332dbd79e20",
			Name:   "containername",
			Image:  "busybox",
			Labels: map[string]string{"label": "bar"},
		},
	})
	assert.Equal(t, len(watcher.deleted), 0)
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

	// Check it doesn't get removed while we request meta for the container
	for i := 0; i < 18; i++ {
		watcher.Container("0332dbd79e20")
		assert.Equal(t, len(watcher.Containers()), 1)
		time.Sleep(50 * time.Millisecond)
	}

	// Now it should get removed
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, len(watcher.Containers()), 0)
}

func runWatcher(t *testing.T, kill bool, containers [][]types.Container, events []interface{}) *watcher {
	client := &MockClient{
		containers: containers,
		events:     events,
		done:       make(chan interface{}),
	}

	watcher, err := NewWatcherWithClient(client, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
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
