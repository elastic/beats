// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"os"
	"sync"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

var (
	gLogger *logger.Logger
	logOnce sync.Once
)

func setLogger(log *logger.Logger) {
	logOnce.Do(func() {
		gLogger = log
	})
}

func getLogger() *logger.Logger {
	logOnce.Do(func() {
		gLogger = logger.New(os.Stderr, true)
	})
	return gLogger
}
