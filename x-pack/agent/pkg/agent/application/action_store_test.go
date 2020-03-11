// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/fleetapi"
)

func TestActionStore(t *testing.T) {
	log, _ := logger.New()
	withFile := func(fn func(t *testing.T, file string)) func(*testing.T) {
		return func(t *testing.T) {
			dir, err := ioutil.TempDir("", "action-store")
			require.NoError(t, err)
			defer os.RemoveAll(dir)
			file := filepath.Join(dir, "config.yml")
			fn(t, file)
		}
	}

	t.Run("action returns empty when no action is saved on disk",
		withFile(func(t *testing.T, file string) {
			s := storage.NewDiskStore(file)
			store, err := newActionStore(log, s)
			require.NoError(t, err)
			require.Equal(t, 0, len(store.Actions()))
		}))

	t.Run("will discard silently unknown action",
		withFile(func(t *testing.T, file string) {
			actionPolicyChange := &fleetapi.ActionUnknown{
				ActionID: "abc123",
			}

			s := storage.NewDiskStore(file)
			store, err := newActionStore(log, s)
			require.NoError(t, err)

			require.Equal(t, 0, len(store.Actions()))
			store.Add(actionPolicyChange)
			err = store.Save()
			require.NoError(t, err)
			require.Equal(t, 0, len(store.Actions()))
		}))

	t.Run("can save to disk known action type",
		withFile(func(t *testing.T, file string) {
			actionPolicyChange := &fleetapi.ActionPolicyChange{
				ActionID:   "abc123",
				ActionType: "POLICY_CHANGE",
				Policy: map[string]interface{}{
					"hello": "world",
				},
			}

			s := storage.NewDiskStore(file)
			store, err := newActionStore(log, s)
			require.NoError(t, err)

			require.Equal(t, 0, len(store.Actions()))
			store.Add(actionPolicyChange)
			err = store.Save()
			require.NoError(t, err)
			require.Equal(t, 1, len(store.Actions()))

			s = storage.NewDiskStore(file)
			store1, err := newActionStore(log, s)
			require.NoError(t, err)

			actions := store1.Actions()
			require.Equal(t, 1, len(actions))

			require.Equal(t, actionPolicyChange, actions[0])
		}))

	t.Run("when we ACK we save to disk",
		withFile(func(t *testing.T, file string) {
			actionPolicyChange := &fleetapi.ActionPolicyChange{
				ActionID: "abc123",
			}

			s := storage.NewDiskStore(file)
			store, err := newActionStore(log, s)
			require.NoError(t, err)

			acker := newActionStoreAcker(&testAcker{}, store)
			require.Equal(t, 0, len(store.Actions()))

			require.NoError(t, acker.Ack(context.Background(), actionPolicyChange))
			require.Equal(t, 1, len(store.Actions()))
		}))
}
