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

package kibana

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorJson(t *testing.T) {
	// also common 200: {"objects":[{"id":"apm-*","type":"index-pattern","error":{"message":"[doc][index-pattern:test-*]: version conflict, document already exists (current version [1])"}}]}
	kibanaTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"objects":[{"id":"test-*","type":"index-pattern","error":{"message":"action [indices:data/write/bulk[s]] is unauthorized for user [test]"}}]}`))
	}))
	defer kibanaTs.Close()

	conn := Connection{
		URL:  kibanaTs.URL,
		HTTP: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, nil, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Error(t, err)
}

func TestErrorBadJson(t *testing.T) {
	kibanaTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{`))
	}))
	defer kibanaTs.Close()

	conn := Connection{
		URL:  kibanaTs.URL,
		HTTP: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, nil, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Error(t, err)
}

func TestSuccess(t *testing.T) {
	kibanaTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"objects":[{"id":"test-*","type":"index-pattern","updated_at":"2018-01-24T19:04:13.371Z","version":1}]}`))

		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "bar", r.Header.Get("foo"))
	}))
	defer kibanaTs.Close()

	conn := Connection{
		URL:  kibanaTs.URL,
		HTTP: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, http.Header{"foo": []string{"bar"}}, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
}
