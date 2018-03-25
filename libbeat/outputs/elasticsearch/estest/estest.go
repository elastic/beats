package estest

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/internal"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

// GetTestingElasticsearch creates a test client.
func GetTestingElasticsearch(t internal.TestLogger) *elasticsearch.Client {
	client, err := elasticsearch.NewClient(elasticsearch.ClientSettings{
		URL:              internal.GetURL(),
		Index:            outil.MakeSelector(),
		Username:         internal.GetUser(),
		Password:         internal.GetPass(),
		Timeout:          60 * time.Second,
		CompressionLevel: 3,
	}, nil)
	internal.InitClient(t, client, err)
	return client
}
