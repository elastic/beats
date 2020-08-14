// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package log

import (
	"fmt"
	"time"
)

// Format used for logging [DefaultFormat, JSONFormat]
type Format bool

const (
	// DefaultFormat is a log format, resulting in: "2006-01-02T15:04:05: type: 'STATE': event type: 'STARTING' message: Application 'filebeat' is starting."
	DefaultFormat Format = true
	// JSONFormat is a log format, resulting in: {"timestamp": "2006-01-02T15:04:05", "type": "STATE", "event": {"type": "STARTING", "message": "Application 'filebeat' is starting."}
	JSONFormat Format = false
)

const (
	// e.g "2006-01-02T15:04:05: type: 'STATE': event type: 'STARTING' message: Application 'filebeat' is starting."
	defaultLogFormat = "%s: type: '%s': sub_type: '%s' message: %s"
	timeFormat       = time.RFC3339
)

var formatMap = map[string]Format{
	"default": DefaultFormat,
	"json":    JSONFormat,
}

var reverseMap = map[bool]string{
	true:  "default",
	false: "json",
}

// Unpack enables using of string values in config
func (m *Format) Unpack(v string) error {
	mgt, ok := formatMap[v]
	if !ok {
		return fmt.Errorf(
			"unknown format, received '%s' and valid values are default or json",
			v,
		)
	}
	*m = mgt
	return nil
}

// MarshalYAML marshal into a string.
func (m Format) MarshalYAML() (interface{}, error) {
	s, ok := reverseMap[bool(m)]
	if !ok {
		return nil, fmt.Errorf("cannot marshal value of %+v", m)
	}

	return s, nil
}
