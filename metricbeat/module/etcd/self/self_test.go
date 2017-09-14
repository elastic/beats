package self

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
	content, err := ioutil.ReadFile("../_meta/test/selfstats.json")
	assert.NoError(t, err)

	event := eventMapping(content)

	assert.Equal(t, event["id"], string("8e9e05c52164694d"))
}

func TestFetchEventContent(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")

	response, err := ioutil.ReadFile(absPath + "/selfstats.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "etcd",
		"metricsets": []string{"self"},
		"hosts":      []string{server.URL},
	}
	f := mbtest.NewEventFetcher(t, config)
	event, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}
