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
	"net/http"

	"github.com/elastic/beats/v8/libbeat/common"
)

// ProxyHeaders is a headers for proxy serialized as a map[string]string.
type ProxyHeaders map[string]string

// MarshalYAML serializes URI as a string.
func (p ProxyHeaders) MarshalYAML() (interface{}, error) {
	return p, nil
}

// MarshalJSON serializes URI as a string.
func (p ProxyHeaders) MarshalJSON() ([]byte, error) {
	var m map[string]string = p
	return json.Marshal(m)
}

// Unpack unpacks string into an proxy URI.
func (p *ProxyHeaders) Unpack(cfg *common.Config) error {
	m := make(map[string]string)
	if err := cfg.Unpack(&m); err != nil {
		return err
	}

	*p = m
	return nil
}

// UnmarshalJSON unpacks string into an proxy URI.
func (p *ProxyHeaders) UnmarshalJSON(b []byte) error {
	m := make(map[string]string)
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	*p = m
	return nil
}

// UnmarshalYAML unpacks string into an proxy URI.
func (p *ProxyHeaders) UnmarshalYAML(unmarshal func(interface{}) error) error {
	m := make(map[string]string)
	if err := unmarshal(&m); err != nil {
		return err
	}

	*p = m
	return nil
}

// URI returns conventional url.URL structure.
func (p ProxyHeaders) Headers() http.Header {
	var httpHeaders http.Header
	if p != nil && len(p) > 0 {
		httpHeaders = http.Header{}
		for k, v := range p {
			httpHeaders.Add(k, v)
		}
	}

	return httpHeaders
}
