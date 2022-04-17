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

package http

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common/match"
)

func TestCheckBody(t *testing.T) {

	var matchTests = []struct {
		description string
		body        string
		positive    []string
		negative    []string
		result      bool
	}{
		{
			"Single regex that matches",
			"ok",
			[]string{"ok"},
			nil,
			true,
		},
		{
			"Regex matching json example",
			`{"status": "ok"}`,
			[]string{`{"status": "ok"}`},
			nil,
			true,
		},
		{
			"Regex matching first line of multiline body string",
			`first line
			second line`,
			[]string{"first"},
			nil,
			true,
		},
		{
			"Regex matching lastline of multiline body string",
			`first line
			second line`,
			[]string{"second"},
			nil,
			true,
		},
		{
			"Regex matching multiple lines of multiline body string",
			`first line
			second line
			third line`,
			[]string{"(?s)first.*second.*third"},
			nil,
			true,
		},
		{
			"Regex not matching multiple lines of multiline body string",
			`first line
			second line
			third line`,
			[]string{"(?s)first.*fourth.*third"},
			nil,
			false,
		},
		{
			"Single regex that doesn't match",
			"ok",
			[]string{"notok"},
			nil,
			false,
		},
		{
			"Multiple regex match where at least one must match",
			"ok",
			[]string{"ok", "yay"},
			nil,
			true,
		},
		{
			"Multiple regex match where none of the patterns match",
			"ok",
			[]string{"notok", "yay"},
			nil,
			false,
		},
		{
			"Only positive check where pattern matches HTTP return body",
			"'status': 'red'",
			[]string{"red"},
			nil,
			true,
		},
		{
			"Only positive check where pattern not match HTTP return body",
			"'status': 'green'",
			[]string{"red"},
			nil,
			false,
		},
		{
			"Only negative check where pattern matches HTTP return body",
			"'status': 'red'",
			nil,
			[]string{"red"},
			false,
		},
		{
			"Only negative check where pattern not match HTTP return body",
			"'status': 'green'",
			nil,
			[]string{"red"},
			true,
		},
		{
			"Positive with negative check where all positive pattern matches and none negative check matches",
			"'status': 'green', 'cluster': 'healthy'",
			[]string{"green"},
			[]string{"unhealthy"},
			true,
		},
		{
			"Positive with negative check where positive and negative pattern both match",
			"'status': 'green', 'cluster': 'healthy'",
			[]string{"green"},
			[]string{"healthy"},
			false,
		},
		{
			"Positive and negative check are both empty",
			"'status': 'green', 'cluster': 'healthy'",
			nil,
			nil,
			false,
		},
	}

	for _, test := range matchTests {
		t.Run(test.description, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, test.body)
			}))
			defer ts.Close()

			res, err := http.Get(ts.URL)
			if err != nil {
				log.Fatal(err)
			}

			var positivePatterns []match.Matcher
			var negativePatterns []match.Matcher
			for _, p := range test.positive {
				positivePatterns = append(positivePatterns, match.MustCompile(p))
			}
			for _, n := range test.negative {
				negativePatterns = append(negativePatterns, match.MustCompile(n))
			}
			body, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			check := checkBody(positivePatterns, negativePatterns)(res, string(body))

			if result := check == nil; result != test.result {
				if test.result {
					t.Fatalf("Expected at least one of positive patterns or all negative patterns: %s %s to match body: %s", test.positive, test.negative, test.body)
				} else {
					t.Fatalf("Did not expect positive patterns or negative patterns: %s %s to match body: %s", test.positive, test.negative, test.body)
				}
			}
		})
	}
}

func TestCheckStatus(t *testing.T) {

	var matchTests = []struct {
		description string
		status      []uint16
		statusRec   int
		result      bool
	}{
		{
			"not match multiple values",
			[]uint16{200, 301, 302},
			500,
			false,
		},
		{
			"match multiple values",
			[]uint16{200, 301, 302},
			200,
			true,
		},
		{
			"not match single value",
			[]uint16{200},
			201,
			false,
		},
		{
			"match single value",
			[]uint16{200},
			200,
			true,
		},
	}

	for _, test := range matchTests {
		t.Run(test.description, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.statusRec)
			}))
			defer ts.Close()

			res, err := http.Get(ts.URL)
			if err != nil {
				log.Fatal(err)
			}

			check := checkStatus(test.status)(res)

			if result := (check == nil); result != test.result {
				if test.result {
					t.Fatalf("Expected at least one of status: %d to match status: %d", test.status, test.statusRec)
				} else {
					t.Fatalf("Did not expect status: %d to match status: %d", test.status, test.statusRec)
				}
			}
		})
	}
}
