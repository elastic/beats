package elasticsearch

import (
	"math/rand"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/outputs/outil"
)

const (
	// ElasticsearchDefaultHost is the default host for elasticsearch.
	ElasticsearchDefaultHost = "localhost"
	// ElasticsearchDefaultPort is the default port for elasticsearch.
	ElasticsearchDefaultPort = "9200"
)

// GetEsHost returns the elasticsearch host.
func GetEsHost() string {

	host := os.Getenv("ES_HOST")

	if len(host) == 0 {
		host = ElasticsearchDefaultHost
	}

	return host
}

// GetEsPort returns the elasticsearch port.
func GetEsPort() string {
	port := os.Getenv("ES_PORT")

	if len(port) == 0 {
		port = ElasticsearchDefaultPort
	}
	return port
}

// GetTestingElasticsearch creates a test client.
func GetTestingElasticsearch() *Client {
	var address = "http://" + GetEsHost() + ":" + GetEsPort()
	username := os.Getenv("ES_USER")
	pass := os.Getenv("ES_PASS")
	client := newTestClientAuth(address, username, pass)

	// Load version number
	client.Connect(3 * time.Second)
	return client
}

func newTestClientAuth(url, user, pass string) *Client {
	client, err := NewClient(ClientSettings{
		URL:              url,
		Index:            outil.MakeSelector(),
		Username:         user,
		Password:         pass,
		Timeout:          60 * time.Second,
		CompressionLevel: 3,
	}, nil)
	if err != nil {
		panic(err)
	}
	return client
}

func (t *elasticsearchOutput) randomClient() *Client {
	switch len(t.clients) {
	case 0:
		return nil
	case 1:
		return t.clients[0].(*Client).Clone()
	default:
		return t.clients[rand.Intn(len(t.clients))].(*Client).Clone()
	}
}
