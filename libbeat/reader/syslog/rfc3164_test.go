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

package syslog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var parseRFC3164Cases = map[string]struct {
	In      string
	Want    message
	WantErr error
}{
	"ok": {
		In: "<13>Oct 11 22:14:15 test-host this is the message",
		Want: message{
			timestamp: mustParseTimeLoc(time.Stamp, "Oct 11 22:14:15", time.Local),
			priority:  13,
			facility:  1,
			severity:  5,
			hostname:  "test-host",
			msg:       "this is the message",
		},
	},
	"ok-rfc3339": {
		In: "<13>2003-08-24T05:14:15.000003-07:00 test-host this is the message",
		Want: message{
			timestamp: mustParseTime(time.RFC3339Nano, "2003-08-24T05:14:15.000003-07:00"),
			priority:  13,
			facility:  1,
			severity:  5,
			hostname:  "test-host",
			msg:       "this is the message",
		},
	},
	"ok-process": {
		In: "<13>Oct 11 22:14:15 test-host su: this is the message",
		Want: message{
			timestamp: mustParseTimeLoc(time.Stamp, "Oct 11 22:14:15", time.Local),
			priority:  13,
			facility:  1,
			severity:  5,
			hostname:  "test-host",
			process:   "su",
			msg:       "this is the message",
		},
	},
	"ok-process-pid": {
		In: "<13>Oct 11 22:14:15 test-host su[1024]: this is the message",
		Want: message{
			timestamp: mustParseTimeLoc(time.Stamp, "Oct 11 22:14:15", time.Local),
			priority:  13,
			facility:  1,
			severity:  5,
			hostname:  "test-host",
			process:   "su",
			pid:       "1024",
			msg:       "this is the message",
		},
	},
	"err-pri-not-a-number": {
		In:      "<abc>Oct 11 22:14:15 test-host this is the message",
		WantErr: ErrPriority,
	},
	"err-pri-out-of-range": {
		In:      "<192>Oct 11 22:14:15 test-host this is the message",
		WantErr: ErrPriority,
	},
	"err-pri-negative": {
		In:      "<-1>Oct 11 22:14:15 test-host this is the message",
		WantErr: ErrPriority,
	},
	"err-pri-missing-brackets": {
		In:      "13 Oct 11 22:14:15 test-host this is the message",
		WantErr: ErrPriorityPart,
	},
	"err-ts-invalid-missing": {
		In:      "<13> test-host this is the message",
		WantErr: ErrTimestamp,
	},
	"err-ts-invalid-bsd": {
		In:      "<13>Foo 11 22:14:15 test-host this is the message",
		WantErr: ErrTimestamp,
	},
	"err-ts-invalid-rfc3339": {
		In:      "<13>2003-08-24 05:14:15-07:00 test-host this is the message",
		WantErr: ErrTimestamp,
	},
	"err-hostname-too-long": {
		In:      "<13>Oct 11 22:14:15 abcdefghijklmnopqrstuvwxyz12345.abcdefghijklmnopqrstuvwxyz12345.abcdefghijklmnopqrstuvwxyz12345.abcdefghijklmnopqrstuvwxyz12345.abcdefghijklmnopqrstuvwxyz12345.abcdefghijklmnopqrstuvwxyz12345.abcdefghijklmnopqrstuvwxyz12345.abcdefghijklmnopqrstuvwxyz1234567 this is the message",
		WantErr: ErrHostname,
	},
}

func TestParseRFC3164(t *testing.T) {
	for name, tc := range parseRFC3164Cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := parseRFC3164(tc.In, time.Local)

			if tc.WantErr != nil {
				assert.Equal(t, tc.WantErr, gotErr)
			} else {
				assert.Nil(t, gotErr)
				assert.Equal(t, tc.Want, got)
			}
		})
	}
}

func BenchmarkParseRFC3164(b *testing.B) {
	for name, bc := range parseRFC3164Cases {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = parseRFC3164(bc.In, time.Local)
			}
		})
	}
}
