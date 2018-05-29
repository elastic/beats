package rabbitmq

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/rabbitmq/mtest"

	"github.com/stretchr/testify/assert"
)

func init() {
	mb.Registry.MustAddMetricSet("rabbitmq", "test", newTestMetricSet,
		mb.WithHostParser(HostParser),
	)
}

type testMetricSet struct {
	*MetricSet
}

func newTestMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := NewMetricSet(base, "/api/overview")
	if err != nil {
		return nil, err
	}
	return &testMetricSet{ms}, nil
}

// Fetch makes an HTTP request to fetch connections metrics from the connections endpoint.
func (m *testMetricSet) Fetch() ([]common.MapStr, error) {
	_, err := m.HTTP.FetchContent()
	return nil, err
}

func TestManagementPathPrefix(t *testing.T) {
	server := mtest.Server(t, mtest.ServerConfig{
		ManagementPathPrefix: "/management_prefix",
		DataDir:              "./_meta/testdata",
	})
	defer server.Close()

	config := map[string]interface{}{
		"module":      "rabbitmq",
		"metricsets":  []string{"test"},
		"hosts":       []string{server.URL},
		pathConfigKey: "/management_prefix",
	}

	f := mbtest.NewEventsFetcher(t, config)
	_, err := f.Fetch()
	assert.NoError(t, err)
}
