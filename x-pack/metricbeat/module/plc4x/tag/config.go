// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tag

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// supportedDrivers are the PLC4X drivers this metricset registers. Only drivers
// covered by tests are enabled.
var supportedDrivers = []string{"modbus-tcp"}

type tagConfig struct {
	Address string `config:"address" validate:"required"`
	Name    string `config:"name"`
}

type config struct {
	Tags []tagConfig `config:"plc_tags" validate:"required"`
}

// Validate checks the configured tags.
func (c *config) Validate() error {
	if len(c.Tags) == 0 {
		return errors.New("at least one tag must be configured in plc_tags")
	}
	for i, t := range c.Tags {
		if strings.TrimSpace(t.Address) == "" {
			return fmt.Errorf("plc_tags[%d]: address is empty", i)
		}
		area, _, _ := strings.Cut(t.Address, ":")
		if unsupportedAreas[area] {
			return fmt.Errorf("plc_tags[%d]: %s addresses are not supported yet: the PLC4X version in use decodes Modbus coils and discrete inputs incorrectly; read holding or input registers instead", i, area)
		}
	}
	return nil
}

// unsupportedAreas are Modbus address areas that PLC4X v0.13 decodes
// incorrectly: a single bit fails to decode, and bit lists come back in reverse
// order within each byte.
var unsupportedAreas = map[string]bool{
	"coil":           true,
	"discrete-input": true,
}

// driverOf returns the PLC4X driver name of a connection string, such as
// modbus-tcp in modbus-tcp://plc.example.com:502.
func driverOf(connection string) (string, error) {
	driver, _, ok := strings.Cut(connection, "://")
	if !ok || driver == "" {
		return "", fmt.Errorf("invalid PLC4X connection string %q: it must look like <driver>://<address>", connection)
	}
	if slices.Contains(supportedDrivers, driver) {
		return driver, nil
	}
	return "", fmt.Errorf("unsupported PLC4X driver %q in %q, use one of %s", driver, connection, strings.Join(supportedDrivers, ", "))
}
