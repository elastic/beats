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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

const overlapWarning = "harvested by more than one receiver"

func overlapWarnings(logs *observer.ObservedLogs) int {
	n := 0
	for _, e := range logs.All() {
		if strings.Contains(e.Message, overlapWarning) {
			n++
		}
	}
	return n
}

func finishedIn(r *Registrar, id string) (finished, found bool) {
	for _, st := range r.GetStates() {
		if st.Id == id {
			return st.Finished, true
		}
	}
	return false, false
}

func startOverlapOwners(t *testing.T) (regA, regB *Registrar, logsA, logsB *observer.ObservedLogs, a, b file.State) {
	t.Helper()
	stateStore, a, b := seededStore(t)
	loggerA, logsA := logptest.NewTestingLoggerWithObserver(t, "")
	loggerB, logsB := logptest.NewTestingLoggerWithObserver(t, "")
	regA, err := New(stateStore, &atomicSpy{}, 0, loggerA)
	require.NoError(t, err, "registrar A must be created")
	regB, err = New(stateStore, &atomicSpy{}, 0, loggerB)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	t.Cleanup(regA.Stop)
	require.NoError(t, regB.Start(), "registrar B must start")
	t.Cleanup(regB.Stop)
	return regA, regB, logsA, logsB, a, b
}

func TestOverlapOtherOwnersUnfinishedStateIsFinished(t *testing.T) {
	regA, regB, _, _, a, _ := startOverlapOwners(t)

	a.Offset = 500
	regA.Channel <- []file.State{a}
	require.Eventually(t, func() bool { return inMemoryOffset(regB, a.Id) == 500 }, 5*time.Second, 10*time.Millisecond)

	finishedA, _ := finishedIn(regA, a.Id)
	finishedB, _ := finishedIn(regB, a.Id)
	assert.False(t, finishedA, "the harvesting owner must see its state unfinished")
	assert.True(t, finishedB, "another owner must see the state finished")
}

func TestOverlapSingleOwnerKeepsUnfinishedStates(t *testing.T) {
	regA, _, _, _, a, b := startOverlapOwners(t)

	a.Offset, b.Offset = 500, 600
	regA.Channel <- []file.State{a, b}
	require.Eventually(t, func() bool { return inMemoryOffset(regA, b.Id) == 600 }, 5*time.Second, 10*time.Millisecond)

	for _, id := range []string{a.Id, b.Id} {
		finished, _ := finishedIn(regA, id)
		assert.False(t, finished, "a single owner's unfinished states must stay unfinished")
	}
}

func TestOverlapWarnsOncePerFileOnBothOwners(t *testing.T) {
	regA, regB, logsA, logsB, a, _ := startOverlapOwners(t)

	a.Offset = 500
	regA.Channel <- []file.State{a}
	require.Eventually(t, func() bool { return inMemoryOffset(regA, a.Id) == 500 }, 5*time.Second, 10*time.Millisecond)
	a.Offset = 600
	regB.Channel <- []file.State{a}
	require.Eventually(t, func() bool { return inMemoryOffset(regA, a.Id) == 600 }, 5*time.Second, 10*time.Millisecond)
	a.Offset = 700
	regA.Channel <- []file.State{a}
	require.Eventually(t, func() bool { return inMemoryOffset(regA, a.Id) == 700 }, 5*time.Second, 10*time.Millisecond)
	a.Offset = 800
	regB.Channel <- []file.State{a}
	require.Eventually(t, func() bool { return inMemoryOffset(regA, a.Id) == 800 }, 5*time.Second, 10*time.Millisecond)

	stopWithin(t, regB, 5*time.Second)
	assert.Equal(t, 1, overlapWarnings(logsA), "owner A must log the warning once")
	assert.Equal(t, 1, overlapWarnings(logsB), "owner B must log the warning once")
}

