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
	"fmt"
	"hash"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/dtfmt"
	conf "github.com/elastic/elastic-agent-libs/config"
)

const (
	millisecPrecision TimestampPrecision = iota + 1
	microsecPrecision
	nanosecPrecision

	DefaultTimestampPrecision = millisecPrecision

	tsLayoutMillis = "2006-01-02T15:04:05.000Z"
	tsLayoutMicros = "2006-01-02T15:04:05.000000Z"
	tsLayoutNanos  = "2006-01-02T15:04:05.000000000Z"

	millisecPrecisionFmt = "yyyy-MM-dd'T'HH:mm:ss.fff'Z'"
	microsecPrecisionFmt = "yyyy-MM-dd'T'HH:mm:ss.ffffff'Z'"
	nanosecPrecisionFmt  = "yyyy-MM-dd'T'HH:mm:ss.fffffffff'Z'"

	localMillisecPrecisionFmt = "yyyy-MM-dd'T'HH:mm:ss.fffz"
	localMicrosecPrecisionFmt = "yyyy-MM-dd'T'HH:mm:ss.ffffffz"
	localNanosecPrecisionFmt  = "yyyy-MM-dd'T'HH:mm:ss.fffffffffz"
)

// timestampFmt stores the format strings for both UTC and local
// form of a specific precision.
type timestampFmt struct {
	utc   string
	local string
}

var (
	defaultParseFormats = []string{
		tsLayoutMillis,
		tsLayoutMicros,
		tsLayoutNanos,
	}

	precisions = map[TimestampPrecision]timestampFmt{
		millisecPrecision: timestampFmt{utc: millisecPrecisionFmt, local: localMillisecPrecisionFmt},
		microsecPrecision: timestampFmt{utc: microsecPrecisionFmt, local: localMicrosecPrecisionFmt},
		nanosecPrecision:  timestampFmt{utc: nanosecPrecisionFmt, local: localNanosecPrecisionFmt},
	}

	// tsFmt is the selected timestamp format
	tsFmt = precisions[DefaultTimestampPrecision]
	// timeFormatter is a datettime formatter with a selected timestamp precision in UTC.
	timeFormatter = dtfmt.MustNewFormatter(tsFmt.utc)
)

// Time is an abstraction for the time.Time type
type Time time.Time

type TimestampPrecision uint8

type TimestampConfig struct {
	Precision TimestampPrecision `config:"precision"`
}

func defaultTimestampConfig() TimestampConfig {
	return TimestampConfig{Precision: DefaultTimestampPrecision}
}

func (p *TimestampPrecision) Unpack(v string) error {
	switch v {
	case "millisecond", "":
		*p = millisecPrecision
	case "microsecond":
		*p = microsecPrecision
	case "nanosecond":
		*p = nanosecPrecision
	default:
		return fmt.Errorf("invalid timestamp precision %s, available options: millisecond, microsecond, nanosecond", v)
	}
	return nil
}

// SetTimestampPrecision sets the precision of timestamps in the Beat.
// It is only supposed to be called during init because it changes
// the format of the timestamps globally. Calling it with a nil value
// resets it to the default format.
func SetTimestampPrecision(c *conf.C) error {
	p := defaultTimestampConfig()

	if c != nil {
		if err := c.Unpack(&p); err != nil {
			return fmt.Errorf("failed to set timestamp precision: %w", err)
		}
	}

	tsFmt = precisions[p.Precision]
	timeFormatter = dtfmt.MustNewFormatter(precisions[p.Precision].utc)

	return nil
}

// TimestampFormat returns the datettime format string
// with the configured timestamp precision. It can return
// either the UTC format or the local one.
func TimestampFormat(local bool) string {
	if local {
		return tsFmt.local
	}
	return tsFmt.utc
}

// MarshalJSON implements json.Marshaler interface.
// The time is a quoted string in the JsTsLayout format.
func (t Time) MarshalJSON() ([]byte, error) {
	str, _ := timeFormatter.Format(time.Time(t).UTC())
	return json.Marshal(str)
}

// UnmarshalJSON implements js.Unmarshaler interface.
// The time is expected to be a quoted string in TsLayout
// format.
func (t *Time) UnmarshalJSON(data []byte) (err error) {
	if data[0] != []byte(`"`)[0] || data[len(data)-1] != []byte(`"`)[0] {
		return errors.New("not quoted")
	}
	*t, err = ParseTime(string(data[1 : len(data)-1]))
	return err
}

func (t Time) Hash32(h hash.Hash32) error {
	err := binary.Write(h, binary.LittleEndian, time.Time(t).UnixNano())
	return err
}

// ParseTime parses a time in the MillisTsLayout, then micros and finally nanos.
func ParseTime(timespec string) (Time, error) {
	var err error
	var t time.Time

	for _, layout := range defaultParseFormats {
		t, err = time.Parse(layout, timespec)
		if err == nil {
			break
		}
	}

	return Time(t), err
}

func (t Time) String() string {
	str, _ := timeFormatter.Format(time.Time(t).UTC())
	return str
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
