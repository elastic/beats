// Package config provides the winlogbeat specific configuration options.
package config

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/joeshaw/multierror"
)

const (
	// DefaultRegistryFile specifies the default filename of the registry file.
	DefaultRegistryFile = ".winlogbeat.yml"
)

// Validator is the interface for configuration data that can be validating.
//
// Validate reads the configuration and validates all fields. An error
// describing all problems is returned (versus returning an error only for the
// first problem encountered).
type Validator interface {
	Validate() error
}

// Settings is the root of the Winlogbeat configuration data hierarchy.
type Settings struct {
	Winlogbeat WinlogbeatConfig       `config:"winlogbeat"`
	All        map[string]interface{} `config:",inline"`
}

// Validate validates the Settings data and returns an error describing
// all problems or nil if there are none.
func (s Settings) Validate() error {
	validKeys := []string{"winlogbeat", "output", "shipper", "logging"}
	sort.Strings(validKeys)

	// Check for invalid top-level keys.
	var errs multierror.Errors
	for k := range s.All {
		k = strings.ToLower(k)
		i := sort.SearchStrings(validKeys, k)
		if i >= len(validKeys) || validKeys[i] != k {
			errs = append(errs, fmt.Errorf("Invalid top-level key '%s' "+
				"found. Valid keys are %s", k, strings.Join(validKeys, ", ")))
		}
	}

	err := s.Winlogbeat.Validate()
	if err != nil {
		errs = append(errs, err)
	}

	return errs.Err()
}

// WinlogbeatConfig contains all of Winlogbeat configuration data.
type WinlogbeatConfig struct {
	IgnoreOlder  string           `config:"ignore_older"`
	EventLogs    []EventLogConfig `config:"event_logs"`
	Metrics      MetricsConfig    `config:"metrics"`
	RegistryFile string           `config:"registry_file"`
}

// Validate validates the WinlogbeatConfig data and returns an error describing
// all problems or nil if there are none.
func (ebc WinlogbeatConfig) Validate() error {
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

// MetricsConfig holds the configuration data for the HTTP metric service.
type MetricsConfig struct {
	BindAddress string // Bind address for the metric service. Format is host:port.
}

// Validate validates the MetricsConfig data and returns an error describing any
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

// EventLogConfig holds the configuration data that specifies which event logs
// to monitor.
type EventLogConfig struct {
	common.EventMetadata `config:",inline"`
	Name                 string
	IgnoreOlder          string `config:"ignore_older"`
	API                  string
}

// Validate validates the EventLogConfig data and returns an error describing
// any problems or nil.
func (elc EventLogConfig) Validate() error {
	var errs multierror.Errors
	if elc.Name == "" {
		err := fmt.Errorf("event log is missing a 'name'")
		errs = append(errs, err)
	}

	if _, err := IgnoreOlderDuration(elc.IgnoreOlder); err != nil {
		err := fmt.Errorf("Invalid ignore_older value ('%s') for event_log "+
			"'%s' (%v)", elc.IgnoreOlder, elc.Name, err)
		errs = append(errs, err)
	}

	switch strings.ToLower(elc.API) {
	case "", "eventlogging", "wineventlog":
		break
	default:
		err := fmt.Errorf("Invalid api value ('%s') for event_log '%s'",
			elc.API, elc.Name)
		errs = append(errs, err)
	}

	return errs.Err()
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
