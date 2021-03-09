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

	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule/cron"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		schedStr string
		want     Schedule
		wantErr  bool
	}{
		{
			"every second",
			"@every 1s",
			intervalScheduler{time.Duration(1 * time.Second)},
			false,
		},
		{
			"every year",
			"@every 1m",
			intervalScheduler{time.Duration(1 * time.Minute)},
			false,
		},
		{
			"cron every minute",
			"* * * * *",
			cron.MustParse("* * * * *"),
			false,
		},
		{
			"cron complex",
			"*/15 4 * 2 *",
			cron.MustParse("*/15 4 * 2 *"),
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
			got, err := Parse(tt.schedStr, "myId")
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

func TestSchedule_Timespan(t *testing.T) {
	tests := []struct {
		name   string
		sched  Schedule
		t      time.Time
		wantTs TimespanBounds
	}{
		{
			"One second interval",
			intervalScheduler{time.Second},
			time.Unix(1000, 0),
			TimespanBounds{Gte: time.Unix(1000, 0), Lt: time.Unix(1001, 0)},
		},
		{
			"One minute interval",
			intervalScheduler{time.Minute},
			time.Unix(60, 0),
			TimespanBounds{Gte: time.Unix(60, 0), Lt: time.Unix(120, 0)},
		},
		{
			"One minute interval, odd time",
			intervalScheduler{time.Minute},
			time.Unix(83, 0),
			TimespanBounds{Gte: time.Unix(60, 0), Lt: time.Unix(120, 0)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotTs := Timespan(tt.t, tt.sched); !reflect.DeepEqual(gotTs, tt.wantTs) {
				t.Errorf("Timespan.Gte() = %v, want %v", gotTs.Gte, tt.wantTs.Gte)
				t.Errorf("Timespan.Lt() = %v, want %v", gotTs.Lt, tt.wantTs.Lt)
			}
		})
	}
}
