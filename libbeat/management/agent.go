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

package management

import (
	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

var (
	// underAgent is set to true with this beat is being ran under the elastic-agent
	underAgent = atomic.MakeBool(false)

	// underAgentTrace is set to true when the elastic-agent has placed this beat into
	// trace mode (which enables logging of published events)
	underAgentTrace = atomic.MakeBool(false)
)

// SetUnderAgent sets that the processing pipeline is being ran under the elastic-agent.
func SetUnderAgent(val bool) {
	underAgent.Store(val)
}

// SetUnderAgentTrace sets that trace mode has been enabled by the elastic-agent.
//
// SetUnderAgent must also be called and set to true before this has an effect.
func SetUnderAgentTrace(val bool) {
	underAgentTrace.Store(val)
}

// UnderAgent returns true when running under Elastic Agent.
func UnderAgent() bool {
	return underAgent.Load()
}

// TraceLevelEnabled returns true when the "trace log level" is enabled.
//
// It always returns true when not running under Elastic Agent.
// Otherwise it returns true when the trace level is enabled
func TraceLevelEnabled() bool {
	if underAgent.Load() {
		return underAgentTrace.Load()
	}

	// Always true when not running under the Elastic Agent.
	return true
}
