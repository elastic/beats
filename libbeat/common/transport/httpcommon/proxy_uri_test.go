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

package httpcommon

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestYamlSerializeDeserialize(t *testing.T) {
	raw := "http://localhost:8080/path\n"
	var proxyURI *ProxyURI
	err := yaml.Unmarshal([]byte(raw), &proxyURI)
	require.NoError(t, err)

	out, err := yaml.Marshal(proxyURI)
	require.NoError(t, err)
	require.Equal(t, raw, string(out))
}

func TestJSONSerializeDeserialize(t *testing.T) {
	raw := `{"proxy_uri":"http://localhost:8080/path"}`
	proxyURI, err := NewProxyURIFromString("http://localhost:8080/path")
	require.NoError(t, err)
	s := struct {
		P *ProxyURI `json:"proxy_uri"`
	}{
		P: proxyURI,
	}
	err = json.Unmarshal([]byte(raw), &s)
	require.NoError(t, err)

	out, err := json.Marshal(s)
	require.NoError(t, err)
	require.Equal(t, raw, string(out))
}

func TestYamlSerializeDeserializeSettings(t *testing.T) {
	raw := `proxy_url: http://localhost:8080/path
proxy_headers:
  key: val
proxy_disable: true
`

	s := &HTTPClientProxySettings{}
	err := yaml.Unmarshal([]byte(raw), &s)
	require.NoError(t, err)

	out, err := yaml.Marshal(s)
	require.NoError(t, err)
	require.Equal(t, raw, string(out))
}
