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

package transport // import "go.elastic.co/apm/transport"

var (
	// Default is the default Transport, using the
	// ELASTIC_APM_* environment variables.
	//
	// If ELASTIC_APM_SERVER_URL is set to an invalid
	// location, Default will be set to a Transport
	// returning an error for every operation.
	Default Transport

	// Discard is a Transport on which all operations
	// succeed without doing anything.
	Discard = discardTransport{}
)

func init() {
	_, _ = InitDefault()
}

// InitDefault (re-)initializes Default, the default Transport, returning
// its new value along with the error that will be returned by the Transport
// if the environment variable configuration is invalid. The result is always
// non-nil.
func InitDefault() (Transport, error) {
	t, err := getDefault()
	Default = t
	return t, err
}

func getDefault() (Transport, error) {
	s, err := NewHTTPTransport()
	if err != nil {
		return discardTransport{err}, err
	}
	return s, nil
}
