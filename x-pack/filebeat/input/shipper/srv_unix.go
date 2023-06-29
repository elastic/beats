// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package shipper

import (
	"net"

	"github.com/elastic/elastic-agent-libs/logp"
)

func newListener(_ *logp.Logger, addr string) (net.Listener, error) {
	return net.Listen("unix", addr)
}
