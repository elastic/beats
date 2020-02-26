// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package eslegtest

import (
	"fmt"
	"net/http"
	"os"

	"github.com/elastic/beats/libbeat/esclientleg"

	"github.com/elastic/beats/libbeat/outputs/transport"
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

// InitConnection initializes a new connection if the no error value from creating the
// connection instance is reported.
// The test logger will be used if an error is found.
func InitConnection(t TestLogger, conn Connectable, err error) {
	if err == nil {
		err = conn.Connect()
	}

	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}
}

// GetTestingElasticsearch creates a test connection.
func GetTestingElasticsearch(t TestLogger) *esclientleg.Connection {
	conn, err := esclientleg.NewConnection(esclientleg.ConnectionSettings{
		URL: GetURL(),
		HTTP: &http.Client{
			Transport: &http.Transport{
				Dial: transport.NetDialer(0).Dial,
			},
			Timeout: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	conn.Encoder = esclientleg.NewJSONEncoder(nil, false)

	err = conn.Connect()
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	return conn
}

// GetURL return the Elasticsearch testing URL.
func GetURL() string {
	return fmt.Sprintf("http://%v:%v", GetEsHost(), getEsPort())
}

// GetEsHost returns the Elasticsearch testing host.
func GetEsHost() string {
	return getEnv("ES_HOST", ElasticsearchDefaultHost)
}

// getEsPort returns the Elasticsearch testing port.
func getEsPort() string {
	return getEnv("ES_PORT", ElasticsearchDefaultPort)
}

// GetUser returns the Elasticsearch testing user.
func GetUser() string {
	return getEnv("ES_USER", "")
}

// GetPass returns the Elasticsearch testing user's password.
func GetPass() string {
	return getEnv("ES_PASS", "")
}

func getEnv(name, def string) string {
	if v := os.Getenv(name); len(v) > 0 {
		return v
	}
	return def
}
