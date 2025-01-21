// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
)

func TestStateHandler(t *testing.T) {
	tests := []struct {
		name            string
		storeWithCfg    config
		storingState    StorableState
		retrieveWithCfg config
		expectEpoch     int64
	}{
		{
			name: "Simple store and retrival",
			storeWithCfg: config{
				LogGroupARN: "logGroupARN",
			},
			storingState: StorableState{
				LastSyncEpoch: 1111111111,
			},
			retrieveWithCfg: config{
				LogGroupARN: "logGroupARN",
			},
			expectEpoch: 1111111111,
		},
		{
			name: "Missing retrival should return epoch zero - different ARN",
			storeWithCfg: config{
				LogGroupARN: "MyLogGroup_A",
			},
			storingState: StorableState{
				LastSyncEpoch: 1111111111,
			},
			retrieveWithCfg: config{
				LogGroupARN: "MyLogGroup_B",
			},
			expectEpoch: 0,
		},
		{
			name: "Missing retrival should return epoch zero - different log group identification",
			storeWithCfg: config{
				LogGroupARN: "MyLogGroup_A",
			},
			storingState: StorableState{
				LastSyncEpoch: 1111111111,
			},
			retrieveWithCfg: config{
				LogGroupName: "logGroupName",
				RegionName:   "region-A",
			},
			expectEpoch: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stHandler, err := createStateHandler(createTestInputStore())
			require.NoError(t, err)

			err = stHandler.StoreState(test.storeWithCfg, test.storingState)
			require.NoError(t, err)

			retried, err := stHandler.GetState(test.retrieveWithCfg)
			require.NoError(t, err)

			require.Equal(t, test.expectEpoch, retried.LastSyncEpoch)
		})
	}
}

func Test_getID(t *testing.T) {
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
