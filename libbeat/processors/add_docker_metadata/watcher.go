package add_docker_metadata

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

	"github.com/elastic/beats/libbeat/logp"
)

// Select Docker API version
const dockerAPIVersion = "1.22"

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
}

// Container info retrieved by the watcher
type Container struct {
	ID     string
	Name   string
	Image  string
	Labels map[string]string
}

// Client for docker interface
type Client interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error)
}

type WatcherConstructor func(host string, tls *TLSConfig) (Watcher, error)

// NewWatcher returns a watcher running for the given settings
func NewWatcher(host string, tls *TLSConfig) (Watcher, error) {
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

	return NewWatcherWithClient(client, 60*time.Second)
}

func NewWatcherWithClient(client Client, cleanupTimeout time.Duration) (*watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &watcher{
		client:         client,
		ctx:            ctx,
		stop:           cancel,
		containers:     make(map[string]*Container),
		deleted:        make(map[string]time.Time),
		cleanupTimeout: cleanupTimeout,
	}, nil
}

// Container returns the running container with the given ID or nil if unknown
func (w *watcher) Container(ID string) *Container {
	w.RLock()
	container := w.containers[ID]
	_, ok := w.deleted[ID]
	w.RUnlock()

	// Update last access time if it's deleted
	if ok {
		w.Lock()
		w.deleted[ID] = time.Now()
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
		res[k] = v
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
	containers, err := w.client.ContainerList(w.ctx, types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, c := range containers {
		w.containers[c.ID] = &Container{
			ID:     c.ID,
			Name:   c.Names[0][1:], // Strip '/' from container names
			Image:  c.Image,
			Labels: c.Labels,
		}
	}

	w.stopped.Add(2)
	go w.watch()
	go w.cleanupWorker()

	return nil
}

func (w *watcher) Stop() {
	w.stop()
}

func (w *watcher) watch() {
	filters := filters.NewArgs()
	filters.Add("type", "container")

	options := types.EventsOptions{
		Since:   fmt.Sprintf("%d", w.lastValidTimestamp),
		Filters: filters,
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
					name := event.Actor.Attributes["name"]
					image := event.Actor.Attributes["image"]
					delete(event.Actor.Attributes, "name")
					delete(event.Actor.Attributes, "image")

					w.Lock()
					w.containers[event.Actor.ID] = &Container{
						ID:     event.Actor.ID,
						Name:   name,
						Image:  image,
						Labels: event.Actor.Attributes,
					}

					// un-delete if it's flagged (in case of update or recreation)
					delete(w.deleted, event.Actor.ID)
					w.Unlock()
				}

				// Delete
				if event.Action == "die" {
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
					logp.Debug("docker", "Removing container %s after cool down timeout")
					toDelete = append(toDelete, key)
				}
			}
			w.RUnlock()

			// Delete timed out entries:
			w.Lock()
			for _, key := range toDelete {
				delete(w.deleted, key)
				delete(w.containers, key)
			}
			w.Unlock()
		}
	}
}
