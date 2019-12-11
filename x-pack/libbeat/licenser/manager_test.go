// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
)

type message struct {
	license *License
	err     error
}

type mockFetcher struct {
	sync.Mutex
	bus  chan message
	last *message
}

func newMockFetcher() *mockFetcher {
	return &mockFetcher{bus: make(chan message, 1)}
}

func (m *mockFetcher) Fetch() (*License, error) {
	m.Lock()
	defer m.Unlock()
	for {
		select {
		case message := <-m.bus:
			m.last = &message

			// assume other calls to receive the same value,
			// until we change it.
			return message.license, message.err
		default:
			if m.last != nil {
				return m.last.license, m.last.err
			}
			continue
		}
	}
}

func (m *mockFetcher) Insert(license *License, err error) {
	m.bus <- message{license: license, err: err}
}

func (m *mockFetcher) Close() {
	close(m.bus)
}

func TestRetrieveLicense(t *testing.T) {
	i := &License{
		UUID:   mustUUIDV4().String(),
		Type:   Basic,
		Mode:   Basic,
		Status: Active,
	}
	mock := newMockFetcher()
	mock.Insert(i, nil)
	defer mock.Close()

	t.Run("return an error if the manager is stopped", func(t *testing.T) {
		m := NewWithFetcher(mock, time.Duration(2*time.Second), time.Duration(1*time.Second))
		m.Start()
		m.Stop()

		_, err := m.Get()

		assert.Error(t, ErrManagerStopped, err)
	})

	t.Run("at startup when no license is retrieved return an error", func(t *testing.T) {
		mck := newMockFetcher()
		mck.Insert(nil, errors.New("not found"))
		defer mck.Close()

		m := NewWithFetcher(mck, time.Duration(2*time.Second), time.Duration(1*time.Second))
		m.Start()
		defer m.Stop()
		_, err := m.Get()

		assert.Error(t, ErrNoLicenseFound, err)
	})

	t.Run("at startup", func(t *testing.T) {
		m := NewWithFetcher(mock, time.Duration(2*time.Second), time.Duration(1*time.Second))
		m.Start()
		defer m.Stop()

		// Lets us find the first license.
		time.Sleep(1 * time.Second)
		_, err := m.Get()

		assert.NoError(t, err)
	})

	t.Run("periodically", func(t *testing.T) {
		period := time.Duration(1)
		m := NewWithFetcher(mock, period, time.Duration(5*time.Second))

		m.Start()
		defer m.Stop()

		// Lets us find the first license.
		time.Sleep(1 * time.Second)

		l, err := m.Get()
		if !assert.NoError(t, err) {
			return
		}
		if !assert.True(t, l.Is(Basic)) {
			return
		}

		i := &License{
			UUID:   mustUUIDV4().String(),
			Type:   Platinum,
			Mode:   Platinum,
			Status: Active,
		}
		mock.Insert(i, nil)

		select {
		case <-time.After(time.Duration(1 * time.Second)):
			l, err := m.Get()
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, l.Is(Platinum))
		}
	})
}

func TestWatcher(t *testing.T) {
	i := &License{
		UUID:   mustUUIDV4().String(),
		Type:   Basic,
		Mode:   Basic,
		Status: Active,
	}

	t.Run("watcher must be uniquely registered", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(2*time.Second), time.Duration(1*time.Second))

		m.Start()
		defer m.Stop()

		w := CallbackWatcher{New: func(license License) {}}

		err := m.AddWatcher(&w)
		if assert.NoError(t, err) {
			return
		}
		defer m.RemoveWatcher(&w)

		err = m.AddWatcher(&w)
		assert.Error(t, ErrWatcherAlreadyExist, err)
	})

	t.Run("cannot remove non existing watcher", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(2*time.Second), time.Duration(1*time.Second))

		m.Start()
		defer m.Stop()

		w := CallbackWatcher{New: func(license License) {}}

		err := m.RemoveWatcher(&w)

		assert.Error(t, ErrWatcherDoesntExist, err)
	})

	t.Run("adding a watcher trigger a a new license callback", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(2*time.Second), time.Duration(1*time.Second))

		m.Start()
		defer m.Stop()

		chanLicense := make(chan License)
		defer close(chanLicense)

		w := CallbackWatcher{
			New: func(license License) {
				chanLicense <- license
			},
		}

		m.AddWatcher(&w)
		defer m.RemoveWatcher(&w)

		select {
		case license := <-chanLicense:
			assert.Equal(t, Basic, license.Get())
		}
	})

	t.Run("periodically trigger a new license callback when the license change", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(1), time.Duration(1*time.Second))

		m.Start()
		defer m.Stop()

		chanLicense := make(chan License)
		defer close(chanLicense)

		w := CallbackWatcher{
			New: func(license License) {
				chanLicense <- license
			},
		}

		m.AddWatcher(&w)
		defer m.RemoveWatcher(&w)

		c := 0
		for {
			select {
			case license := <-chanLicense:
				if c == 0 {
					assert.Equal(t, Basic, license.Get())
					mock.Insert(&License{
						UUID:   mustUUIDV4().String(),
						Type:   Platinum,
						Mode:   Platinum,
						Status: Active,
					}, nil)
					c++
					continue
				}
				assert.Equal(t, Platinum, license.Get())
				return
			}
		}
	})

	t.Run("trigger OnManagerStopped when the manager is stopped", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(1), time.Duration(1*time.Second))
		m.Start()

		var wg sync.WaitGroup

		wg.Add(1)
		w := CallbackWatcher{
			Stopped: func() {
				wg.Done()
			},
		}

		m.AddWatcher(&w)
		defer m.RemoveWatcher(&w)

		m.Stop()

		wg.Wait()
	})
}

func TestWaitForLicense(t *testing.T) {
	i := &License{
		UUID:   mustUUIDV4().String(),
		Type:   Basic,
		Mode:   Basic,
		Status: Active,
	}

	t.Run("when license is available and valid", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(1), time.Duration(1*time.Second))

		m.Start()
		defer m.Stop()

		err := WaitForLicense(context.Background(), logp.NewLogger(""), m, CheckBasic)
		assert.NoError(t, err)
	})

	t.Run("when license is available and not valid", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(1), time.Duration(1*time.Second))

		m.Start()
		defer m.Stop()

		err := WaitForLicense(context.Background(), logp.NewLogger(""), m, CheckLicenseCover(Platinum))
		assert.Error(t, err)
	})

	t.Run("when license is not available we can still interrupt", func(t *testing.T) {
		mock := newMockFetcher()
		mock.Insert(i, nil)
		defer mock.Close()

		m := NewWithFetcher(mock, time.Duration(1), time.Duration(1*time.Second))

		m.Start()
		defer m.Stop()

		ctx, cancel := context.WithCancel(context.Background())
		executed := make(chan struct{})
		go func() {
			err := WaitForLicense(ctx, logp.NewLogger(""), m, CheckLicenseCover(Platinum))
			assert.Error(t, err)
			close(executed)
		}()
		cancel()
		<-executed
	})
}
