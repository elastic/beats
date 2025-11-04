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

package etw

type Config struct {
	Logfile        string           // Path to the logfile
	SessionName    string           // Name for new ETW session
	Session        string           // Existing session to attach
	BufferSize     uint32           // Kilobytes for the session buffer size
	MinimumBuffers uint32           // Minimum number of buffers for the session
	MaximumBuffers uint32           // Maximum number of buffers for the session
	Providers      []ProviderConfig // List of ETW providers to enable
}

type ProviderConfig struct {
	GUID            string      // GUID of the ETW provider
	Name            string      // Name of the ETW provider
	TraceLevel      string      // Level of tracing (e.g., "verbose")
	MatchAnyKeyword uint64      // Filter for any matching keywords (bitmask)
	MatchAllKeyword uint64      // Filter for all matching keywords (bitmask)
	EnableProperty  []string    // Properties to enable for the session
	EnableFlags     uint32      // Bitmask for enabling flags on kernel sessions
	EventFilter     EventFilter // Filters for events from the provider
}

type EventFilter struct {
	EventIDs []uint16 // Event IDs to filter
	FilterIn bool     // Whether to include or exclude these event IDs
}
