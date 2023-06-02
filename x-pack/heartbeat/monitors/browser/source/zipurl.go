// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package source

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
)

type ZipURLSource struct {
}

var ErrZipURLUnsupportedType = fmt.Errorf("zip_url %w", ErrUnsupportedSource)

func (z *ZipURLSource) Validate() (err error) {
	return ecserr.NewUnsupportedMonitorTypeError(ErrZipURLUnsupportedType)
}

func (z *ZipURLSource) Fetch() error {
	return ecserr.NewUnsupportedMonitorTypeError(ErrZipURLUnsupportedType)
}

func (z *ZipURLSource) Workdir() string {
	return ""
}

func (z *ZipURLSource) Close() error {
	return ecserr.NewUnsupportedMonitorTypeError(ErrZipURLUnsupportedType)
}
