// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"os"
	"sync/atomic"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

var gLogger atomic.Pointer[logger.Logger]

func setLogger(log *logger.Logger) {
	gLogger.Store(log)
}

func getLogger() *logger.Logger {
	if l := gLogger.Load(); l != nil {
		return l
	}
	l := logger.New(os.Stderr, true)
	if gLogger.CompareAndSwap(nil, l) {
		return l
	}
	return gLogger.Load()
}
