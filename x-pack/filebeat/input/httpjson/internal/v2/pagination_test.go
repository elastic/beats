// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPage(t *testing.T) {
	var tests = []struct {
		body   string
		result string
	}{
		{"{\"a\":\"b\"}", "{\"a\":\"b\"}"},
		{"{\"a\":\"b\"}\n{\"c\":\"d\"}", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
		{"{\"a\":\"b\"}\r\n{\"c\":\"d\"}", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
		{"{\"a\":\"b\"}\r\n{\"c\":\"d\"}\n", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
		{"{\"a\":\"b\"}\r\n{\"c\":\"d\"}\r\n", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
	}

	for _, test := range tests {
		iter := &pageIterator{}
		resp := &http.Response{}
		resp.Body = ioutil.NopCloser(strings.NewReader(test.body))
		req, err := http.NewRequest("GET", "http://localhost", nil)
		if err != nil {
			t.Fatalf("NewRequest failed: %v", err)
		}
		resp.Request = req
		iter.resp = resp
		r, err := iter.getPage()
		if err != nil {
			t.Fatalf("getPage failed: %v", err)
		}
		j, err := json.Marshal(r.body)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		assert.Equal(t, test.result, string(j))
	}
}
