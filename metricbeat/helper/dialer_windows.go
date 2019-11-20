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

//+build windows

package helper

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/api/npipe"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/metricbeat/mb"
)

func makeDialer(t time.Duration, hostData mb.HostData) (transport.Dialer, string, error) {
	switch hostData.Transport {
	case mb.TransportUnix:
		return nil, "", fmt.Errorf(
			"cannot use %s as the URI, unix sockets are not supported on Windows, use npipe instead",
			hostData.SanitizedURI,
		)
	case mb.TransportNpipe:
		return npipe.DialContext(
			strings.TrimSuffix(npipe.TransformString(p), "/"),
		), hostData.SanitizedURI, nil
	default:
		return transport.NetDialer(t), hostData.SanitizedURI, nil
	}
}
