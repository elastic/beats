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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func seededStore(t *testing.T) (*testStateStore, file.State, file.State) {
	t.Helper()
	memBackend := storetest.NewMemoryStoreBackend()
	stateStore := &testStateStore{registry: statestore.NewRegistry(memBackend)}
	store, err := stateStore.StoreFor("", "")
	require.NoError(t, err, "shared store must open")
	fileA := file.State{Id: "native::1-1", IdentifierName: "native", Source: "/var/log/a.log", Offset: 100, TTL: -1, FileStateOS: testStateOS(1)}
	fileB := file.State{Id: "native::2-1", IdentifierName: "native", Source: "/var/log/b.log", Offset: 100, TTL: -1, FileStateOS: testStateOS(2)}
	require.NoError(t, writeStates(store, []file.State{fileA, fileB}), "seeding previous run state must succeed")
	require.NoError(t, store.Close(), "seed store handle must close")
	return stateStore, fileA, fileB
}

func readState(t *testing.T, stateStore *testStateStore, id string) file.State {
	t.Helper()
	store, err := stateStore.StoreFor("", "")
	require.NoError(t, err, "store must open for reading")
	defer store.Close()
	var st file.State
	require.NoError(t, store.Get(fileStatePrefix+id, &st), "state %q must be readable", id)
	return st
}

func TestSharedRegistrarPreservesOtherOwnersOffsets(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, fileB := seededStore(t)

	spyA, spyB := &spyLogger{}, &spyLogger{}
	regA, err := New(stateStore, spyA, 0, logger)
	require.NoError(t, err, "registrar A must be created")
	regB, err := New(stateStore, spyB, 0, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	require.NoError(t, regB.Start(), "registrar B must start")
	assert.Same(t, regA.shared, regB.shared, "owners with the same store key must share one registrar")
	assert.Len(t, regA.GetStates(), 2, "both owners must see the previously persisted states")

	fileA.Offset = 5000
	regA.Channel <- []file.State{fileA}
	regA.Stop()

	fileB.Offset = 7000
	regB.Channel <- []file.State{fileB}
	regB.Stop()

	assert.EqualValues(t, 5000, readState(t, stateStore, fileA.Id).Offset, "a.log offset written by A must survive B's later flush")
	assert.EqualValues(t, 7000, readState(t, stateStore, fileB.Id).Offset, "b.log offset written by B must be persisted")
	assert.Equal(t, 1, spyA.n, "A must be acked for exactly its own state")
	assert.Equal(t, 1, spyB.n, "B must be acked for exactly its own state")
	assert.Equal(t, 0, sharedRegistrars.Len(), "shared registrar must be released after the last owner stops")
}

func TestSharedRegistrarAcksEachOwnerSeparately(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, fileB := seededStore(t)

	spyA, spyB := &spyLogger{}, &spyLogger{}
	regA, err := New(stateStore, spyA, time.Second, logger)
	require.NoError(t, err, "registrar A must be created")
	regB, err := New(stateStore, spyB, time.Second, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	require.NoError(t, regB.Start(), "registrar B must start")

	regA.Channel <- []file.State{fileA, fileA}
	regB.Channel <- []file.State{fileB}
	regA.Stop()
	regB.Stop()

	assert.Equal(t, 2, spyA.n, "A must be acked for its two states only")
	assert.Equal(t, 1, spyB.n, "B must be acked for its one state only")
}

func TestRegistrarsOnDifferentStoresStayIsolated(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStoreA, fileA, _ := seededStore(t)
	stateStoreB, _, fileB := seededStore(t)

	regA, err := New(stateStoreA, &spyLogger{}, 0, logger)
	require.NoError(t, err, "registrar A must be created")
	regB, err := New(stateStoreB, &spyLogger{}, 0, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	require.NoError(t, regB.Start(), "registrar B must start")
	assert.NotSame(t, regA.shared, regB.shared, "different store keys must not share a registrar")
	assert.Equal(t, 2, sharedRegistrars.Len(), "one shared registrar per store key must exist")

	fileA.Offset = 5000
	regA.Channel <- []file.State{fileA}
	regA.Stop()
	fileB.Offset = 7000
	regB.Channel <- []file.State{fileB}
	regB.Stop()

	assert.EqualValues(t, 5000, readState(t, stateStoreA, fileA.Id).Offset, "store A must hold A's update")
	assert.EqualValues(t, 100, readState(t, stateStoreA, fileB.Id).Offset, "store A must not see B's update")
	assert.EqualValues(t, 7000, readState(t, stateStoreB, fileB.Id).Offset, "store B must hold B's update")
	assert.EqualValues(t, 100, readState(t, stateStoreB, fileA.Id).Offset, "store B must not see A's update")
	assert.Equal(t, 0, sharedRegistrars.Len(), "all shared registrars must be released")
}

func TestSharedRegistrarSurvivesFirstOwnerStopping(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	stateStore, fileA, fileB := seededStore(t)

	regA, err := New(stateStore, &spyLogger{}, 0, logger)
	require.NoError(t, err, "registrar A must be created")
	require.NoError(t, regA.Start(), "registrar A must start")
	regB, err := New(stateStore, &spyLogger{}, 0, logger)
	require.NoError(t, err, "registrar B must be created")
	require.NoError(t, regB.Start(), "registrar B must start")

	fileA.Offset = 5000
	regA.Channel <- []file.State{fileA}
	regA.Stop()
	assert.Equal(t, 1, sharedRegistrars.Len(), "shared registrar must stay open while B is still attached")

	fileB.Offset = 7000
	regB.Channel <- []file.State{fileB}
	regB.Stop()

	assert.EqualValues(t, 5000, readState(t, stateStore, fileA.Id).Offset, "A's update must persist after A stopped first")
	assert.EqualValues(t, 7000, readState(t, stateStore, fileB.Id).Offset, "B's update must persist via the registrar A created")
	assert.Equal(t, 0, sharedRegistrars.Len(), "shared registrar must close after the last owner stops")
}
