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

package common

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash"
	"time"
)

// TsLayout is the layout to be used in the timestamp marshaling/unmarshaling everywhere.
// The timezone must always be UTC.
const TsLayout = "2006-01-02T15:04:05.000Z"

// Time is an abstraction for the time.Time type
type Time time.Time

// MarshalJSON implements json.Marshaler interface.
// The time is a quoted string in the JsTsLayout format.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).UTC().Format(TsLayout))
}

// UnmarshalJSON implements js.Unmarshaler interface.
// The time is expected to be a quoted string in TsLayout
// format.
func (t *Time) UnmarshalJSON(data []byte) (err error) {
	if data[0] != []byte(`"`)[0] || data[len(data)-1] != []byte(`"`)[0] {
		return errors.New("Not quoted")
	}
	*t, err = ParseTime(string(data[1 : len(data)-1]))
	return
}

func (t Time) Hash32(h hash.Hash32) error {
	err := binary.Write(h, binary.LittleEndian, time.Time(t).UnixNano())
	return err
}

// ParseTime parses a time in the TsLayout format.
func ParseTime(timespec string) (Time, error) {
	t, err := time.Parse(TsLayout, timespec)
	return Time(t), err
}

func (t Time) String() string {
	return time.Time(t).Format(TsLayout)
}

// MustParseTime is a convenience equivalent of the ParseTime function
// that panics in case of errors.
func MustParseTime(timespec string) Time {
	ts, err := ParseTime(timespec)
	if err != nil {
		panic(err)
	}

	return ts
}
