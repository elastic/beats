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

package registrar

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type atomicSpy struct{ n atomic.Int64 }

func (s *atomicSpy) Published(n int) bool {
	s.n.Add(int64(n))
	return true
}

func (s *atomicSpy) count() int { return int(s.n.Load()) }

var errInjectedWrite = errors.New("injected write failure")

type failingBackend struct {
	backend.Registry
	fail *atomic.Bool
}

func (b failingBackend) Access(name string) (backend.Store, error) {
	st, err := b.Registry.Access(name)
	if err != nil {
		return nil, err
	}
	return failingStore{Store: st, fail: b.fail}, nil
}

type failingStore struct {
	backend.Store
	fail *atomic.Bool
}

func (s failingStore) Set(key string, value any) error {
	if s.fail.Load() {
		return errInjectedWrite
	}
	return s.Store.Set(key, value)
}

func stopWithin(t *testing.T, r *Registrar, d time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		r.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatal("Stop did not return in time")
	}
}

func inMemoryOffset(r *Registrar, id string) int64 {
	for _, st := range r.GetStates() {
		if st.Id == id {
			return st.Offset
		}
	}
	return -1
}

func fileState(i int, offset int64) file.State {
	return file.State{
		Id:             fmt.Sprintf("native::%d-1", i),
		IdentifierName: "native",
		Source:         fmt.Sprintf("/var/log/%d.log", i),
		Offset:         offset,
		TTL:            -1,
		FileStateOS:    testStateOS(uint64(i)), //nolint:gosec // test identities are small positive integers
	}
}

func TestSharedRegistrarAcksAndStopsWhenWriteFails(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	fail := &atomic.Bool{}
	memBackend := storetest.NewMemoryStoreBackend()
	stateStore := &testStateStore{registry: statestore.NewRegistry(failingBackend{Registry: memBackend, fail: fail})}

	spy := &atomicSpy{}
	r, err := New(stateStore, spy, 0, logger)
	require.NoError(t, err, "registrar must be created")
	require.NoError(t, r.Start(), "registrar must start")

	failsBefore := registryFails.Get()
	successBefore := registrySuccess.Get()

	fail.Store(true)
	r.Channel <- []file.State{fileState(1, 5000)}
	require.Eventually(t, func() bool { return spy.count() == 1 }, 5*time.Second, 10*time.Millisecond, "owner must still be acked after a failed write so shutdown accounting completes; at-least-once tolerates the replay")
	assert.Greater(t, registryFails.Get(), failsBefore, "the failed write must be counted as a failure")
	assert.Equal(t, successBefore, registrySuccess.Get(), "a failed write must not be counted as a success")
	_, persisted := memBackend.Stores[testStoreName].Table[fileStatePrefix+"native::1-1"]
	assert.False(t, persisted, "the state must not have reached the backend while writes fail")
	assert.EqualValues(t, 5000, inMemoryOffset(r, "native::1-1"), "the shared table must keep the update so the next commit retries it")

	fail.Store(false)
	r.Channel <- []file.State{fileState(2, 7000)}
	stopWithin(t, r, 5*time.Second)

	assert.Equal(t, 2, spy.count(), "owner must be acked once per update")
	assert.EqualValues(t, 5000, readState(t, stateStore, "native::1-1").Offset, "the earlier update must be persisted by the first successful commit")
	assert.EqualValues(t, 7000, readState(t, stateStore, "native::2-1").Offset, "the later update must be persisted")
	assert.Greater(t, registrySuccess.Get(), successBefore, "a successful write must be counted as a success")
}

