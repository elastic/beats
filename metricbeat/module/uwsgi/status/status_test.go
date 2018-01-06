package status

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func testData(t *testing.T) (data []byte) {
	absPath, err := filepath.Abs(filepath.Join("..", "_meta", "testdata"))
	if err != nil {
		t.Fatalf("filepath failed: %s", err.Error())
		return
	}

	data, err = ioutil.ReadFile(filepath.Join(absPath, "/data.json"))
	if err != nil {
		t.Fatalf("ReadFile failed: %s", err.Error())
		return
	}
	return
}

func findItems(mp []common.MapStr, key string) []common.MapStr {
	result := make([]common.MapStr, 0, 1)
	for _, v := range mp {
		if el, ok := v[key]; ok {
			result = append(result, el.(common.MapStr))
		}
	}

	return result
}

func assertTestData(t *testing.T, mp []common.MapStr) {
	totals := findItems(mp, "total")
	assert.Equal(t, 1, len(totals))
	assert.Equal(t, 2042, totals[0]["requests"])
	assert.Equal(t, 0, totals[0]["exceptions"])
	assert.Equal(t, 34, totals[0]["write_errors"])
	assert.Equal(t, 38, totals[0]["read_errors"])

	workers := findItems(mp, "core")
	assert.Equal(t, 4, len(workers))
}

func TestFetchDataTCP(t *testing.T) {

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		conn, err := listener.Accept()
		assert.NoError(t, err)

		data := testData(t)
		conn.Write(data)
		conn.Close()
		wg.Done()
	}()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"tcp://" + listener.Addr().String()},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	assert.NoError(t, err)

	assertTestData(t, events)
	wg.Wait()
}

func TestFetchDataHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := testData(t)

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write(data)
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	assert.NoError(t, err)

	assertTestData(t, events)
}

func TestFetchDataUnmarshalledError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte("fail json.Unmarshal"))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	_, err := f.Fetch()
	assert.Error(t, err)
}

func TestFetchDataSourceDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	server.Close()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	_, err := f.Fetch()
	assert.Error(t, err)
}

func TestConfigError(t *testing.T) {
	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"unix://127.0.0.1:8080"},
	}

	f := mbtest.NewEventsFetcher(t, config)
	_, err := f.Fetch()
	assert.Error(t, err)

	config = map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"unknown_url_format"},
	}

	f = mbtest.NewEventsFetcher(t, config)
	_, err = f.Fetch()
	assert.Error(t, err)

	config = map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"ftp://127.0.0.1:8080"},
	}

	f = mbtest.NewEventsFetcher(t, config)
	_, err = f.Fetch()
	assert.Error(t, err)
}
