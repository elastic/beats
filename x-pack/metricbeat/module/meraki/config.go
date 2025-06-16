// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package meraki

import (
	"fmt"
	"time"
)

type config struct {
	BaseURL       string        `config:"apiBaseURL"`
	ApiKey        string        `config:"apiKey"`
	DebugMode     string        `config:"apiDebugMode"`
	Organizations []string      `config:"organizations"`
	Period        time.Duration `config:"period"`
	// todo: device/network filtering?
}

func DefaultConfig() *config {
	return &config{
		BaseURL:   "https://api.meraki.com",
		DebugMode: "false",
		Period:    time.Second * 300,
	}
}

func (c *config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("apiBaseURL is required")
	}

	if c.ApiKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	if c.Organizations == nil || len(c.Organizations) == 0 {
		return fmt.Errorf("organizations is required")
	}

	// the reason for this is due to restrictions imposed by some dashboard API endpoints.
	// for example, "/api/v1/organizations/{organizationId}/devices/uplinksLossAndLatency"
	// has a maximum 'timespan' of 5 minutes.
	if c.Period.Seconds() > 300 {
		return fmt.Errorf("the maximum allowed collection period is 5 minutes (300s)")
	}

	return nil
}
