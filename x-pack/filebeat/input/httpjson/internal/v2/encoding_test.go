// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeNdjson(t *testing.T) {
	tests := []struct {
		body   string
		result string
	}{
		{"{}", "[{}]"},
		{"{\"a\":\"b\"}", "[{\"a\":\"b\"}]"},
		{"{\"a\":\"b\"}\n{\"c\":\"d\"}", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
		{"{\"a\":\"b\"}\r\n{\"c\":\"d\"}", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
		{"{\"a\":\"b\"}\r\n{\"c\":\"d\"}\n", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
		{"{\"a\":\"b\"}\r\n{\"c\":\"d\"}\r\n", "[{\"a\":\"b\"},{\"c\":\"d\"}]"},
	}
	for _, test := range tests {
		resp := &response{}
		err := decodeAsNdjson([]byte(test.body), resp)
		if err != nil {
			t.Fatalf("decodeAsNdjson failed: %v", err)
		}
		j, err := json.Marshal(resp.body)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		assert.Equal(t, test.result, string(j))
	}
}

func TestEncodeAsForm(t *testing.T) {
	tests := []struct {
		params map[string]string
		body   string
	}{
		{map[string]string{"a": "b"}, "a=b"},
		{map[string]string{"a": "b", "c": "d"}, "a=b&c=d"},
		{nil, ""},
	}
	for _, test := range tests {
		u, err := url.Parse("http://localhost")
		if err != nil {
			t.Fatalf("url parse failed: %v", err)
		}
		q := u.Query()
		for k, v := range test.params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		trReq := transformable{}
		trReq.setURL(*u)
		res, err := encodeAsForm(trReq)
		assert.Equal(t, test.body, string(res))
		assert.Equal(t, "application/x-www-form-urlencoded", trReq.header().Get("Content-Type"))
	}
}
