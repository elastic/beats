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

package inputmon

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

type TestCase struct {
	request string
	status  int
	method  string
	body    string
}

var testCases = []TestCase{
	{request: "/inputs/", status: http.StatusOK, body: `[{"gauge":13344,"id":"123abc","input":"foo"}]`},
	{request: "/inputs", status: http.StatusOK},
	{request: "/inputs/", method: "POST", status: http.StatusMethodNotAllowed},
	{request: "/inputs/?XX", status: http.StatusBadRequest},
	{request: "/inputs/?pretty", status: http.StatusOK},
	{request: "/inputs/?type", status: http.StatusBadRequest},
	{request: "/inputs/?type=udp", status: http.StatusOK, body: `[]`},
	{request: "/inputs/?type=FOO", status: http.StatusOK, body: `[{"gauge":13344,"id":"123abc","input":"foo"}]`},
	{request: "/inputs/XX", status: http.StatusNotFound},
}

func TestHandler(t *testing.T) {
	parent := monitoring.NewRegistry()
	reg, _ := NewInputRegistry("foo", "123abc", parent)
	monitoring.NewInt(reg, "gauge").Set(13344)

	// Register legacy metrics without id or input. This must be ignored.
	{
		legacy := parent.NewRegistry("f49c0680-fc5f-4b78-bd98-7b16628f9a77")
		monitoring.NewString(legacy, "name").Set("/var/log/wifi.log")
		monitoring.NewTimestamp(legacy, "last_event_published_time").Set(time.Now())
	}

	r := mux.NewRouter()
	s := httptest.NewServer(r)
	defer s.Close()

	if err := attachHandler(r, parent); err != nil {
		t.Fatal(err)
	}

	t.Logf("http://%s", s.Listener.Addr().String())

	for _, tc := range testCases {
		tc := tc
		if tc.method == "" {
			tc.method = http.MethodGet
		}

		t.Run(tc.method+" "+strings.ReplaceAll(tc.request, "/", "_"), func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), tc.method, s.URL+tc.request, nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := s.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("body=%s", body)
			if resp.StatusCode != tc.status {
				t.Fatalf("bad status code, want=%d, got=%d", tc.status, resp.StatusCode)
			}

			if tc.body != "" {
				assert.JSONEq(t, tc.body, string(body))
			}
		})
	}
}

func BenchmarkHandlers(b *testing.B) {
	reg := monitoring.NewRegistry()
	for i := 0; i < 1000; i++ {
		reg, _ := NewInputRegistry("foo", "id-"+strconv.Itoa(i), reg)
		monitoring.NewInt(reg, "gauge").Set(int64(i))
	}

	h := &handler{registry: reg}

	b.Run("allInputs", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/inputs/", nil)
		resp := httptest.NewRecorder()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.allInputs(resp, req)
		}
	})
}
