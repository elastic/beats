package leader

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"testing"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("../_meta/test/leaderstats.json")
	assert.NoError(t, err)

	event := eventMapping(content)

	assert.Equal(t, event["leader"], string("924e2e83e93f2560"))
}

func TestFetchEventContent(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")

	response, err := ioutil.ReadFile(absPath + "/leaderstats.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "etcd",
		"metricsets": []string{"leader"},
		"hosts":      []string{server.URL},
	}
	f := mbtest.NewEventFetcher(t, config)
	event, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}
