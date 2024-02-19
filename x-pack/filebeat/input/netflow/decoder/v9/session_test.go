// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"io"
	"log"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/test"
)

var logger = log.New(io.Discard, "", 0)

func makeSessionKey(t testing.TB, ipPortPair string, domain uint32) SessionKey {
	return MakeSessionKey(test.MakeAddress(t, ipPortPair), domain, false)
}

func TestSessionMap_GetOrCreate(t *testing.T) {
	t.Run("consistent behavior", func(t *testing.T) {
		sm := NewSessionMap(logger, nil)

		// Session is created
		s1 := sm.GetOrCreate(makeSessionKey(t, "127.0.0.1:1234", 42))
		assert.NotNil(t, s1)

		// Get a different Session
		s2 := sm.GetOrCreate(makeSessionKey(t, "127.0.0.1:1235", 42))
		assert.NotNil(t, s1)
		assert.False(t, s1 == s2)

		// Get a different Session for diff IP same port
		s3 := sm.GetOrCreate(makeSessionKey(t, "127.0.0.2:1234", 42))
		assert.NotNil(t, s3)
		assert.False(t, s1 == s3 || s2 == s3)

		// Get a different Session for same IP diff port
		s4 := sm.GetOrCreate(makeSessionKey(t, "127.0.0.1:1236", 42))
		assert.NotNil(t, s4)
		assert.False(t, s1 == s4 || s2 == s4 || s3 == s4)

		// Get same Session for same params
		s1b := sm.GetOrCreate(makeSessionKey(t, "127.0.0.1:1234", 42))
		assert.NotNil(t, s1b)
		assert.True(t, s1 == s1b)

		// Get diff Session same source different observation domain
		s2b := sm.GetOrCreate(makeSessionKey(t, "127.0.0.1:1235", 43))
		assert.NotNil(t, s2b)
		assert.False(t, s2 == s2b)
	})
	t.Run("parallel", func(t *testing.T) {
		// Goroutines should observe the same session when created in parallel
		sm := NewSessionMap(logger, nil)
		key := makeSessionKey(t, "127.0.0.1:9995", 42)
		const N = 8
		const Iters = 200
		C := make(chan *SessionState, N*Iters)
		wg := sync.WaitGroup{}
		wg.Add(N)
		for i := 0; i < N; i++ {
			go func() {
				last := sm.GetOrCreate(key)
				for iter := 0; iter < Iters; iter++ {
					s := sm.GetOrCreate(key)
					if last != s {
						C <- last
						last = s
					}
				}
				C <- last
				wg.Done()
			}()
		}
		wg.Wait()
		if !assert.NotEmpty(t, C) {
			return
		}
		base := <-C
		close(C)
		for s := range C {
			if !assert.True(t, s == base) {
				return
			}
		}
	})
}

func testTemplate(id uint16) *template.Template {
	return &template.Template{
		ID: id,
	}
}

func TestSessionState(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	t.Run("create and get", func(t *testing.T) {
		s := NewSession(logger)
		t1 := testTemplate(1)
		s.AddTemplate(t1)
		t2 := s.GetTemplate(1)
		assert.True(t, t1 == t2)
	})
	t.Run("update", func(t *testing.T) {
		s := NewSession(logger)
		t1 := testTemplate(1)
		s.AddTemplate(t1)

		t2 := testTemplate(2)
		s.AddTemplate(t2)

		t1c := s.GetTemplate(1)
		assert.True(t, t1 == t1c)

		t2c := s.GetTemplate(2)
		assert.True(t, t2 == t2c)

		t1b := testTemplate(1)
		s.AddTemplate(t1b)

		t1c = s.GetTemplate(1)
		assert.False(t, t1 == t1c)
		assert.True(t, t1b == t1b)
	})
}

func TestSessionMap_Cleanup(t *testing.T) {
	sm := NewSessionMap(logger, nil)

	// Session is created
	k1 := makeSessionKey(t, "127.0.0.1:1234", 1)
	s1 := sm.GetOrCreate(k1)
	assert.NotNil(t, s1)

	sm.cleanup()

	// After a cleanup, first session still exists
	assert.Len(t, sm.Sessions, 1)

	// Add new session
	k2 := makeSessionKey(t, "127.0.0.1:1234", 2)
	s2 := sm.GetOrCreate(k2)
	assert.NotNil(t, s2)
	assert.Len(t, sm.Sessions, 2)

	// After a new cleanup, s1 is removed because it was not accessed
	// since the last cleanup.
	sm.cleanup()
	assert.Len(t, sm.Sessions, 1)

	_, found := sm.Sessions[k1]
	assert.False(t, found)

	// s2 is still there
	_, found = sm.Sessions[k2]
	assert.True(t, found)

	// Access s2 again
	sm.GetOrCreate(k2)

	// Cleanup should keep s2 because it has been used since the last cleanup
	sm.cleanup()

	assert.Len(t, sm.Sessions, 1)
	s2b, found := sm.Sessions[k2]
	assert.True(t, found)
	assert.True(t, s2 == s2b)

	sm.cleanup()
	assert.Empty(t, sm.Sessions)
}

func TestSessionMap_CleanupLoop(t *testing.T) {
	timeout := time.Millisecond * 100
	sm := NewSessionMap(log.New(io.Discard, "", 0), nil)
	key := makeSessionKey(t, "127.0.0.1:1", 42)
	s := sm.GetOrCreate(key)

	done := make(chan struct{})
	go sm.CleanupLoop(timeout, done)

	for found := true; found; {
		sm.mutex.RLock()
		_, found = sm.Sessions[key]
		sm.mutex.RUnlock()
	}
	close(done)
	s2 := sm.GetOrCreate(key)
	assert.True(t, s != s2)
	time.Sleep(timeout * 2)
	s3 := sm.GetOrCreate(key)
	assert.True(t, s2 == s3)
}

func TestTemplateExpiration(t *testing.T) {
	s := NewSession(logger)
	assert.Nil(t, s.GetTemplate(256))
	assert.Nil(t, s.GetTemplate(257))
	s.AddTemplate(testTemplate(256))
	s.AddTemplate(testTemplate(257))

	s.ExpireTemplates()

	assert.NotNil(t, s.GetTemplate(256))
	_, found := s.Templates[TemplateKey(257)]
	assert.True(t, found)

	s.ExpireTemplates()

	_, found = s.Templates[TemplateKey(256)]
	assert.True(t, found)

	assert.Nil(t, s.GetTemplate(257))

	s.ExpireTemplates()

	assert.Nil(t, s.GetTemplate(256))
}

func TestSessionCheckReset(t *testing.T) {
	for _, testCase := range []struct {
		title         string
		current, next uint32
		reset         bool
	}{
		{
			title:   "Regular advance",
			current: 12345,
			next:    12385,
			reset:   false,
		},
		{
			title:   "Out of order packet",
			current: 12388,
			next:    12345,
			reset:   false,
		},
		{
			title:   "Actual reset",
			current: 12345,
			next:    9,
			reset:   true,
		},
		{
			title:   "32-bit Wrap around",
			current: math.MaxUint32,
			next:    9,
			reset:   false,
		},
		{
			title:   "Non-sequential stream",
			current: 12345,
			next:    78910,
			reset:   true,
		},
	} {
		t.Run(testCase.title, func(t *testing.T) {
			s := NewSession(logger)
			s.lastSequence = testCase.current
			prev, isReset := s.CheckReset(testCase.next)
			assert.Equal(t, prev, testCase.current)
			assert.Equal(t, testCase.reset, isReset)
		})
	}
}
