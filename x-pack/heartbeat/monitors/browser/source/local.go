// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package source

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
)

type LocalSource struct {
}

var ErrLocalUnsupportedType = fmt.Errorf("local %w", ErrUnsupportedSource)

func (l *LocalSource) Validate() (err error) {
	return ecserr.NewUnsupportedMonitorTypeError(ErrLocalUnsupportedType)
}

func (l *LocalSource) Fetch() (err error) {
	return ecserr.NewUnsupportedMonitorTypeError(ErrLocalUnsupportedType)
}

func (l *LocalSource) Workdir() string {
	return ""
}

func (l *LocalSource) Close() error {
	return ecserr.NewUnsupportedMonitorTypeError(ErrLocalUnsupportedType)
}
