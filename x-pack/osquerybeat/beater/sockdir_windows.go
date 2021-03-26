// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package beater

import "github.com/elastic/beats/v7/libbeat/logp"

func createSockDir(log *logp.Logger) (string, func(), error) {
	// Noop on winders
	return "", func() {
	}, nil
}
