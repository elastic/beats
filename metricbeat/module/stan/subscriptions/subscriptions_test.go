package subscriptions

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("./_meta/test/subscriptions.json")
	assert.NoError(t, err)
	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(content, reporter)
	assert.NoError(t, err)
	evts := reporter.GetEvents()

	// 115 subscribers
	assert.Equal(t, len(evts), 115)
	errErrs := reporter.GetErrors()
	assert.Equal(t, len(errErrs), 0)
}
func TestFetchEventContent(t *testing.T) {
	absPath, _ := filepath.Abs("./_meta/test/")

	response, _ := ioutil.ReadFile(absPath + "/subscriptions.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "stan",
		"metricsets": []string{"subscriptions"},
		"hosts":      []string{server.URL},
	}
	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	metricSet.Fetch(reporter)

	for idx, evt := range reporter.GetEvents() {
		e := mbtest.StandardizeEvent(metricSet, evt)
		t.Logf("[%d] %s/%s event: %+v", idx, metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
	}

}
