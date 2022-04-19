// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeZip(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	var files = []struct {
		Name, Body string
	}{
		{
			"a.json",
			`{"a":"b"}`,
		},
		{
			"b.ndjson",
			`{"a":"b"}` + "\n" + `{"c":"d"}`,
		},
		{
			"c.ndjson",
			"{\n\t" + `"a":"b"` + "\n}\n{\n\t" + `"c":"d"` + "\n}",
		},
	}
	for _, file := range files {
		f, err := w.Create(file.Name)
		require.NoError(t, err)
		_, err = f.Write([]byte(file.Body))
		require.NoError(t, err)
	}

	// Make sure to check the error on Close.
	require.NoError(t, w.Close())

	const expected = `[{"a":"b"},{"a":"b"},{"c":"d"},{"a":"b"},{"c":"d"}]`

	resp := &response{}
	if err := decodeAsZip(buf.Bytes(), resp); err != nil {
		t.Fatalf("decodeAsZip failed: %v", err)
	}

	j, err := json.Marshal(resp.body)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	assert.Equal(t, expected, string(j))
}

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

func TestDecodeCSV(t *testing.T) {
	tests := []struct {
		body   string
		result string
		err    string
	}{
		{"", "", ""},
		{
			"EVENT_TYPE,TIMESTAMP,REQUEST_ID,ORGANIZATION_ID,USER_ID\n" +
				"Login,20211018071353.465,id1,id2,user1\n" +
				"Login,20211018071505.579,id4,id5,user2\n",
			`[{"EVENT_TYPE":"Login","TIMESTAMP":"20211018071353.465","REQUEST_ID":"id1","ORGANIZATION_ID":"id2","USER_ID":"user1"},
			{"EVENT_TYPE":"Login","TIMESTAMP":"20211018071505.579","REQUEST_ID":"id4","ORGANIZATION_ID":"id5","USER_ID":"user2"}]`,
			"",
		},
		{
			"EVENT_TYPE,TIMESTAMP,REQUEST_ID,ORGANIZATION_ID,USER_ID\n" +
				"Login,20211018071505.579,id4,user2\n",
			"",
			"record on line 2: wrong number of fields",
		},
	}
	for _, test := range tests {
		resp := &response{}
		err := decodeAsCSV([]byte(test.body), resp)
		if test.err != "" {
			assert.Error(t, err)
			assert.EqualError(t, err, test.err)
		} else {
			assert.NoError(t, err)

			var j []byte
			if test.body != "" {
				j, err = json.Marshal(resp.body)
				if err != nil {
					t.Fatalf("Marshal failed: %v", err)
				}
				assert.JSONEq(t, test.result, string(j))
			} else {
				assert.Equal(t, test.result, string(j))
			}
		}
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
		assert.NoError(t, err)
		assert.Equal(t, test.body, string(res))
		assert.Equal(t, "application/x-www-form-urlencoded", trReq.header().Get("Content-Type"))
	}
}
