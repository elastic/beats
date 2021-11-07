// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package download

import (
	"fmt"
	"time"

	"github.com/open-policy-agent/opa/plugins"
)

const (
	defaultMinDelaySeconds = int64(60)
	defaultMaxDelaySeconds = int64(120)
)

// PollingConfig represents polling configuration for the downloader.
type PollingConfig struct {
	MinDelaySeconds           *int64 `json:"min_delay_seconds,omitempty"`            // min amount of time to wait between successful poll attempts
	MaxDelaySeconds           *int64 `json:"max_delay_seconds,omitempty"`            // max amount of time to wait between poll attempts
	LongPollingTimeoutSeconds *int64 `json:"long_polling_timeout_seconds,omitempty"` // max amount of time the server should wait before issuing a timeout if there's no update available
}

// Config represents the configuration for the downloader.
type Config struct {
	Trigger *plugins.TriggerMode `json:"trigger,omitempty"`
	Polling PollingConfig        `json:"polling"`
}

// ValidateAndInjectDefaults checks for configuration errors and ensures all
// values are set on the Config object.
func (c *Config) ValidateAndInjectDefaults() error {

	if c.Trigger == nil {
		t := plugins.DefaultTriggerMode
		c.Trigger = &t
	}

	switch *c.Trigger {
	case plugins.TriggerPeriodic, plugins.TriggerManual:
		break
	default:
		return fmt.Errorf("invalid trigger mode %q (want %q or %q)", *c.Trigger, plugins.TriggerPeriodic, plugins.TriggerManual)
	}

	min := defaultMinDelaySeconds
	max := defaultMaxDelaySeconds

	// reject bad min/max values
	if c.Polling.MaxDelaySeconds != nil && c.Polling.MinDelaySeconds != nil {
		if *c.Polling.MaxDelaySeconds < *c.Polling.MinDelaySeconds {
			return fmt.Errorf("max polling delay must be >= min polling delay")
		}
		min = *c.Polling.MinDelaySeconds
		max = *c.Polling.MaxDelaySeconds
	} else if c.Polling.MaxDelaySeconds == nil && c.Polling.MinDelaySeconds != nil {
		return fmt.Errorf("polling configuration missing 'max_delay_seconds'")
	} else if c.Polling.MinDelaySeconds == nil && c.Polling.MaxDelaySeconds != nil {
		return fmt.Errorf("polling configuration missing 'min_delay_seconds'")
	}

	// scale to seconds
	minSeconds := int64(time.Duration(min) * time.Second)
	c.Polling.MinDelaySeconds = &minSeconds

	maxSeconds := int64(time.Duration(max) * time.Second)
	c.Polling.MaxDelaySeconds = &maxSeconds

	if c.Polling.LongPollingTimeoutSeconds != nil {
		if *c.Polling.LongPollingTimeoutSeconds < 1 {
			return fmt.Errorf("'long_polling_timeout_seconds' must be at least 1")
		}
	}

	return nil
}
