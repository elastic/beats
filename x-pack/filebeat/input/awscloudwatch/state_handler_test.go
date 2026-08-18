// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/libbeat/statestore"
	"github.com/elastic/beats/v9/libbeat/statestore/storetest"
)

func TestStateHandler(t *testing.T) {
	t.Run("simple run - register once and complete once", func(t *testing.T) {
		cfg := config{LogGroupARN: "logGroupARN"}
		st, err := newStateHandler(nil, cfg, createTestInputStore())
		assert.NoError(t, err)

		st.WorkRegister(100, 1)
		st.WorkComplete(100)

		// pause for backgroundRunner to complete
		<-time.After(100 * time.Millisecond)

		state, err := st.GetState()
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.Equal(t, int64(100), state.LastSyncEpoch)
	})

	t.Run("Track and validate multiple work counts", func(t *testing.T) {
		// given
		cfg := config{LogGroupARN: "logGroupARN"}
		st, err := newStateHandler(nil, cfg, createTestInputStore())
		assert.NoError(t, err)

		tStamp := int64(100)
		workCount := 5
		st.WorkRegister(tStamp, workCount)

		for i := 0; i < (workCount - 1); i++ {
			st.WorkComplete(tStamp)
			<-time.After(100 * time.Millisecond)

			state, err := st.GetState()
			assert.NoError(t, err)
			// zero value - state not updated
			assert.Equal(t, int64(0), state.LastSyncEpoch)
		}

		st.WorkComplete(tStamp)
		<-time.After(100 * time.Millisecond)

		state, err := st.GetState()
		assert.NoError(t, err)
		assert.Equal(t, tStamp, state.LastSyncEpoch)

	})

	t.Run("State is not updated if oldest work is not yet complete", func(t *testing.T) {
		cfg := config{LogGroupARN: "logGroupARN"}
		st, err := newStateHandler(nil, cfg, createTestInputStore())
		assert.NoError(t, err)

		st.WorkRegister(100, 1)
		st.WorkRegister(200, 1)

		// complete the newest
		st.WorkComplete(200)

		// pause for backgroundRunner to run
		<-time.After(100 * time.Millisecond)

		// Validation #1 : State is not updated as oldest is not complete
		state, err := st.GetState()
		assert.NoError(t, err)
		assert.NotNil(t, state)

		// we get zero so that sync starts from epoch zero
		assert.Equal(t, int64(0), state.LastSyncEpoch)

		// complete the oldest
		st.WorkComplete(100)

		// pause for backgroundRunner to run
		<-time.After(100 * time.Millisecond)

		// Validation #2 : State is updated to the latest once completion get registered
		state, err = st.GetState()
		assert.NoError(t, err)
		assert.NotNil(t, state)

		// we get most recent completion
		assert.Equal(t, int64(200), state.LastSyncEpoch)

	})
}

func TestStoreAndGetState(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config
		storingState storableState
		expectEpoch  int64
	}{
		{
			name: "Store with arn",
			cfg: config{
				LogGroupARN: "logGroupARN",
			},
			storingState: storableState{
				LastSyncEpoch: 1111111111,
			},
			expectEpoch: 1111111111,
		},
		{
			name: "Store with group name",
			cfg: config{
				LogGroupName: "LogGroupName",
			},
			storingState: storableState{
				LastSyncEpoch: 22222222,
			},
			expectEpoch: 22222222,
		},
		{
			name: "Store with prefix",
			cfg: config{
				LogGroupNamePrefix: "LogGroupNamePrefix",
			},
			storingState: storableState{
				LastSyncEpoch: 333333333,
			},
			expectEpoch: 333333333,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stHandler, err := newStateHandler(nil, test.cfg, createTestInputStore())
			require.NoError(t, err)

			err = stHandler.storeState(test.storingState)
			require.NoError(t, err)

			got, err := stHandler.GetState()
			require.NoError(t, err)

			require.Equal(t, test.expectEpoch, got.LastSyncEpoch)
		})
	}
}

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config
		want    string
		isError bool
	}{
		{
			name: "ID using ARN",
			cfg: config{
				LogGroupARN: "logGroupARN",
			},
			want: "filebeat::aws-cloudwatch::state::groupArn::logGroupARN",
		},
		{
			name: "ID using Group Name",
			cfg: config{
				LogGroupName: "logGroupName",
				RegionName:   "region-A",
			},
			want: "filebeat::aws-cloudwatch::state::groupName::logGroupName::region-A",
		},
		{
			name: "ID using Group Name",
			cfg: config{
				LogGroupNamePrefix: "groupPrefix",
				RegionName:         "region-A",
			},
			want: "filebeat::aws-cloudwatch::state::groupPrefix::groupPrefix::region-A",
		},
		{
			name:    "Invalid configuration results in an error",
			isError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id, err := generateID(test.cfg)

			if test.isError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, id)
		})
	}

}

// testInputStore - State registry for testing needs
type testInputStore struct {
	registry *statestore.Registry
}

func (s *testInputStore) StoreFor(typ string) (*statestore.Store, error) {
	return s.registry.Get(typ)
}

func createTestInputStore() *testInputStore {
	return &testInputStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testInputStore) CleanupInterval() time.Duration {
	return 24 * time.Hour
}

func (s *testInputStore) Close() {
	_ = s.registry.Close()
}
