// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/common/backoff"
	"github.com/elastic/beats/libbeat/logp"
)

func mustUUIDV4() uuid.UUID {
	uuid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return uuid
}

// OSSLicense default license to use.
var (
	OSSLicense = &License{
		UUID:   mustUUIDV4().String(),
		Type:   OSS,
		Mode:   OSS,
		Status: Active,
		Features: features{
			Graph:      graph{},
			Logstash:   logstash{},
			ML:         ml{},
			Monitoring: monitoring{},
			Rollup:     rollup{},
			Security:   security{},
			Watcher:    watcher{},
		},
	}
)

// Watcher allows a type to receive a new event when a new license is received.
type Watcher interface {
	OnNewLicense(license License)
	OnManagerStopped()
}

// Fetcher interface implements the mechanism to retrieve a License. Currently we only
// support license coming from the '/_xpack' rest api.
type Fetcher interface {
	Fetch() (*License, error)
}

// Errors returned by the manager.
var (
	ErrWatcherAlreadyExist = errors.New("watcher already exist")
	ErrWatcherDoesntExist  = errors.New("watcher doesn't exist")

	ErrManagerStopped = errors.New("license manager is stopped")
	ErrNoLicenseFound = errors.New("no license found")

	ErrNoElasticsearchConfig = errors.New("no elasticsearch output configuration found, verify your configuration")
)

// Backoff values when the remote cluster is not responding.
var (
	maxBackoff  = 60 * time.Second
	initBackoff = 1 * time.Second
	jitterCap   = 1000 // 1000 milliseconds
)

// Manager keeps tracks of license management, it uses a fetcher usually the ElasticFetcher to
// retrieve a licence from a specific cluster.
//
// Starting the manager will start a go routine to periodically query the license fetcher.
// if an error occur on the fetcher we will retry until we successfully
// receive a new license. During that period we start a grace counter, we assume the license is
// still valid during the grace period, when this period expire we will keep retrying but the previous
// license will be invalidated and we will fallback to the OSS license.
//
// Retrieving the current license:
// - Call the `Get()` on the manager instance.
// - Or register a `Watcher` with the manager to receive the new license and acts on it, you will
// also receive an event when the Manager is stopped.
//
//
// Notes:
// - When the manager is started no license is set by default.
// - When a license is invalidated, we fallback to the OSS License and the watchers get notified.
// - Adding a watcher will automatically send the current license to the newly added watcher if
//   available.
type Manager struct {
	done chan struct{}
	sync.RWMutex
	wg          sync.WaitGroup
	fetcher     Fetcher
	duration    time.Duration
	gracePeriod time.Duration
	license     *License
	watchers    map[Watcher]Watcher
	log         *logp.Logger
}

// New takes an elasticsearch client and wraps it into a fetcher, the fetch will handle the JSON
// and response code from the cluster.
func New(client esclient, duration time.Duration, gracePeriod time.Duration) *Manager {
	fetcher := NewElasticFetcher(client)
	return NewWithFetcher(fetcher, duration, gracePeriod)
}

// NewWithFetcher takes a fetcher and return a license manager.
func NewWithFetcher(fetcher Fetcher, duration time.Duration, gracePeriod time.Duration) *Manager {
	m := &Manager{
		fetcher:     fetcher,
		duration:    duration,
		log:         logp.NewLogger("license-manager"),
		done:        make(chan struct{}),
		gracePeriod: gracePeriod,
		watchers:    make(map[Watcher]Watcher),
	}

	return m
}

// AddWatcher register a new watcher to receive events when the license is retrieved or when the manager
// is closed.
func (m *Manager) AddWatcher(watcher Watcher) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.watchers[watcher]; ok {
		return ErrWatcherAlreadyExist
	}

	m.watchers[watcher] = watcher

	// when we register a new watchers send the current license unless we did not retrieve it.
	if m.license != nil {
		watcher.OnNewLicense(*m.license)
	}
	return nil
}

// RemoveWatcher removes the watcher if it exist or return an error.
func (m *Manager) RemoveWatcher(watcher Watcher) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.watchers[watcher]; ok {
		delete(m.watchers, watcher)
		return nil
	}
	return ErrWatcherDoesntExist
}

// Get return the current active license, it can return an error if the manager is stopped or when
// there is no license in the manager, Instead of querying the Manager it is easier to register a
// watcher to listen to license change.
func (m *Manager) Get() (*License, error) {
	m.Lock()
	defer m.Unlock()

	select {
	case <-m.done:
		return nil, ErrManagerStopped
	default:
		if m.license == nil {
			return nil, ErrNoLicenseFound
		}
		return m.license, nil
	}
}

