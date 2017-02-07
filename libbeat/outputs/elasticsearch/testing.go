package elasticsearch

import (
	"os"
	"time"

	"github.com/elastic/beats/libbeat/outputs/outil"
)

const ElasticsearchDefaultHost = "localhost"
const ElasticsearchDefaultPort = "9200"

func GetEsPort() string {
	port := os.Getenv("ES_PORT")

	if len(port) == 0 {
		port = ElasticsearchDefaultPort
	}
	return port
}

// Returns
func GetEsHost() string {

	host := os.Getenv("ES_HOST")

	if len(host) == 0 {
		host = ElasticsearchDefaultHost
	}

	return host
}

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
