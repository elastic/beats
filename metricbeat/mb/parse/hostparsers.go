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

package parse

import (
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb"
)

// PassThruHostParser is a HostParser that sets the HostData URI, SanitizedURI,
// and Host to the configured 'host' value. This should only be used by
// MetricSets that do not require host parsing (e.g. host is only addr:port).
// Do not use this if the host value can contain credentials.
func PassThruHostParser(module mb.Module, host string) (mb.HostData, error) {
	return mb.HostData{URI: host, SanitizedURI: host, Host: host}, nil
}

// EmptyHostParser simply returns a zero value HostData. It asserts that host
// value is empty and returns an error if not.
func EmptyHostParser(module mb.Module, host string) (mb.HostData, error) {
	if host != "" {
		return mb.HostData{}, errors.Errorf("hosts must be empty for %v", module.Name())
	}

	return mb.HostData{}, nil
}
