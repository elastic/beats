package internal

import (
	"fmt"
	"os"
)

const (
	// ElasticsearchDefaultHost is the default host for elasticsearch.
	ElasticsearchDefaultHost = "localhost"
	// ElasticsearchDefaultPort is the default port for elasticsearch.
	ElasticsearchDefaultPort = "9200"
)

// TestLogger is used to report fatal errors to the testing framework.
type TestLogger interface {
	Fatal(args ...interface{})
}

// Connectable defines the minimum interface required to initialize a connected
// client.
type Connectable interface {
	Connect() error
}

// InitClient initializes a new client if the no error value from creating the
// client instance is reported.
// The test logger will be used if an error is found.
func InitClient(t TestLogger, client Connectable, err error) {
	if err == nil {
		err = client.Connect()
	}

	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}
}

// GetEsHost returns the Elasticsearch testing host.
func GetEsHost() string {
	return getEnv("ES_HOST", ElasticsearchDefaultHost)
}

// GetEsPort returns the Elasticsearch testing port.
func GetEsPort() string {
	return getEnv("ES_PORT", ElasticsearchDefaultPort)
}

// GetURL return the Elasticsearch testing URL.
func GetURL() string {
	return fmt.Sprintf("http://%v:%v", GetEsHost(), GetEsPort())
}

// GetUser returns the Elasticsearch testing user.
func GetUser() string { return getEnv("ES_USER", "") }

// GetPass returns the Elasticsearch testing user's password.
func GetPass() string { return getEnv("ES_PASS", "") }

func getEnv(name, def string) string {
	if v := os.Getenv(name); len(v) > 0 {
		return v
	}
	return def
}