func TestSharedRegistrarStopOfOneOwnerFlushesAndAcksTheOtherOnce(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, fileB := seededStore(t)

	spyA, spyB := &atomicSpy{}, &atomicSpy{}
	regA, err := New(stateStore, spyA, time.Hour, logger)
	require.NoError(t, err, "registrar A must be created")
	regB, err := New(stateStore, spyB, time.Hour, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	require.NoError(t, regB.Start(), "registrar B must start")

	fileB.Offset = 7000
	regB.Channel <- []file.State{fileB, fileB, fileB}
	require.Eventually(t, func() bool { return inMemoryOffset(regB, fileB.Id) == 7000 }, 2*time.Second, 10*time.Millisecond, "B's update must be applied to the shared table")
	assert.EqualValues(t, 100, readState(t, stateStore, fileB.Id).Offset, "B's update must not be committed before a flush")
	assert.Equal(t, 0, spyB.count(), "B must not be acked before a commit")

	fileA.Offset = 5000
	regA.Channel <- []file.State{fileA}
	stopWithin(t, regA, 5*time.Second)

	assert.EqualValues(t, 7000, readState(t, stateStore, fileB.Id).Offset, "A's shutdown flush must persist B's pending state")
	assert.Equal(t, 3, spyB.count(), "B must be acked exactly once for its three pending states when A's flush committed them")
	assert.Equal(t, 1, spyA.count(), "A must be acked for its own state")

	fileB.Offset = 9000
	regB.Channel <- []file.State{fileB, fileB}
	stopWithin(t, regB, 5*time.Second)

	assert.EqualValues(t, 9000, readState(t, stateStore, fileB.Id).Offset, "B's later update must be persisted by its own stop")
	assert.Equal(t, 5, spyB.count(), "B must be acked for its two later states only once more")
	assert.Equal(t, 0, sharedRegistrars.Len(), "shared registrar must be released")
}

func TestSharedRegistrarReacquiredAfterLastOwnerStops(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, _ := seededStore(t)

	regA, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar A must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	first := regA.shared
	fileA.Offset = 5000
	regA.Channel <- []file.State{fileA}
	stopWithin(t, regA, 5*time.Second)
	require.Equal(t, 0, sharedRegistrars.Len(), "shared registrar must be released after the only owner stops")

	regB, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regB.Start(), "registrar B must start after a full release")
	assert.NotSame(t, first, regB.shared, "a fresh shared registrar must be created")

	var loaded *file.State
	for _, st := range regB.GetStates() {
		if st.Id == fileA.Id {
			loaded = &st
		}
	}
	require.NotNil(t, loaded, "the fresh registrar must load a.log from the store")
	assert.EqualValues(t, 5000, loaded.Offset, "the fresh registrar must see the offset persisted by the previous owner")
	stopWithin(t, regB, 5*time.Second)
}

