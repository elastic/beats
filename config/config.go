// Package config provides the eventbeat specific configuration options.
package config

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/joeshaw/multierror"
)

type Validator interface {
	Validate() error
}

type ConfigSettings struct {
	Eventbeat EventbeatConfig
}

type EventbeatConfig struct {
	IgnoreOlder string           `yaml:"ignore_older"`
	EventLogs   []EventLogConfig `yaml:"event_logs"`
	Metrics     MetricsConfig
}

// Validates the EventbeatConfig data and returns an error describing all
// problems or nil if there are none.
func (ebc EventbeatConfig) Validate() error {
	var errs multierror.Errors
	if _, err := IgnoreOlderDuration(ebc.IgnoreOlder); err != nil {
		errs = append(errs, fmt.Errorf("Invalid top level ignore_older value "+
			"'%s' (%v)", ebc.IgnoreOlder, err))
	}

	if len(ebc.EventLogs) == 0 {
		errs = append(errs, fmt.Errorf("At least one event log must be "+
			"configured as part of event_logs"))
	}

	for _, eventLog := range ebc.EventLogs {
		if err := eventLog.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := ebc.Metrics.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errs.Err()
}

type MetricsConfig struct {
	BindAddress string // Bind address for the metric service. Format is host:port.
}

// Validates the MetricsConfig data and returns an error describing any
// problems or nil.
func (mc MetricsConfig) Validate() error {
	if mc.BindAddress == "" {
		return nil
	}

	host, portStr, err := net.SplitHostPort(mc.BindAddress)
	if err != nil {
		return fmt.Errorf("bind_address must be formatted as host:port but "+
			"was '%s' (%v)", mc.BindAddress, err)
	}

	if len(host) == 0 {
		return fmt.Errorf("bind_address value ('%s') is missing a host",
			mc.BindAddress)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("bind_address port value ('%s') must be a number "+
			"(%v)", portStr, err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("bind_address port must be within [1-65535] but "+
			"was '%d'", port)
	}

	return nil
}

type EventLogConfig struct {
	Name        string
	IgnoreOlder string `yaml:"ignore_older"`
}

// Validates the EventLogConfig data and returns an error describing any
// problems or nil.
func (elc EventLogConfig) Validate() error {
	if elc.Name == "" {
		return fmt.Errorf("event log is missing a 'name'")
	}

	if _, err := IgnoreOlderDuration(elc.IgnoreOlder); err != nil {
		return fmt.Errorf("Invalid ignore_older value ('%s') for event_log "+
			"'%s' (%v)", elc.IgnoreOlder, elc.Name, err)
	}

	return nil
}

// IgnoreOlderDuration returns the parsed value of the IgnoreOlder string. If
// IgnoreOlder is not set then (0, nil) is returned. If IgnoreOlder is not
// parsable as a duration then an error is returned. See time.ParseDuration.
func IgnoreOlderDuration(ignoreOlder string) (time.Duration, error) {
	if ignoreOlder == "" {
		return time.Duration(0), nil
	}

	duration, err := time.ParseDuration(ignoreOlder)
	return duration, err
}
