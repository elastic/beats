package syncgateway

import (
	"fmt"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockReporter struct{}

func (m mockReporter) Event(event mb.Event) bool {
	fmt.Println(event.MetricSetFields.StringToPrint())
	return true
}

func (m mockReporter) Error(err error) bool {
	return true
}

func TestData(t *testing.T) {
	mux := http.NewServeMux()

	mux.Handle("/_expvar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		input, _ := ioutil.ReadFile("./_meta/testdata/expvar.282c.json")
		w.Write(input)
	}))

	server := httptest.NewServer(mux)
	defer server.Close()

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig([]string{"syncgateway"}, server.URL))
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(metricsets []string, host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "couchbase",
		"metricsets": metricsets,
		"hosts":      []string{host},
		"extra": map[string]interface{}{
			"per_replication": true,
			"mem_stats":       true,
		},
	}
}
