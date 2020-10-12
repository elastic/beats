// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// Config is a configuration of GRPC server.
type Config struct {
	Address string `config:"address"`
	Port    uint16 `config:"port"`
}

// DefaultGRPCConfig creates a default server configuration.
func DefaultGRPCConfig() *Config {
	return &Config{
		Address: "localhost",
		Port:    6789,
	}
}

// NewFromConfig creates a new GRPC server for clients to connect to.
func NewFromConfig(logger *logger.Logger, cfg *Config, handler Handler) (*Server, error) {
	return New(logger, fmt.Sprintf("%s:%d", cfg.Address, cfg.Port), handler)
}
