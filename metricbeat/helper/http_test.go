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

package helper

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAuthHeaderFromToken(t *testing.T) {
	tests := []struct {
		Name, Content, Expected string
	}{
		{
			"Test a token is read",
			"testtoken",
			"Bearer testtoken",
		},
		{
			"Test a token is trimmed",
			"testtoken\n",
			"Bearer testtoken",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			content := []byte(test.Content)
			tmpfile, err := ioutil.TempFile("", "token")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write(content); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			header, err := getAuthHeaderFromToken(tmpfile.Name())
			assert.NoError(t, err)
			assert.Equal(t, test.Expected, header)
		})
	}
}

func TestGetAuthHeaderFromTokenNoFile(t *testing.T) {
	header, err := getAuthHeaderFromToken("nonexistingfile")
	assert.Equal(t, "", header)
	assert.Error(t, err)
}

func TestAddBasePath(t *testing.T) {
	tests := []struct {
		Name, Content, Expected string
	}{
		{
			"Test with empty basepath",
			"",
			"http://localhost:9999/some/path",
		},
		{
			"Test with basepath with no leading or trailing slashes",
			"foobar",
			"http://localhost:9999/foobar/some/path",
		},
		{
			"Test with basepath with a leading slash",
			"/foobar",
			"http://localhost:9999/foobar/some/path",
		},
		{
			"Test with basepath with a trailing slash",
			"foobar/",
			"http://localhost:9999/foobar/some/path",
		},
		{
			"Test with basepath with leading and trailing slashes",
			"/foobar/",
			"http://localhost:9999/foobar/some/path",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			http := HTTP{
				uri: "http://localhost:9999/some/path",
			}

			http.AddBasePath(test.Content)
			assert.Equal(t, test.Expected, http.GetURI())
		})
	}
}
