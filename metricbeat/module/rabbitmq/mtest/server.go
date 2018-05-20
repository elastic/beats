package mtest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ServerConfig is the configuration of a testing server
type ServerConfig struct {
	ManagementPathPrefix string
	DataDir              string
}

// DefaultServerConfig is the default configuration of a testing server
var DefaultServerConfig = ServerConfig{
	ManagementPathPrefix: "",
	DataDir:              "../_meta/testdata/",
}

// Server starts a mocked RabbitMQ management API, it has to be closed with `server.Close()`
func Server(t *testing.T, c ServerConfig) *httptest.Server {
	absPath, err := filepath.Abs(c.DataDir)
	assert.Nil(t, err)

	responses := map[string]*struct {
		file string
		body []byte
	}{
		c.ManagementPathPrefix + "/api/connections":               {file: "connection_sample_response.json"},
		c.ManagementPathPrefix + "/api/exchanges":                 {file: "exchange_sample_response.json"},
		c.ManagementPathPrefix + "/api/nodes":                     {file: "nodes_sample_response.json"},
		c.ManagementPathPrefix + "/api/nodes/rabbit@e2b1ae6390fd": {file: "node_sample_response.json"},
		c.ManagementPathPrefix + "/api/overview":                  {file: "overview_sample_response.json"},
		c.ManagementPathPrefix + "/api/queues":                    {file: "queue_sample_response.json"},
	}

	for k := range responses {
		r, err := ioutil.ReadFile(filepath.Join(absPath, responses[k].file))
		responses[k].body = r
		assert.NoError(t, err)
	}

	notFound, err := ioutil.ReadFile(filepath.Join(absPath, "notfound_response.json"))
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json;")
		if response, found := responses[r.URL.Path]; found {
			w.WriteHeader(200)
			w.Write(response.body)
		} else {
			w.WriteHeader(404)
			w.Write(notFound)
			t.Log("RabbitMQ 404 error, url requested: " + r.URL.Path)
		}
	}))
	return server
}
