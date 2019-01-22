// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package login

// config defines the metricset's configuration options.
type config struct {
	UtmpFilePattern string `config:"login.utmp_file_pattern"`
}

func defaultConfig() config {
	return config{
		UtmpFilePattern: "/var/log/wtmp*",
	}
}
