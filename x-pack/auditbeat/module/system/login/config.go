// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package login

// config defines the metricset's configuration options.
type config struct {
	WtmpFilePattern string `config:"login.wtmp_file_pattern"`
	BtmpFilePattern string `config:"login.btmp_file_pattern"`
}

func defaultConfig() config {
	return config{
		WtmpFilePattern: "/var/log/wtmp*",
		BtmpFilePattern: "/var/log/btmp*",
	}
}
