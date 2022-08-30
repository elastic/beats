package monitorstate

import (
	"encoding/json"
	"testing"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/esutil"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/require"
)

// Helpers for tests here and elsewhere

func IntegESLoader(t *testing.T, indexPattern string, location *config.LocationWithID) StateLoader {
	return MakeESLoader(IntegES(t), indexPattern, location)
}

func IntegES(t *testing.T) (esc *elasticsearch.Client) {
	esc, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://127.0.0.1:9200"},
		Username:  "admin",
		Password:  "testing",
	})
	require.NoError(t, err)
	respBody, err := esc.Cluster.Health()
	healthRaw, err := esutil.CheckRetResp(respBody, err)
	require.NoError(t, err)

	healthResp := struct {
		Status string `json:"status"`
	}{}
	err = json.Unmarshal(healthRaw, &healthResp)
	require.NoError(t, err)
	require.Contains(t, []string{"green", "yellow"}, healthResp.Status)

	return esc
}
