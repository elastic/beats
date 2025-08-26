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

package fleetmode

var managementEnabled bool

// SetAgentMode stores if the Beat is running under Elastic Agent.
// Normally this is called when the command line flags are parsed.
// This is stored as a package level variable because some components
// (like filebeat/metricbeat modules) don't have access to the
// configuration information to determine this on their own.
func SetAgentMode(enabled bool) {
	managementEnabled = enabled
}

// Enabled returns true if the Beat is running under Elastic Agent.
func Enabled() bool {
	return managementEnabled
}
