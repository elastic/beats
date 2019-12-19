// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

func newServerClientPair(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *elasticsearch.Client) {
	mux := http.NewServeMux()
	mux.Handle("/_license/", http.HandlerFunc(handler))

	server := httptest.NewServer(mux)

	client, err := elasticsearch.NewClient(elasticsearch.ClientSettings{URL: server.URL}, nil)
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
		oss, err := fetcher.Fetch()
		if assert.NoError(t, err) {
			return
		}

		assert.Equal(t, OSSLicense, oss)
	})

	t.Run("OSS release of Elasticsearch (Code: 400)", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Bad Request", 400)
		}
		s, c := newServerClientPair(t, h)
		defer s.Close()
		defer c.Close()

		fetcher := NewElasticFetcher(c)
		oss, err := fetcher.Fetch()
		if assert.NoError(t, err) {
			return
		}

		assert.Equal(t, OSSLicense, oss)
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

	t.Run("parse milliseconds", func(t *testing.T) {
		t.Run("invalid", func(t *testing.T) {
			b := []byte("{ \"v\": \"\"}")
			ts := struct {
				V expiryTime `json:"v"`
			}{}

			err := json.Unmarshal(b, &ts)
			assert.Error(t, err)
		})

		t.Run("valid", func(t *testing.T) {
			b := []byte("{ \"v\": 1538060781728 }")
			ts := struct {
				V expiryTime `json:"v"`
			}{}

			err := json.Unmarshal(b, &ts)
			if !assert.NoError(t, err) {
				return
			}

			// 2018-09-27 15:06:21.728 +0000 UTC
			d := time.Date(2018, 9, 27, 15, 6, 21, 728000000, time.UTC).Sub((time.Time(ts.V)))
			assert.Equal(t, time.Duration(0), d)
		})
	})
}
