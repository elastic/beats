package rabbitmq

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

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
	ms, err := NewMetricSet(base, "/api/test")
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
	visited := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/management_prefix/api/test":
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json;")
			visited = true
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":      "rabbitmq",
		"metricsets":  []string{"test"},
		"hosts":       []string{server.URL},
		pathConfigKey: "/management_prefix",
	}

	f := mbtest.NewEventsFetcher(t, config)
	f.Fetch()
	assert.True(t, visited)
}
