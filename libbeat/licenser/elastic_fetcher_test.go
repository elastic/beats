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

package licenser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/version"

	"github.com/stretchr/testify/assert"
)

func esRootHandler(w http.ResponseWriter, r *http.Request) {
	respStr := fmt.Sprintf(`
{
  "name" : "582a64c35c16",
  "cluster_name" : "docker-cluster",
  "cluster_uuid" : "fnanWPBeSNS9KZ930Z5JmA",
  "version" : {
    "number" : "%s",
    "build_flavor" : "default",
    "build_type" : "docker",
    "build_hash" : "14b7170921f2f0e4109255b83cb9af175385d87f",
    "build_date" : "2024-08-23T00:26:58.284513650Z",
    "build_snapshot" : true,
    "lucene_version" : "9.11.1",
    "minimum_wire_compatibility_version" : "7.17.0",
    "minimum_index_compatibility_version" : "7.0.0"
  },
  "tagline" : "You Know, for Search"
}`, version.GetDefaultVersion())

	w.Write([]byte(respStr))
}

func newServerClientPair(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *eslegclient.Connection) {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(esRootHandler))
	mux.Handle("/_license/", handler)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL: server.URL,
	})
	if err != nil {
		t.Fatalf("could not create the elasticsearch client, error: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("cannot connect to ES: %s", err)
	}

	return server, client
}

func TestParseJSON(t *testing.T) {
	t.Run("OSS release of Elasticsearch (Code: 405)", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		s, c := newServerClientPair(t, h)
		defer s.Close()
		defer c.Close()

		fetcher := NewElasticFetcher(c)
		_, err := fetcher.Fetch()
		assert.Error(t, err)
	})

	t.Run("OSS release of Elasticsearch (Code: 400)", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Bad Request", 400)
		}
		s, c := newServerClientPair(t, h)
		defer s.Close()
		defer c.Close()

		fetcher := NewElasticFetcher(c)
		_, err := fetcher.Fetch()
		assert.Error(t, err)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("hello bad JSON"))
		}
		s, c := newServerClientPair(t, h)
		defer s.Close()
		defer c.Close()

		fetcher := NewElasticFetcher(c)
		_, err := fetcher.Fetch()
		assert.Error(t, err)
	})

	t.Run("401 response", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		s, c := newServerClientPair(t, h)
		defer s.Close()
		defer c.Close()

		fetcher := NewElasticFetcher(c)
		_, err := fetcher.Fetch()
		assert.Equal(t, err.Error(), "unauthorized access, could not connect to the xpack endpoint, verify your credentials")
	})

	t.Run("any error from the server", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not found", 404)
		}
		s, c := newServerClientPair(t, h)
		defer s.Close()
		defer c.Close()

		fetcher := NewElasticFetcher(c)
		_, err := fetcher.Fetch()
		assert.Error(t, err)
	})

	t.Run("200 response", func(t *testing.T) {
		_ = filepath.Walk("testdata/", func(path string, i os.FileInfo, err error) error {
			if i.IsDir() {
				return nil
			}

			t.Run(path, func(t *testing.T) {
				h := func(w http.ResponseWriter, r *http.Request) {
					json, err := os.ReadFile(path)
					if err != nil {
						t.Fatal("could not read JSON")
					}
					w.Write(json)
				}

				s, c := newServerClientPair(t, h)
				defer s.Close()
				defer c.Close()

				fetcher := NewElasticFetcher(c)
				license, err := fetcher.Fetch()
				if !assert.NoError(t, err) {
					return
				}

				assert.True(t, len(license.UUID) > 0)

				assert.NotNil(t, license.Type)
				assert.NotNil(t, license.Status)
			})

			return nil
		})
	})
}
