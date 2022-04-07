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

package schedule

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/v8/heartbeat/scheduler"
	"github.com/elastic/beats/v8/heartbeat/scheduler/schedule/cron"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		schedStr string
		want     *Schedule
		wantErr  bool
	}{
		{
			"every second",
			"@every 1s",
			&Schedule{intervalScheduler{time.Duration(1 * time.Second)}},
			false,
		},
		{
			"every year",
			"@every 1m",
			&Schedule{intervalScheduler{time.Duration(1 * time.Minute)}},
			false,
		},
		{
			"cron every minute",
			"* * * * *",
			&Schedule{cron.MustParse("* * * * *")},
			false,
		},
		{
			"cron complex",
			"*/15 4 * 2 *",
			&Schedule{cron.MustParse("*/15 4 * 2 *")},
			false,
		},
		{
			"invalid syntax",
			"foobar",
			nil,
			true,
		},
		{
			"empty str",
			"",
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.schedStr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_intervalScheduler_Next(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		s    intervalScheduler
		want time.Time
	}{
		{
			"one second",
			intervalScheduler{time.Duration(time.Second)},
			now.Add(time.Second),
		},
		{
			"one minute",
			intervalScheduler{time.Duration(time.Minute)},
			now.Add(time.Minute),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Next(now); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("intervalScheduler.Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchedule_Unpack(t *testing.T) {
	tests := []struct {
		name     string
		s        *Schedule
		timeStr  string
		expected scheduler.Schedule
		wantErr  bool
	}{
		{
			"one minute -> one second",
			&Schedule{intervalScheduler{time.Minute}},
			"@every 1s",
			intervalScheduler{time.Second},
			false,
		},
		{
			"every 15 cron -> every second interval",
			&Schedule{cron.MustParse("*/15 * * * *")},
			"@every 1s",
			intervalScheduler{time.Second},
			false,
		},
		{
			"every second interval -> every 15 cron",
			&Schedule{intervalScheduler{time.Second}},
			"*/15 * * * *",
			cron.MustParse("*/15 * * * *"),
			false,
		},
		{
			"bad format",
			&Schedule{intervalScheduler{time.Minute}},
			"foobar",
			intervalScheduler{time.Minute},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Unpack(tt.timeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Schedule.Unpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.s.Schedule, tt.expected) {
				t.Errorf("schedule.Unpack(%s) changed internal schedule to %v, wanted %v", tt.timeStr, tt.s.Schedule, tt.expected)
			}
		})
	}
}
