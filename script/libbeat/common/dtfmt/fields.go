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
	"time"
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
	ftMillisOfDay
	ftMillisOfSecond
	ftTimeZoneOffset
)

func getIntField(ft fieldType, ctx *ctx, t time.Time) (int, error) {
	switch ft {
	case ftYear:
		return ctx.year, nil

	case ftDayOfYear:
		return ctx.yearday, nil

	case ftMonthOfYear:
		return int(ctx.month), nil

	case ftDayOfMonth:
		return ctx.day, nil

	case ftWeekyear:
		return ctx.isoYear, nil

	case ftWeekOfWeekyear:
		return ctx.isoWeek, nil

	case ftDayOfWeek:
		return int(ctx.weekday), nil

	case ftHalfdayOfDay:
		if ctx.hour < 12 {
			return 0, nil // AM
		}
		return 1, nil // PM

	case ftHourOfHalfday:
		if ctx.hour < 12 {
			return ctx.hour, nil
		}
		return ctx.hour - 12, nil

	case ftClockhourOfHalfday:
		if ctx.hour < 12 {
			return ctx.hour + 1, nil
		}
		return ctx.hour - 12 + 1, nil

	case ftClockhourOfDay:
		return ctx.hour + 1, nil

	case ftHourOfDay:
		return ctx.hour, nil

	case ftMinuteOfDay:
		return ctx.hour*60 + ctx.min, nil

	case ftMinuteOfHour:
		return ctx.min, nil

	case ftSecondOfDay:
		return (ctx.hour*60+ctx.min)*60 + ctx.sec, nil

	case ftSecondOfMinute:
		return ctx.sec, nil

	case ftMillisOfDay:
		return ((ctx.hour*60+ctx.min)*60+ctx.sec)*1000 + ctx.millis, nil

	case ftMillisOfSecond:
		return ctx.millis, nil
	}

	return 0, nil
}

func getTextField(ft fieldType, ctx *ctx, t time.Time) (string, error) {
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

func getTextFieldShort(ft fieldType, ctx *ctx, t time.Time) (string, error) {
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
