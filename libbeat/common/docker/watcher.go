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
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-connections/tlsconfig"

	"github.com/menderesk/beats/v7/libbeat/common/bus"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

// Select Docker API version
const (
	shortIDLen                         = 12
	dockerRequestTimeout               = 10 * time.Second
	dockerEventsWatchPityTimerInterval = 10 * time.Second
	dockerEventsWatchPityTimerTimeout  = 10 * time.Minute
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
	log            *logp.Logger
	client         Client
	ctx            context.Context
	stop           context.CancelFunc
	containers     map[string]*Container
	deleted        map[string]time.Time // deleted annotations key -> last access time
	cleanupTimeout time.Duration
	clock          clock
	stopped        sync.WaitGroup
	bus            bus.Bus
	shortID        bool // whether to store short ID in "containers" too
}

// clock is an interface used to provide mocked time on testing
type clock interface {
	Now() time.Time
}

// systemClock implements the clock interface using the system clock via the time package
type systemClock struct{}

// Now returns the current time
func (*systemClock) Now() time.Time { return time.Now() }

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
	ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error)
	Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error)
}

// WatcherConstructor represent a function that creates a new Watcher from giving parameters
type WatcherConstructor func(logp *logp.Logger, host string, tls *TLSConfig, storeShortID bool) (Watcher, error)

// NewWatcher returns a watcher running for the given settings
func NewWatcher(log *logp.Logger, host string, tls *TLSConfig, storeShortID bool) (Watcher, error) {
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

	client, err := NewClient(host, httpClient, nil)
	if err != nil {
		return nil, err
	}

	// Extra check to confirm that Docker is available
	_, err = client.Info(context.Background())
	if err != nil {
		client.Close()
		return nil, err
	}

	return NewWatcherWithClient(log, client, 60*time.Second, storeShortID)
}

// NewWatcherWithClient creates a new Watcher from a given Docker client
func NewWatcherWithClient(log *logp.Logger, client Client, cleanupTimeout time.Duration, storeShortID bool) (Watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &watcher{
		log:            log,
		client:         client,
		ctx:            ctx,
		stop:           cancel,
		containers:     make(map[string]*Container),
		deleted:        make(map[string]time.Time),
		cleanupTimeout: cleanupTimeout,
		bus:            bus.New(log, "docker"),
		shortID:        storeShortID,
		clock:          &systemClock{},
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
		w.deleted[container.ID] = w.clock.Now()
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
	w.log.Debug("Start docker containers scanner")

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
	w.stopped.Wait()
}

func (w *watcher) watch() {
	defer w.stopped.Done()

	filter := filters.NewArgs()
	filter.Add("type", "container")

	// Ticker to restart the watcher when no events are received after some time.
	tickChan := time.NewTicker(dockerEventsWatchPityTimerInterval)
	defer tickChan.Stop()

	lastValidTimestamp := w.clock.Now()

	watch := func() bool {
		lastReceivedEventTime := w.clock.Now()

		w.log.Debugf("Fetching events since %s", lastValidTimestamp)

		options := types.EventsOptions{
			Since:   lastValidTimestamp.Format(time.RFC3339Nano),
			Filters: filter,
		}

		ctx, cancel := context.WithCancel(w.ctx)
		defer cancel()

		events, errors := w.client.Events(ctx, options)
		for {
			select {
			case event := <-events:
				w.log.Debugf("Got a new docker event: %v", event)
				if event.TimeNano > 0 {
					lastValidTimestamp = time.Unix(0, event.TimeNano)
				} else {
					lastValidTimestamp = time.Unix(event.Time, 0)
				}
				lastReceivedEventTime = w.clock.Now()

				switch event.Action {
				case "start", "update":
					w.containerUpdate(event)
				case "die":
					w.containerDelete(event)
				}
			case err := <-errors:
				switch err {
				case io.EOF:
					// Client disconnected, watch is not done, reconnect
					w.log.Debug("EOF received in events stream, restarting watch call")
				case context.DeadlineExceeded:
					w.log.Debug("Context deadline exceeded for docker request, restarting watch call")
				case context.Canceled:
					// Parent context has been canceled, watch is done.
					return true
				default:
					w.log.Errorf("Error watching for docker events: %+v", err)
				}
				return false
			case <-tickChan.C:
				if time.Since(lastReceivedEventTime) > dockerEventsWatchPityTimerTimeout {
					w.log.Infof("No events received within %s, restarting watch call", dockerEventsWatchPityTimerTimeout)
					return false
				}
			case <-w.ctx.Done():
				w.log.Debug("Watcher stopped")
				return true
			}
		}
	}

	for {
		done := watch()
		if done {
			return
		}
		// Wait before trying to reconnect
		time.Sleep(1 * time.Second)
	}
}

func (w *watcher) containerUpdate(event events.Message) {
	filter := filters.NewArgs()
	filter.Add("id", event.Actor.ID)

	containers, err := w.listContainers(types.ContainerListOptions{
		Filters: filter,
	})
	if err != nil || len(containers) != 1 {
		w.log.Errorf("Error getting container info: %v", err)
		return
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

func (w *watcher) containerDelete(event events.Message) {
	container := w.Container(event.Actor.ID)

	w.Lock()
	w.deleted[event.Actor.ID] = w.clock.Now()
	w.Unlock()

	if container != nil {
		w.bus.Publish(bus.Event{
			"stop":      true,
			"container": container,
		})
	}
}

func (w *watcher) listContainers(options types.ContainerListOptions) ([]*Container, error) {
	log := w.log

	log.Debug("List containers")
	ctx, cancel := context.WithTimeout(w.ctx, dockerRequestTimeout)
	defer cancel()

	containers, err := w.client.ContainerList(ctx, options)
	if err != nil {
		return nil, err
	}

	var result []*Container
	for _, c := range containers {
		var ipaddresses []string
		if c.NetworkSettings != nil {
			// Handle alternate platforms like VMWare's VIC that might not have this data.
			for _, net := range c.NetworkSettings.Networks {
				if net.IPAddress != "" {
					ipaddresses = append(ipaddresses, net.IPAddress)
				}
			}
		}

		// If there are no network interfaces, assume that the container is on host network
		// Inspect the container directly and use the hostname as the IP address in order
		if len(ipaddresses) == 0 {
			log.Debugf("Inspect container %s", c.ID)
			ctx, cancel := context.WithTimeout(w.ctx, dockerRequestTimeout)
			defer cancel()
			info, err := w.client.ContainerInspect(ctx, c.ID)
			if err == nil {
				ipaddresses = append(ipaddresses, info.Config.Hostname)
			} else {
				log.Warnf("unable to inspect container %s due to error %+v", c.ID, err)
			}
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
	defer w.stopped.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		// Wait a full period
		case <-time.After(w.cleanupTimeout):
			w.runCleanup()
		}
	}
}

func (w *watcher) runCleanup() {
	// Check entries for timeout
	var toDelete []string
	timeout := w.clock.Now().Add(-w.cleanupTimeout)
	w.RLock()
	for key, lastSeen := range w.deleted {
		if lastSeen.Before(timeout) {
			w.log.Debugf("Removing container %s after cool down timeout", key)
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

// ListenStart returns a bus listener to receive container started events, with a `container` key holding it
func (w *watcher) ListenStart() bus.Listener {
	return w.bus.Subscribe("start")
}

// ListenStop returns a bus listener to receive container stopped events, with a `container` key holding it
func (w *watcher) ListenStop() bus.Listener {
	return w.bus.Subscribe("stop")
}
