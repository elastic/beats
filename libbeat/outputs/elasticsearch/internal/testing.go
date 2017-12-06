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

type TestLogger interface {
	Fatal(args ...interface{})
}

type Connectable interface {
	Connect() error
}

func InitClient(t TestLogger, client Connectable, err error) {
	if err == nil {
		err = client.Connect()
	}

	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}
}

// getEsHost returns the elasticsearch host.
func GetEsHost() string {
	return getEnv("ES_HOST", ElasticsearchDefaultHost)
}

// getEsPort returns the elasticsearch port.
func GetEsPort() string {
	return getEnv("ES_PORT", ElasticsearchDefaultPort)
}

func GetURL() string {
	return fmt.Sprintf("http://%v:%v", GetEsHost(), GetEsPort())
}

func GetUser() string { return getEnv("ES_USER", "") }
func GetPass() string { return getEnv("ES_PASS", "") }

func getEnv(name, def string) string {
	if v := os.Getenv(name); len(v) > 0 {
		return v
	}
	return def
}
