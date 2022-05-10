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

package dtfmt

import (
	"errors"
)

type fieldType uint8

const (
	ftYear fieldType = iota
	ftDayOfYear
	ftMonthOfYear
	ftDayOfMonth
	ftWeekyear
	ftWeekOfWeekyear
	ftDayOfWeek
	ftHalfdayOfDay
	ftHourOfHalfday
	ftClockhourOfHalfday
	ftClockhourOfDay
	ftHourOfDay
	ftMinuteOfDay
	ftMinuteOfHour
	ftSecondOfDay
	ftSecondOfMinute
	ftTimeZoneOffset
	ftNanoOfSecond
)

func getIntField(ft fieldType, ctx *ctx) int {
	switch ft {
	case ftYear:
		return ctx.year

	case ftDayOfYear:
		return ctx.yearday

	case ftMonthOfYear:
		return int(ctx.month)

	case ftDayOfMonth:
		return ctx.day

	case ftWeekyear:
		return ctx.isoYear

	case ftWeekOfWeekyear:
		return ctx.isoWeek

	case ftDayOfWeek:
		return int(ctx.weekday)

	case ftHalfdayOfDay:
		if ctx.hour < 12 {
			return 0 // AM
		}
		return 1 // PM

	case ftHourOfHalfday:
		if ctx.hour < 12 {
			return ctx.hour
		}
		return ctx.hour - 12

	case ftClockhourOfHalfday:
		if ctx.hour < 12 {
			return ctx.hour + 1
		}
		return ctx.hour - 12 + 1

	case ftClockhourOfDay:
		return ctx.hour + 1

	case ftHourOfDay:
		return ctx.hour

	case ftMinuteOfDay:
		return ctx.hour*60 + ctx.min

	case ftMinuteOfHour:
		return ctx.min

	case ftSecondOfDay:
		return (ctx.hour*60+ctx.min)*60 + ctx.sec

	case ftSecondOfMinute:
		return ctx.sec

	case ftNanoOfSecond:
		return ctx.nano
	}

	return 0

}

func getTextField(ft fieldType, ctx *ctx) (string, error) {
	switch ft {
	case ftHalfdayOfDay:
		if ctx.hour < 12 {
			return "AM", nil
		}
		return "PM", nil
	case ftDayOfWeek:
		return ctx.weekday.String(), nil
	case ftMonthOfYear:
		return ctx.month.String(), nil
	case ftTimeZoneOffset:
		return tzOffsetString(ctx)
	default:
		return "", errors.New("no text field")
	}
}

func tzOffsetString(ctx *ctx) (string, error) {
	buf := make([]byte, 6)

	tzOffsetMinutes := ctx.tzOffset / 60 // convert to minutes
	if tzOffsetMinutes >= 0 {
		buf[0] = '+'
	} else {
		buf[0] = '-'
		tzOffsetMinutes = -tzOffsetMinutes
	}

	tzOffsetHours := tzOffsetMinutes / 60
	tzOffsetMinutes = tzOffsetMinutes % 60
	buf[1] = byte(tzOffsetHours/10) + '0'
	buf[2] = byte(tzOffsetHours%10) + '0'
	buf[3] = ':'
	buf[4] = byte(tzOffsetMinutes/10) + '0'
	buf[5] = byte(tzOffsetMinutes%10) + '0'
	return string(buf), nil
}

func getTextFieldShort(ft fieldType, ctx *ctx) (string, error) {
	switch ft {
	case ftHalfdayOfDay:
		if ctx.hour < 12 {
			return "AM", nil
		}
		return "PM", nil
	case ftDayOfWeek:
		return ctx.weekday.String()[:3], nil
	case ftMonthOfYear:
		return ctx.month.String()[:3], nil
	default:
		return "", errors.New("no text field")
	}
}
