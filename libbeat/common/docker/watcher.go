package docker

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"golang.org/x/net/context"

	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

// Select Docker API version
const (
	dockerAPIVersion = "1.22"
	shortIDLen       = 12
)

// Watcher reads docker events and keeps a list of known containers
type Watcher interface {
	// Start watching docker API for new containers
	Start() error

	// Stop watching docker API for new containers
	Stop()

	// Container returns the running container with the given ID or nil if unknown
	Container(ID string) *Container

	// Containers returns the list of known containers
	Containers() map[string]*Container

	// ListenStart returns a bus listener to receive container started events, with a `container` key holding it
	ListenStart() bus.Listener

	// ListenStop returns a bus listener to receive container stopped events, with a `container` key holding it
	ListenStop() bus.Listener
}

// TLSConfig for docker socket connection
type TLSConfig struct {
	CA          string `config:"certificate_authority"`
	Certificate string `config:"certificate"`
	Key         string `config:"key"`
}

type watcher struct {
	sync.RWMutex
	client             Client
	ctx                context.Context
	stop               context.CancelFunc
	containers         map[string]*Container
	deleted            map[string]time.Time // deleted annotations key -> last access time
	cleanupTimeout     time.Duration
	lastValidTimestamp int64
	stopped            sync.WaitGroup
	bus                bus.Bus
	shortID            bool // whether to store short ID in "containers" too
}

// Container info retrieved by the watcher
type Container struct {
	ID          string
	Name        string
	Image       string
	Labels      map[string]string
	IPAddresses []string
	Ports       []types.Port
}

// Client for docker interface
type Client interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error)
}

// WatcherConstructor represent a function that creates a new Watcher from giving parameters
type WatcherConstructor func(host string, tls *TLSConfig, storeShortID bool) (Watcher, error)

// NewWatcher returns a watcher running for the given settings
func NewWatcher(host string, tls *TLSConfig, storeShortID bool) (Watcher, error) {
	var httpClient *http.Client
	if tls != nil {
		options := tlsconfig.Options{
			CAFile:   tls.CA,
			CertFile: tls.Certificate,
			KeyFile:  tls.Key,
		}

		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
		}
	}

	client, err := client.NewClient(host, dockerAPIVersion, httpClient, nil)
	if err != nil {
		return nil, err
	}

	return NewWatcherWithClient(client, 60*time.Second, storeShortID)
}

// NewWatcherWithClient creates a new Watcher from a given Docker client
func NewWatcherWithClient(client Client, cleanupTimeout time.Duration, storeShortID bool) (Watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &watcher{
		client:         client,
		ctx:            ctx,
		stop:           cancel,
		containers:     make(map[string]*Container),
		deleted:        make(map[string]time.Time),
		cleanupTimeout: cleanupTimeout,
		bus:            bus.New("docker"),
		shortID:        storeShortID,
	}, nil
}

// Container returns the running container with the given ID or nil if unknown
func (w *watcher) Container(ID string) *Container {
	w.RLock()
	container := w.containers[ID]
	if container == nil {
		w.RUnlock()
		return nil
	}
	_, ok := w.deleted[container.ID]
	w.RUnlock()

	// Update last access time if it's deleted
	if ok {
		w.Lock()
		w.deleted[container.ID] = time.Now()
		w.Unlock()
	}

	return container
}

// Containers returns the list of known containers
func (w *watcher) Containers() map[string]*Container {
	w.RLock()
	defer w.RUnlock()
	res := make(map[string]*Container)
	for k, v := range w.containers {
		if !w.shortID || len(k) != shortIDLen {
			res[k] = v
		}
	}
	return res
}

