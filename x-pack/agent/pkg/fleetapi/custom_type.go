// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"encoding/json"
	"time"
)

const timeFormat = time.RFC3339Nano

// Time is a custom time that impose the serialization format.
type Time time.Time

// MarshalJSON make sure that all the times are serialized with the RFC3339 format.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(timeFormat))
}
