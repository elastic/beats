// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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
	GUID            string   // GUID of the ETW provider
	Name            string   // Name of the ETW provider
	TraceLevel      string   // Level of tracing (e.g., "verbose")
	MatchAnyKeyword uint64   // Filter for any matching keywords (bitmask)
	MatchAllKeyword uint64   // Filter for all matching keywords (bitmask)
	EnableProperty  []string // Properties to enable for the session
	EnableFlags     uint32   // Bitmask for enabling flags on kernel sessions
}
