// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cfgtype

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	// Embed the timezone database so this code works across platforms.
	_ "time/tzdata"
)

var fixedOffsetFormats = []string{"-07", "-0700", "-07:00"}

// Timezone maps time instants to the zone in use at that time. Typically, the
// Timezone represents the collection of time offsets in use in a geographical
// area. For many Locations the time offset varies depending on whether daylight
// savings time is in use at the time instant.
type Timezone time.Location

// NewTimezone returns a new timezone.
func NewTimezone(tz string) (*Timezone, error) {
	loc, err := loadLocation(tz)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse timezone %q", tz)
	}
	return (*Timezone)(loc), nil
}

// MustNewTimezone returns a new timezone. If tz is invalid it panics.
func MustNewTimezone(tz string) *Timezone {
	timestamp, err := NewTimezone(tz)
	if err != nil {
		panic(err)
	}
	return timestamp
}

// Location returns a *time.Location. If timezone is nil it returns *time.UTC.
func (tz *Timezone) Location() *time.Location {
	if tz == nil {
		return time.UTC
	}
	return (*time.Location)(tz)
}

// MarshalJSON implements json.Marshaler interface.
func (tz *Timezone) MarshalJSON() ([]byte, error) {
	if tz == nil {
		return []byte("null"), nil
	}
	return json.Marshal(tz.Location().String())
}

// Unpack converts a time zone name or offset to Timezone. If using a fixed
// offset then the format must be [+-]HHMM (e.g +0800 or -0530).
func (tz *Timezone) Unpack(v string) error {
	timezone, err := NewTimezone(v)
	if err != nil {
		return err
	}
	*tz = *timezone
	return nil
}

func loadLocation(timezone string) (*time.Location, error) {
	for _, format := range fixedOffsetFormats {
		t, err := time.Parse(format, timezone)
		if err == nil {
			name, offset := t.Zone()
			return time.FixedZone(name, offset), nil
		}
	}

	// Handle IANA time zones.
	return time.LoadLocation(timezone)
}
