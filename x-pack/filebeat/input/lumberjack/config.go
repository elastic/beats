// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type config struct {
	ListenAddress  string                  `config:"listen_address" validate:"nonzero"` // Bind address for the server (e.g. address:port). Default to localhost:5044.
	Versions       []string                `config:"versions"`                          // List of Lumberjack version (e.g. v1, v2).
	TLS            *tlscommon.ServerConfig `config:"ssl"`                               // TLS options.
	Keepalive      time.Duration           `config:"keepalive"       validate:"min=0"`  // Keepalive interval for notifying clients that batches that are not yet ACKed.
	Timeout        time.Duration           `config:"timeout"         validate:"min=0"`  // Read / write timeouts for Lumberjack server.
	MaxConnections int                     `config:"max_connections" validate:"min=0"`  // Maximum number of concurrent connections. Default is 0 which means no limit.
}

func (c *config) InitDefaults() {
	c.ListenAddress = "localhost:5044"
	c.Versions = []string{"v1", "v2"}
}

func (c *config) Validate() error {
	for _, v := range c.Versions {
		switch strings.ToLower(v) {
		case "v1", "v2":
		default:
			return fmt.Errorf("invalid lumberjack version %q: allowed values are v1 and v2", v)
		}
	}

	return nil
}
