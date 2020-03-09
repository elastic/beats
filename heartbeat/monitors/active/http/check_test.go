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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/conditions"
)

func TestCheckBody(t *testing.T) {

	var matchTests = []struct {
		description string
		body        string
		patterns    []string
		result      bool
	}{
		{
			"Single regex that matches",
			"ok",
			[]string{"ok"},
			true,
		},
		{
			"Regex matching json example",
			`{"status": "ok"}`,
			[]string{`{"status": "ok"}`},
			true,
		},
		{
			"Regex matching first line of multiline body string",
			`first line
			second line`,
			[]string{"first"},
			true,
		},
		{
			"Regex matching lastline of multiline body string",
			`first line
			second line`,
			[]string{"second"},
			true,
		},
		{
			"Regex matching multiple lines of multiline body string",
			`first line
			second line
			third line`,
			[]string{"(?s)first.*second.*third"},
			true,
		},
		{
			"Regex not matching multiple lines of multiline body string",
			`first line
			second line
			third line`,
			[]string{"(?s)first.*fourth.*third"},
			false,
		},
		{
			"Single regex that doesn't match",
			"ok",
			[]string{"notok"},
			false,
		},
		{
			"Multiple regex match where at least one must match",
			"ok",
			[]string{"ok", "yay"},
			true,
		},
		{
			"Multiple regex match where none of the patterns match",
			"ok",
			[]string{"notok", "yay"},
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

			patterns := []match.Matcher{}
			for _, pattern := range test.patterns {
				patterns = append(patterns, match.MustCompile(pattern))
			}
			body, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			check := checkBody(patterns)(res, string(body))

			if result := (check == nil); result != test.result {
				if test.result {
					t.Fatalf("Expected at least one of patterns: %s to match body: %s", test.patterns, test.body)
				} else {
					t.Fatalf("Did not expect patterns: %s to match body: %s", test.patterns, test.body)
				}
			}
		})
	}
}

func TestCheckJson(t *testing.T) {
	fooBazEqualsBar := common.MustNewConfigFrom(map[string]interface{}{"equals": map[string]interface{}{"foo": map[string]interface{}{"baz": "bar"}}})
	fooBazEqualsBarConf := &conditions.Config{}
	err := fooBazEqualsBar.Unpack(fooBazEqualsBarConf)
	require.NoError(t, err)

	fooBazEqualsBarDesc := "foo.baz equals bar"

	var tests = []struct {
		description string
		body        string
		condDesc    string
		condConf    *conditions.Config
		result      bool
	}{
		{
			"positive match",
			"{\"foo\": {\"baz\": \"bar\"}}",
			fooBazEqualsBarDesc,
			fooBazEqualsBarConf,
			true,
		},
		{
			"Negative match",
			"{\"foo\": 123}",
			fooBazEqualsBarDesc,
			fooBazEqualsBarConf,
			false,
		},
		{
			"unparseable",
			`notjson`,
			fooBazEqualsBarDesc,
			fooBazEqualsBarConf,
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, test.body)
			}))
			defer ts.Close()

			res, err := http.Get(ts.URL)
			if err != nil {
				log.Fatal(err)
			}

			checker, err := checkJSON([]*jsonResponseCheck{{test.condDesc, test.condConf}})
			require.NoError(t, err)
			body, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			checkRes := checker(res, string(body))

			if result := checkRes == nil; result != test.result {
				if test.result {
					t.Fatalf("Expected condition: '%s' to match body: %s. got: %s", test.condDesc, test.body, checkRes)
				} else {
					t.Fatalf("Did not expect condition: '%s' to match body: %s. got: %s", test.condDesc, test.body, checkRes)
				}
			}
		})
	}

}

func TestCheckJsonWithIntegerComparison(t *testing.T) {
	fooBazEqualsBar := common.MustNewConfigFrom(map[string]interface{}{"equals": map[string]interface{}{"foo": 1}})
	fooBazEqualsBarConf := &conditions.Config{}
	err := fooBazEqualsBar.Unpack(fooBazEqualsBarConf)
	require.NoError(t, err)

	fooBazEqualsBarDesc := "foo equals 1"

	var tests = []struct {
		description string
		body        string
		condDesc    string
		condConf    *conditions.Config
		result      bool
	}{
		{
			"positive match",
			"{\"foo\": 1}",
			fooBazEqualsBarDesc,
			fooBazEqualsBarConf,
			true,
		},
		{
			"Negative match",
			"{\"foo\": 2}",
			fooBazEqualsBarDesc,
			fooBazEqualsBarConf,
			false,
		},
		{
			"Negative match",
			"{\"foo\": \"some string\"}",
			fooBazEqualsBarDesc,
			fooBazEqualsBarConf,
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, test.body)
			}))
			defer ts.Close()

			res, err := http.Get(ts.URL)
			if err != nil {
				log.Fatal(err)
			}

			checker, err := checkJSON([]*jsonResponseCheck{{test.condDesc, test.condConf}})
			require.NoError(t, err)
			body, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			checkRes := checker(res, string(body))

			if result := checkRes == nil; result != test.result {
				if test.result {
					t.Fatalf("Expected condition: '%s' to match body: %s. got: %s", test.condDesc, test.body, checkRes)
				} else {
					t.Fatalf("Did not expect condition: '%s' to match body: %s. got: %s", test.condDesc, test.body, checkRes)
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