// Start starts the License manager, the manager will start a go routine to periodically
// retrieve the license from the fetcher.
func (m *Manager) Start() {
	// First update should be in sync at startup to ensure a
	// consistent state.
	m.log.Info("License manager started, retrieving initial license")
	m.wg.Add(1)
	go m.worker()
}

// Stop terminates the license manager, the go routine will be stopped and the cached license will
// be removed and no more checks can be done on the manager.
func (m *Manager) Stop() {
	select {
	case <-m.done:
		m.log.Error("License manager already stopped")
	default:
	}

	defer m.log.Info("License manager stopped")
	defer m.notify(func(w Watcher) {
		w.OnManagerStopped()
	})

	// stop the periodic check license and wait for it to complete
	close(m.done)
	m.wg.Wait()

	// invalidate current license
	m.Lock()
	defer m.Unlock()
	m.license = nil
}

func (m *Manager) notify(op func(Watcher)) {
	m.RLock()
	defer m.RUnlock()

	if len(m.watchers) == 0 {
		m.log.Debugf("No watchers configured")
		return
	}

	m.log.Debugf("Notifying %d watchers", len(m.watchers))
	for _, w := range m.watchers {
		op(w)
	}
}

func (m *Manager) worker() {
	defer m.wg.Done()
	m.log.Debugf("Starting periodic license check, refresh: %s grace: %s ", m.duration, m.gracePeriod)
	defer m.log.Debug("Periodic license check is stopped")

	jitter := rand.Intn(jitterCap)

	// Add some jitter to space requests from a large fleet of beats.
	select {
	case <-time.After(time.Duration(jitter) * time.Millisecond):
	}

	// eager initial check.
	m.update()

	// periodically checks license.
	for {
		select {
		case <-m.done:
			return
		case <-time.After(m.duration):
			m.log.Debug("License is too old, updating, grace period: %s", m.gracePeriod)
			m.update()
		}
	}
}

func (m *Manager) update() {
	backoff := backoff.NewEqualJitterBackoff(m.done, initBackoff, maxBackoff)
	startedAt := time.Now()
	for {
		select {
		case <-m.done:
			return
		default:
			license, err := m.fetcher.Fetch()
			if err != nil {
				m.log.Infof("Cannot retrieve license, retrying later, error: %+v", err)

				// check if the license is still in the grace period.
				// permit some operations if the license could not be checked
				// right away. This is to smooth any networks problems.
				if grace := time.Now().Sub(startedAt); grace > m.gracePeriod {
					m.log.Info("Grace period expired, invalidating license")
					m.invalidate()
				} else {
					m.log.Debugf("License is too old, grace time remaining: %s", m.gracePeriod-grace)
				}

				backoff.Wait()
				continue
			}

			// we have a valid license, notify watchers and sleep until next check.
			m.log.Infow(
				"Valid license retrieved",
				"license mode",
				license.Get(),
				"type",
				license.Type,
				"status",
				license.Status,
			)
			m.saveAndNotify(license)
			return
		}
	}
}

func (m *Manager) saveAndNotify(license *License) {
	if !m.save(license) {
		return
	}

	l := *license
	m.notify(func(w Watcher) {
		w.OnNewLicense(l)
	})
}

func (m *Manager) save(license *License) bool {
	m.Lock()
	defer m.Unlock()

	// License didn't change no need to notify watchers.
	if m.license != nil && m.license.EqualTo(license) {
		return false
	}
	defer m.log.Debug("License information updated")

	m.license = license
	return true
}

func (m *Manager) invalidate() {
	defer m.log.Debug("Invalidate cached license, fallback to OSS")
	m.saveAndNotify(OSSLicense)
}

// WaitForLicense transforms the async manager into a sync check, this is useful if you want
// to block you application until you have received an initial license from the cluster, the manager
// is not affected and will stay asynchronous.
func WaitForLicense(ctx context.Context, log *logp.Logger, manager *Manager, checks ...CheckFunc) (err error) {
	log.Info("Waiting on synchronous license check")
	received := make(chan struct{})
	callback := CallbackWatcher{New: func(license License) {
		log.Debug("Validating license")
		if !Validate(log, license, checks...) {
			err = errors.New("invalid license")
		}
		close(received)
		log.Infof("License is valid, mode: %s", license.Get())
	}}

	if err := manager.AddWatcher(&callback); err != nil {
		return err
	}
	defer manager.RemoveWatcher(&callback)

	select {
	case <-ctx.Done():
		return fmt.Errorf("license check was interrupted")
	case <-received:
	}

	return err
}
