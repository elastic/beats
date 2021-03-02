// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package store

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

func TestStateStore(t *testing.T) {
	t.Run("ack token", func(t *testing.T) {
		runTestStateStore(t, "")
	})

	t.Run("no ack token", func(t *testing.T) {
		runTestStateStore(t, "czlV93YBwdkt5lYhBY7S")
	})
}

func runTestStateStore(t *testing.T, ackToken string) {
	log, _ := logger.New("state_store")
	withFile := func(fn func(t *testing.T, file string)) func(*testing.T) {
		return func(t *testing.T) {
			dir, err := ioutil.TempDir("", "state-store")
			require.NoError(t, err)
			defer os.RemoveAll(dir)
			file := filepath.Join(dir, "state.yml")
			fn(t, file)
		}
	}

	t.Run("action returns empty when no action is saved on disk",
		withFile(func(t *testing.T, file string) {
			s := storage.NewDiskStore(file)
			store, err := NewStateStore(log, s)
			require.NoError(t, err)
			require.Equal(t, 0, len(store.Actions()))
		}))

	t.Run("will discard silently unknown action",
		withFile(func(t *testing.T, file string) {
			actionPolicyChange := &fleetapi.ActionUnknown{
				ActionID: "abc123",
			}

			s := storage.NewDiskStore(file)
			store, err := NewStateStore(log, s)
			require.NoError(t, err)

			require.Equal(t, 0, len(store.Actions()))
			store.Add(actionPolicyChange)
			store.SetAckToken(ackToken)
			err = store.Save()
			require.NoError(t, err)
			require.Equal(t, 0, len(store.Actions()))
			require.Equal(t, ackToken, store.AckToken())
		}))

	t.Run("can save to disk known action type",
		withFile(func(t *testing.T, file string) {
			ActionPolicyChange := &fleetapi.ActionPolicyChange{
				ActionID:   "abc123",
				ActionType: "POLICY_CHANGE",
				Policy: map[string]interface{}{
					"hello": "world",
				},
			}

			s := storage.NewDiskStore(file)
			store, err := NewStateStore(log, s)
			require.NoError(t, err)

			require.Equal(t, 0, len(store.Actions()))
			store.Add(ActionPolicyChange)
			store.SetAckToken(ackToken)
			err = store.Save()
			require.NoError(t, err)
			require.Equal(t, 1, len(store.Actions()))
			require.Equal(t, ackToken, store.AckToken())

			s = storage.NewDiskStore(file)
			store1, err := NewStateStore(log, s)
			require.NoError(t, err)

			actions := store1.Actions()
			require.Equal(t, 1, len(actions))

			require.Equal(t, ActionPolicyChange, actions[0])
			require.Equal(t, ackToken, store.AckToken())
		}))

	t.Run("can save to disk unenroll action type",
		withFile(func(t *testing.T, file string) {
			action := &fleetapi.ActionUnenroll{
				ActionID:   "abc123",
				ActionType: "UNENROLL",
			}

			s := storage.NewDiskStore(file)
			store, err := NewStateStore(log, s)
			require.NoError(t, err)

			require.Equal(t, 0, len(store.Actions()))
			store.Add(action)
			store.SetAckToken(ackToken)
			err = store.Save()
			require.NoError(t, err)
			require.Equal(t, 1, len(store.Actions()))
			require.Equal(t, ackToken, store.AckToken())

			s = storage.NewDiskStore(file)
			store1, err := NewStateStore(log, s)
			require.NoError(t, err)

			actions := store1.Actions()
			require.Equal(t, 1, len(actions))

			require.Equal(t, action, actions[0])
			require.Equal(t, ackToken, store.AckToken())
		}))

	t.Run("when we ACK we save to disk",
		withFile(func(t *testing.T, file string) {
			ActionPolicyChange := &fleetapi.ActionPolicyChange{
				ActionID: "abc123",
			}

			s := storage.NewDiskStore(file)
			store, err := NewStateStore(log, s)
			require.NoError(t, err)
			store.SetAckToken(ackToken)

			acker := NewStateStoreActionAcker(&testAcker{}, store)
			require.Equal(t, 0, len(store.Actions()))

			require.NoError(t, acker.Ack(context.Background(), ActionPolicyChange))
			require.Equal(t, 1, len(store.Actions()))
			require.Equal(t, ackToken, store.AckToken())
		}))

	t.Run("migrate actions file does not exists",
		withFile(func(t *testing.T, actionStorePath string) {
			withFile(func(t *testing.T, stateStorePath string) {
				err := migrateStateStore(log, actionStorePath, stateStorePath)
				require.NoError(t, err)
				stateStore, err := NewStateStore(log, storage.NewDiskStore(stateStorePath))
				require.NoError(t, err)
				stateStore.SetAckToken(ackToken)
				require.Equal(t, 0, len(stateStore.Actions()))
				require.Equal(t, ackToken, stateStore.AckToken())
			})
		}))

	t.Run("migrate",
		withFile(func(t *testing.T, actionStorePath string) {
			ActionPolicyChange := &fleetapi.ActionPolicyChange{
				ActionID:   "abc123",
				ActionType: "POLICY_CHANGE",
				Policy: map[string]interface{}{
					"hello": "world",
				},
			}

			actionStore, err := NewActionStore(log, storage.NewDiskStore(actionStorePath))
			require.NoError(t, err)

			require.Equal(t, 0, len(actionStore.Actions()))
			actionStore.Add(ActionPolicyChange)
			err = actionStore.Save()
			require.NoError(t, err)
			require.Equal(t, 1, len(actionStore.Actions()))

			withFile(func(t *testing.T, stateStorePath string) {
				err = migrateStateStore(log, actionStorePath, stateStorePath)
				require.NoError(t, err)

				stateStore, err := NewStateStore(log, storage.NewDiskStore(stateStorePath))
				require.NoError(t, err)
				stateStore.SetAckToken(ackToken)
				diff := cmp.Diff(actionStore.Actions(), stateStore.Actions())
				if diff != "" {
					t.Error(diff)
				}
				require.Equal(t, ackToken, stateStore.AckToken())
			})
		}))

}

type testAcker struct {
	acked     []string
	ackedLock sync.Mutex
}

func (t *testAcker) Ack(_ context.Context, action fleetapi.Action) error {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()

	if t.acked == nil {
		t.acked = make([]string, 0)
	}

	t.acked = append(t.acked, action.ID())
	return nil
}

func (t *testAcker) Commit(_ context.Context) error {
	return nil
}

func (t *testAcker) Clear() {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()

	t.acked = make([]string, 0)
}

func (t *testAcker) Items() []string {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()
	return t.acked
}
