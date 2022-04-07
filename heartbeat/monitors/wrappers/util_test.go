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

package wrappers

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"

	"github.com/elastic/beats/v8/libbeat/common"
)

func TestURLFields(t *testing.T) {
	tests := []struct {
		name string
		u    string
		want common.MapStr
	}{
		{
			"simple-http",
			"http://elastic.co",
			common.MapStr{
				"full":   "http://elastic.co",
				"scheme": "http",
				"domain": "elastic.co",
				"port":   uint16(80),
			},
		},
		{
			"simple-https",
			"https://elastic.co",
			common.MapStr{
				"full":   "https://elastic.co",
				"scheme": "https",
				"domain": "elastic.co",
				"port":   uint16(443),
			},
		},
		{
			"fancy-proto",
			"tcp+ssl://elastic.co:1234",
			common.MapStr{
				"full":   "tcp+ssl://elastic.co:1234",
				"scheme": "tcp+ssl",
				"domain": "elastic.co",
				"port":   uint16(1234),
			},
		},
		{
			"complex",
			"tcp+ssl://myuser:mypass@elastic.co:65500/foo/bar?q=dosomething&x=y",
			common.MapStr{
				"full":     "tcp+ssl://myuser:%3Chidden%3E@elastic.co:65500/foo/bar?q=dosomething&x=y",
				"scheme":   "tcp+ssl",
				"domain":   "elastic.co",
				"port":     uint16(65500),
				"path":     "/foo/bar",
				"query":    "q=dosomething&x=y",
				"username": "myuser",
				"password": "<hidden>",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.u)
			require.NoError(t, err)

			got := URLFields(parsed)
			testslike.Test(t, lookslike.MustCompile(map[string]interface{}(tt.want)), got)
		})
	}
}
