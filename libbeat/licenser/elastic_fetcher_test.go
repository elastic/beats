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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v8/libbeat/esleg/eslegclient"

	"github.com/stretchr/testify/assert"
)

func newServerClientPair(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *eslegclient.Connection) {
	mux := http.NewServeMux()
	mux.Handle("/_license/", http.HandlerFunc(handler))

	server := httptest.NewServer(mux)

	client, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL: server.URL,
	})
	if err != nil {
		t.Fatalf("could not create the elasticsearch client, error: %s", err)
	}

	return server, client
}

func TestParseJSON(t *testing.T) {
	t.Run("OSS release of Elasticsearch (Code: 405)", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Method Not Allowed", 405)
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
			w.Write([]byte("hello bad JSON"))
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
			http.Error(w, "Unauthorized", 401)
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
		filepath.Walk("testdata/", func(path string, i os.FileInfo, err error) error {
			if i.IsDir() {
				return nil
			}

			t.Run(path, func(t *testing.T) {
				h := func(w http.ResponseWriter, r *http.Request) {
					json, err := ioutil.ReadFile(path)
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