func TestOverlapHandoffDoesNotWarn(t *testing.T) {
	regA, regB, logsA, logsB, a, _ := startOverlapOwners(t)

	a.Offset = 500
	regA.Channel <- []file.State{a}
	a.Offset, a.Finished = 600, true
	regA.Channel <- []file.State{a}
	require.Eventually(t, func() bool { return inMemoryOffset(regB, a.Id) == 600 }, 5*time.Second, 10*time.Millisecond)
	a.Offset, a.Finished = 700, false
	regB.Channel <- []file.State{a}

	require.Eventually(t, func() bool { return inMemoryOffset(regA, a.Id) == 700 }, 5*time.Second, 10*time.Millisecond)
	stopWithin(t, regB, 5*time.Second)
	assert.Zero(t, overlapWarnings(logsA), "a handoff must not warn on A")
	assert.Zero(t, overlapWarnings(logsB), "a handoff must not warn on B")
}

func TestOverlapLeavingOwnerAndGCLeaveNoTracking(t *testing.T) {
	regA, regB, _, _, a, b := startOverlapOwners(t)

	a.Offset = 500
	regA.Channel <- []file.State{a}
	b.Offset = 600
	regB.Channel <- []file.State{b}
	require.Eventually(t, func() bool { return inMemoryOffset(regA, b.Id) == 600 }, 5*time.Second, 10*time.Millisecond)
	shared := regA.shared
	shared.mu.Lock()
	shared.warned[a.Id] = struct{}{}
	shared.mu.Unlock()

	stopWithin(t, regA, 5*time.Second)
	shared.mu.Lock()
	_, hasA := shared.owned[regA]
	_, warned := shared.warned[a.Id]
	shared.mu.Unlock()
	assert.False(t, hasA, "a leaving owner must be forgotten")
	assert.False(t, warned, "a leaving owner's warnings must be forgotten")

	b.Offset, b.Finished, b.TTL = 700, true, 0
	regB.Channel <- []file.State{b}
	require.Eventually(t, func() bool {
		shared.mu.Lock()
		defer shared.mu.Unlock()
		_, ok := shared.owned[regB][b.Id]
		return !ok
	}, 5*time.Second, 10*time.Millisecond, "a finished state must leave the owner's set")
}

func TestOverlapWarnsAgainAfterHandoff(t *testing.T) {
	regA, regB, logsA, logsB, a, _ := startOverlapOwners(t)
	send := func(r *Registrar, offset int64, finished bool) {
		a.Offset, a.Finished = offset, finished
		r.Channel <- []file.State{a}
		require.Eventually(t, func() bool { return inMemoryOffset(regA, a.Id) == offset }, 5*time.Second, 10*time.Millisecond)
	}

	send(regA, 500, false)
	send(regB, 600, false)
	send(regA, 700, true)
	send(regA, 800, false)

	stopWithin(t, regB, 5*time.Second)
	assert.Equal(t, 2, overlapWarnings(logsA), "owner A must warn again after the handoff")
	assert.Equal(t, 2, overlapWarnings(logsB), "owner B must warn again after the handoff")
}

func TestOverlapLeavingOwnerKeepsStateWithExpiredTTL(t *testing.T) {
	regA, regB, _, _, a, b := startOverlapOwners(t)
	stateStore, ok := regA.stateStore.(*testStateStore)
	require.True(t, ok, "state store must be a testStateStore")

	a.Offset, a.TTL = 500, 500*time.Millisecond
	regA.Channel <- []file.State{a}
	start := time.Now()
	require.Eventually(t, func() bool { return inMemoryOffset(regA, a.Id) == 500 }, 5*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool { return time.Since(start) > 700*time.Millisecond }, 5*time.Second, 10*time.Millisecond)

	stopWithin(t, regA, 5*time.Second)
	b.Offset = 600
	regB.Channel <- []file.State{b}
	require.Eventually(t, func() bool { return inMemoryOffset(regB, b.Id) == 600 }, 5*time.Second, 10*time.Millisecond)
	stopWithin(t, regB, 5*time.Second)

	assert.EqualValues(t, 500, readState(t, stateStore, a.Id).Offset, "an expired but active state must survive its owner stopping")
}
