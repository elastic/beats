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

package apmhttp

import (
	"net/http"
	"regexp"
	"sync"

	"go.elastic.co/apm/internal/configutil"
	"go.elastic.co/apm/internal/wildcard"
)

const (
	envIgnoreURLs = "ELASTIC_APM_IGNORE_URLS"
)

var (
	defaultServerRequestIgnorerOnce sync.Once
	defaultServerRequestIgnorer     RequestIgnorerFunc = IgnoreNone
)

// DefaultServerRequestIgnorer returns the default RequestIgnorer to use in
// handlers. If ELASTIC_APM_IGNORE_URLS is set, it will be treated as a
// comma-separated list of wildcard patterns; requests that match any of the
// patterns will be ignored.
func DefaultServerRequestIgnorer() RequestIgnorerFunc {
	defaultServerRequestIgnorerOnce.Do(func() {
		matchers := configutil.ParseWildcardPatternsEnv(envIgnoreURLs, nil)
		if len(matchers) != 0 {
			defaultServerRequestIgnorer = NewWildcardPatternsRequestIgnorer(matchers)
		}
	})
	return defaultServerRequestIgnorer
}

// NewRegexpRequestIgnorer returns a RequestIgnorerFunc which matches requests'
// URLs against re. Note that for server requests, typically only Path and
// possibly RawQuery will be set, so the regular expression should take this
// into account.
func NewRegexpRequestIgnorer(re *regexp.Regexp) RequestIgnorerFunc {
	if re == nil {
		panic("re == nil")
	}
	return func(r *http.Request) bool {
		return re.MatchString(r.URL.String())
	}
}

// NewWildcardPatternsRequestIgnorer returns a RequestIgnorerFunc which matches
// requests' URLs against any of the matchers. Note that for server requests,
// typically only Path and possibly RawQuery will be set, so the wildcard patterns
// should take this into account.
func NewWildcardPatternsRequestIgnorer(matchers wildcard.Matchers) RequestIgnorerFunc {
	if len(matchers) == 0 {
		panic("len(matchers) == 0")
	}
	return func(r *http.Request) bool {
		return matchers.MatchAny(r.URL.String())
	}
}

// IgnoreNone is a RequestIgnorerFunc which ignores no requests.
func IgnoreNone(*http.Request) bool {
	return false
}
