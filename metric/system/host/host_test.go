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

package host

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo/types"
)

func TestMapHostInfo(t *testing.T) {
	bootTime := time.Now()
	containerized := true
	osInfo := types.OSInfo{
		Type:     "darwin",
		Family:   "family",
		Platform: "platform",
		Name:     "macos ventura",
		Version:  "13.2.1",
		Major:    13,
		Minor:    2,
		Patch:    1,
		Build:    "build",
		Codename: "ventura",
	}
	hostInfo := types.HostInfo{
		Architecture:      "x86_64",
		BootTime:          bootTime,
		Containerized:     &containerized,
		Hostname:          "fOo",
		IPs:               []string{"1.2.3.4", "192.168.1.1"},
		KernelVersion:     "22.3.0",
		MACs:              []string{"56:9c:17:54:19:15", "5c:e9:1e:c4:37:66"},
		OS:                &osInfo,
		Timezone:          "",
		TimezoneOffsetSec: 0,
		UniqueID:          "a39b4c1ee4",
	}

	tests := map[string]struct {
		fqdn     string
		expected mapstr.M
	}{
		"with_fqdn": {
			fqdn: "foo.bar.local",
			expected: mapstr.M{
				"host": mapstr.M{
					"architecture":  "x86_64",
					"containerized": true,
					"hostname":      "fOo",
					"id":            "a39b4c1ee4",
					"name":          "foo.bar.local",
					"os": mapstr.M{
						"build":    "build",
						"codename": "ventura",
						"family":   "family",
						"kernel":   "22.3.0",
						"name":     "macos ventura",
						"platform": "platform",
						"type":     "darwin",
						"version":  "13.2.1",
					},
				},
			},
		},
		"without_fqdn": {
			expected: mapstr.M{
				"host": mapstr.M{
					"architecture":  "x86_64",
					"containerized": true,
					"hostname":      "fOo",
					"id":            "a39b4c1ee4",
					"name":          "foo",
					"os": mapstr.M{
						"build":    "build",
						"codename": "ventura",
						"family":   "family",
						"kernel":   "22.3.0",
						"name":     "macos ventura",
						"platform": "platform",
						"type":     "darwin",
						"version":  "13.2.1",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := MapHostInfo(hostInfo, test.fqdn)
			require.Equal(t, test.expected, actual)
		})
	}
}
