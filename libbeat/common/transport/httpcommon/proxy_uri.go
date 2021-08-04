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
	"net/url"
	"strings"
)

// ProxyURI is a URL used for proxy serialized as a string.
type ProxyURI url.URL

func NewProxyURIFromString(s string) (*ProxyURI, error) {
	if s == "" {
		return nil, nil
	}

	u, err := url.Parse(s)
	if err != nil || u == nil {
		return nil, err
	}

	return NewProxyURIFromURL(*u), nil
}

func NewProxyURIFromURL(u url.URL) *ProxyURI {
	if u == (url.URL{}) {
		return nil
	}

	p := ProxyURI(u)
	return &p
}

// MarshalYAML serializes URI as a string.
func (p *ProxyURI) MarshalYAML() (interface{}, error) {
	u := url.URL(*p)
	return u.String(), nil
}

// MarshalJSON serializes URI as a string.
func (p *ProxyURI) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// Unpack unpacks string into an proxy URI.
func (p *ProxyURI) Unpack(s string) error {
	uri, err := NewProxyURIFromString(s)
	if err != nil {
		return err
	}

	*p = *uri
	return nil
}

// UnmarshalJSON unpacks string into an proxy URI.
func (p *ProxyURI) UnmarshalJSON(b []byte) error {
	unqoted := strings.Trim(string(b), `"`)
	uri, err := NewProxyURIFromString(unqoted)
	if err != nil {
		return err
	}

	*p = *uri
	return nil
}

// UnmarshalYAML unpacks string into an proxy URI.
func (p *ProxyURI) UnmarshalYAML(unmarshal func(interface{}) error) error {
	rawURI := ""
	if err := unmarshal(&rawURI); err != nil {
		return err
	}

	uri, err := NewProxyURIFromString(rawURI)
	if err != nil {
		return err
	}

	*p = *uri
	return nil
}

// URI returns conventional url.URL structure.
func (p *ProxyURI) URI() *url.URL {
	return (*url.URL)(p)
}

// MarshalJSON serializes URI as a string.
func (p *ProxyURI) String() string {
	return p.URI().String()
}
