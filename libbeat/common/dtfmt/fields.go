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
	default:
		return "", errors.New("no text field")
	}
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