// Start watching docker API for new containers
func (w *watcher) Start() error {
	// Do initial scan of existing containers
	logp.Debug("docker", "Start docker containers scanner")
	w.lastValidTimestamp = time.Now().Unix()

	w.Lock()
	defer w.Unlock()
	containers, err := w.listContainers(types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, c := range containers {
		w.containers[c.ID] = c
		if w.shortID {
			w.containers[c.ID[:shortIDLen]] = c
		}
	}

	// Emit all start events (avoid blocking if the bus get's blocked)
	go func() {
		for _, c := range containers {
			w.bus.Publish(bus.Event{
				"start":     true,
				"container": c,
			})
		}
	}()

	w.stopped.Add(2)
	go w.watch()
	go w.cleanupWorker()

	return nil
}

func (w *watcher) Stop() {
	w.stop()
}

func (w *watcher) watch() {
	filter := filters.NewArgs()
	filter.Add("type", "container")

	options := types.EventsOptions{
		Since:   fmt.Sprintf("%d", w.lastValidTimestamp),
		Filters: filter,
	}

	for {
		events, errors := w.client.Events(w.ctx, options)

	WATCH:
		for {
			select {
			case event := <-events:
				logp.Debug("docker", "Got a new docker event: %v", event)
				w.lastValidTimestamp = event.Time

				// Add / update
				if event.Action == "start" || event.Action == "update" {
					filter := filters.NewArgs()
					filter.Add("id", event.Actor.ID)

					containers, err := w.listContainers(types.ContainerListOptions{
						Filters: filter,
					})
					if err != nil || len(containers) != 1 {
						logp.Err("Error getting container info: %v", err)
						continue
					}
					container := containers[0]

					w.Lock()
					w.containers[event.Actor.ID] = container
					if w.shortID {
						w.containers[event.Actor.ID[:shortIDLen]] = container
					}
					// un-delete if it's flagged (in case of update or recreation)
					delete(w.deleted, event.Actor.ID)
					w.Unlock()

					w.bus.Publish(bus.Event{
						"start":     true,
						"container": container,
					})
				}

				// Delete
				if event.Action == "die" {
					container := w.Container(event.Actor.ID)
					if container != nil {
						w.bus.Publish(bus.Event{
							"stop":      true,
							"container": container,
						})
					}

					w.Lock()
					w.deleted[event.Actor.ID] = time.Now()
					w.Unlock()
				}

			case err := <-errors:
				// Restart watch call
				logp.Err("Error watching for docker events: %v", err)
				time.Sleep(1 * time.Second)
				break WATCH

			case <-w.ctx.Done():
				logp.Debug("docker", "Watcher stopped")
				w.stopped.Done()
				return
			}
		}
	}
}

func (w *watcher) listContainers(options types.ContainerListOptions) ([]*Container, error) {
	containers, err := w.client.ContainerList(w.ctx, options)
	if err != nil {
		return nil, err
	}

	var result []*Container
	for _, c := range containers {
		var ipaddresses []string
		for _, net := range c.NetworkSettings.Networks {
			ipaddresses = append(ipaddresses, net.IPAddress)
		}
		result = append(result, &Container{
			ID:          c.ID,
			Name:        c.Names[0][1:], // Strip '/' from container names
			Image:       c.Image,
			Labels:      c.Labels,
			Ports:       c.Ports,
			IPAddresses: ipaddresses,
		})
	}

	return result, nil
}

// Clean up deleted containers after they are not used anymore
func (w *watcher) cleanupWorker() {
	for {
		// Wait a full period
		time.Sleep(w.cleanupTimeout)

		select {
		case <-w.ctx.Done():
			w.stopped.Done()
			return
		default:
			// Check entries for timeout
			var toDelete []string
			timeout := time.Now().Add(-w.cleanupTimeout)
			w.RLock()
			for key, lastSeen := range w.deleted {
				if lastSeen.Before(timeout) {
					logp.Debug("docker", "Removing container %s after cool down timeout", key)
					toDelete = append(toDelete, key)
				}
			}
			w.RUnlock()

			// Delete timed out entries:
			for _, key := range toDelete {
				container := w.Container(key)
				if container != nil {
					w.bus.Publish(bus.Event{
						"delete":    true,
						"container": container,
					})
				}
			}

			w.Lock()
			for _, key := range toDelete {
				delete(w.deleted, key)
				delete(w.containers, key)
				if w.shortID {
					delete(w.containers, key[:shortIDLen])
				}
			}
			w.Unlock()
		}
	}
}

// ListenStart returns a bus listener to receive container started events, with a `container` key holding it
func (w *watcher) ListenStart() bus.Listener {
	return w.bus.Subscribe("start")
}

// ListenStop returns a bus listener to receive container stopped events, with a `container` key holding it
func (w *watcher) ListenStop() bus.Listener {
	return w.bus.Subscribe("stop")
}
