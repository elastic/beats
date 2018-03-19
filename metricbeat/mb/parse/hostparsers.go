package parse

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
)

// PassThruHostParser is a HostParser that sets the HostData URI, SanitizedURI,
// and Host to the configured 'host' value. This should only be used by
// MetricSets that do not require host parsing (e.g. host is only addr:port).
// Do not use this if the host value can contain credentials.
func PassThruHostParser(module mb.Module, host string) (mb.HostData, error) {
	return mb.HostData{URI: host, SanitizedURI: host, Host: host}, nil
}

// EmptyHostParser simply returns a zero value HostData. It asserts that host
// value is empty and returns an error if not.
func EmptyHostParser(module mb.Module, host string) (mb.HostData, error) {
	if host != "" {
		return mb.HostData{}, errors.Errorf("hosts must be empty for %v", module.Name())
	}

	return mb.HostData{}, nil
}
