// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

type Config struct {
	Logfile         string
	ProviderGUID    string
	ProviderName    string
	SessionName     string // Tag for the new session
	TraceLevel      string
	MatchAnyKeyword uint64
	MatchAllKeyword uint64
	Session         string
}