func TestSharedRegistrarUsesShortestFlushTimeout(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, fileB := seededStore(t)

	spyA := &atomicSpy{}
	regA, err := New(stateStore, spyA, time.Hour, logger)
	require.NoError(t, err, "registrar A must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	regB, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regB.Start(), "registrar B must start")

	fileA.Offset = 5000
	regA.Channel <- []file.State{fileA}
	assert.Eventually(t, func() bool { return spyA.count() == 1 }, 5*time.Second, 10*time.Millisecond, "A must be committed and acked immediately once an owner with no flush timeout has joined")
	assert.EqualValues(t, 5000, readState(t, stateStore, fileA.Id).Offset, "A's state must be persisted without waiting for its own hour-long timeout")

	stopWithin(t, regA, 5*time.Second)
	fileB.Offset = 7000
	regB.Channel <- []file.State{fileB}
	stopWithin(t, regB, 5*time.Second)
	assert.EqualValues(t, 7000, readState(t, stateStore, fileB.Id).Offset, "B's state must be persisted")
}

func TestSharedRegistrarConcurrentOwnersStress(t *testing.T) {
	const owners, updates = 8, 200
	logger := logptest.NewTestingLogger(t, "")
	memBackend := storetest.NewMemoryStoreBackend()
	stateStore := &testStateStore{registry: statestore.NewRegistry(memBackend)}

	regs := make([]*Registrar, owners)
	spies := make([]*atomicSpy, owners)
	for i := range regs {
		spies[i] = &atomicSpy{}
		timeout := time.Duration(i%3) * 5 * time.Millisecond
		r, err := New(stateStore, spies[i], timeout, logger)
		require.NoError(t, err, "registrar %d must be created", i)
		require.NoError(t, r.Start(), "registrar %d must start", i)
		regs[i] = r
	}

	var wg sync.WaitGroup
	for i, r := range regs {
		wg.Go(func() {
			for n := 1; n <= updates; n++ {
				r.Channel <- []file.State{fileState(i+1, int64(n*100))}
			}
			r.Stop()
		})
	}
	wg.Wait()

	for i := range regs {
		assert.Equal(t, updates, spies[i].count(), "owner %d must be acked exactly once per update", i)
		assert.EqualValues(t, updates*100, readState(t, stateStore, fileState(i+1, 0).Id).Offset, "owner %d's final offset must be persisted", i)
	}
	assert.Equal(t, 0, sharedRegistrars.Len(), "shared registrar must be released after all owners stop")
}

type accessFailBackend struct {
	backend.Registry
	fail *atomic.Bool
}

func (b accessFailBackend) Access(name string) (backend.Store, error) {
	if b.fail.Load() {
		return nil, errInjectedWrite
	}
	return b.Registry.Access(name)
}

func TestRegistrarStartCanBeRetriedAfterFailedAcquire(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	fail := &atomic.Bool{}
	fail.Store(true)
	memBackend := storetest.NewMemoryStoreBackend()
	stateStore := &testStateStore{registry: statestore.NewRegistry(accessFailBackend{Registry: memBackend, fail: fail})}

	r, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar must be created")
	require.Error(t, r.Start(), "Start must fail while the store cannot be opened")
	assert.Equal(t, 0, sharedRegistrars.Len(), "a failed Start must leave no shared registrar behind")

	fail.Store(false)
	require.NoError(t, r.Start(), "Start must succeed on retry once the store opens")
	r.Channel <- []file.State{fileState(1, 5000)}
	stopWithin(t, r, 5*time.Second)
	assert.EqualValues(t, 5000, readState(t, stateStore, "native::1-1").Offset, "the retried handle must persist state normally")
	assert.Equal(t, 0, sharedRegistrars.Len(), "Stop must release the reference taken by the successful Start")
}

type unhashableLogger struct {
	_ []int
}

func (unhashableLogger) Published(int) bool { return true }

func TestSharedRegistrarMarksLeavingOwnersStatesFinished(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, fileB := seededStore(t)

	regA, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar A must be created")
	regB, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	require.NoError(t, regB.Start(), "registrar B must start")

	fileA.Offset, fileA.Finished = 5000, false
	fileB.Offset, fileB.Finished = 7000, false
	regA.Channel <- []file.State{fileA}
	regB.Channel <- []file.State{fileB}
	require.Eventually(t, func() bool {
		return inMemoryOffset(regA, fileA.Id) == 5000 && inMemoryOffset(regA, fileB.Id) == 7000
	}, 5*time.Second, 10*time.Millisecond, "both owners' states must be applied")

	stopWithin(t, regB, 5*time.Second)

	stateOf := func(id string) file.State {
		for _, st := range regA.GetStates() {
			if st.Id == id {
				return st
			}
		}
		return file.State{}
	}
	assert.True(t, stateOf(fileB.Id).Finished, "the leaving owner's file must be marked finished so a restarted input can claim it")
	assert.EqualValues(t, 7000, stateOf(fileB.Id).Offset, "marking finished must not lose the offset")
	assert.False(t, stateOf(fileA.Id).Finished, "a still-running owner's file must stay unfinished")
	assert.EqualValues(t, 7000, readState(t, stateStore, fileB.Id).Offset, "the leaving flush must persist the offset")

	stopWithin(t, regA, 5*time.Second)
}

func TestRegistrarStartGuards(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, _, _ := seededStore(t)

	stopped, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar must be created")
	stopped.Stop()
	assert.Error(t, stopped.Start(), "Start after Stop must fail instead of leaking a shared registrar reference")
	assert.Equal(t, 0, sharedRegistrars.Len(), "a refused Start must not acquire the shared registrar")

	twice, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar must be created")
	require.NoError(t, twice.Start(), "first Start must succeed")
	assert.Error(t, twice.Start(), "second Start must fail instead of acquiring a second reference")
	stopWithin(t, twice, 5*time.Second)
	assert.Equal(t, 0, sharedRegistrars.Len(), "Stop must release the single reference the first Start took")
}

func TestSharedRegistrarAcceptsUnhashableSuccessLogger(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, _ := seededStore(t)

	r, err := New(stateStore, unhashableLogger{}, 0, logger)
	require.NoError(t, err, "registrar must be created")
	require.NoError(t, r.Start(), "registrar must start")
	fileA.Offset = 5000
	r.Channel <- []file.State{fileA}
	stopWithin(t, r, 5*time.Second)
	assert.EqualValues(t, 5000, readState(t, stateStore, fileA.Id).Offset, "a value-type success logger must not break commits")
}

func TestSharedRegistrarLoweredTimeoutFlushesArmedBatch(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, _ := seededStore(t)

	spyA := &atomicSpy{}
	regA, err := New(stateStore, spyA, time.Hour, logger)
	require.NoError(t, err, "registrar A must be created")
	require.NoError(t, regA.Start(), "registrar A must start")

	fileA.Offset = 5000
	regA.Channel <- []file.State{fileA}
	require.Eventually(t, func() bool { return inMemoryOffset(regA, fileA.Id) == 5000 }, 5*time.Second, 10*time.Millisecond, "A's batch must be applied and waiting on its hour-long timer")
	assert.Equal(t, 0, spyA.count(), "A must not be acked while its timer is armed")

	regB, err := New(stateStore, &atomicSpy{}, 0, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regB.Start(), "registrar B must start")

	assert.Equal(t, 1, spyA.count(), "joining with a shorter timeout must have committed A's armed batch before Start returned")
	assert.EqualValues(t, 5000, readState(t, stateStore, fileA.Id).Offset, "A's batch must be persisted by the join flush")

	stopWithin(t, regA, 5*time.Second)
	stopWithin(t, regB, 5*time.Second)
}
