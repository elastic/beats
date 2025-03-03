// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package etw

type Config struct {
	Logfile         string // Path to the logfile
	ProviderGUID    string // GUID of the ETW provider
	ProviderName    string // Name of the ETW provider
	SessionName     string // Name for new ETW session
	TraceLevel      string // Level of tracing (e.g., "verbose")
	MatchAnyKeyword uint64 // Filter for any matching keywords (bitmask)
	MatchAllKeyword uint64 // Filter for all matching keywords (bitmask)
	Session         string // Existing session to attach
}
