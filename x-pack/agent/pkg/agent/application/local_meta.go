// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"os"
	"runtime"

	"github.com/elastic/beats/agent/release"
)

func metadata() (map[string]interface{}, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"platform": runtime.GOOS,
		"version":  release.Version(),
		"host":     hostname,
	}, nil
}
