package awscloudwatch

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
)

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
			want: "filebeat::aws-cloudwatch::state::logGroupARN",
		},
		{
			name: "ID using Group Name",
			cfg: config{
				LogGroupName: "logGroupName",
				RegionName:   "region-A",
			},
			want: "filebeat::aws-cloudwatch::state::logGroupName::region-A",
		},
		{
			name: "ID using Group Name",
			cfg: config{
				LogGroupNamePrefix: "groupPrefix",
				RegionName:         "region-A",
			},
			want: "filebeat::aws-cloudwatch::state::groupPrefix::region-A",
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

func createTestInputStore() beater.StateStore {
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

func (s *testInputStore) Access() (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}
