// +build integration

package status

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/uwsgi"

	"github.com/stretchr/testify/assert"
)

func TestFetchTCP(t *testing.T) {
	compose.EnsureUp(t, "uwsgi_tcp")

	f := mbtest.NewEventsFetcher(t, getConfig("tcp"))
	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.True(t, len(events) > 0)
	totals := findItems(events, "total")
	assert.Equal(t, 1, len(totals))
}

func TestFetchHTTP(t *testing.T) {
	compose.EnsureUp(t, "uwsgi_http")

	f := mbtest.NewEventsFetcher(t, getConfig("http"))
	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.True(t, len(events) > 0)
	totals := findItems(events, "total")
	assert.Equal(t, 1, len(totals))
}

func getConfig(scheme string) map[string]interface{} {
	conf := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
	}

	switch scheme {
	case "tcp":
		conf["hosts"] = []string{uwsgi.GetEnvTCPServer()}
	case "http", "https":
		conf["hosts"] = []string{uwsgi.GetEnvHTTPServer()}
	default:
		conf["hosts"] = []string{uwsgi.GetEnvTCPServer()}
	}
	return conf
}
