// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
)

// DataType specifies one of CEF data types.
type DataType uint8

// List of DataTypes.
const (
	Unset DataType = iota
	IntegerType
	LongType
	FloatType
	DoubleType
	StringType
	BooleanType
	IPType
	MACAddressType
	TimestampType
)

// toType converts the given value string value to the specified data type.
func toType(value string, typ DataType, settings *Settings) (interface{}, error) {
	switch typ {
	case StringType:
		return value, nil
	case LongType:
		return toLong(value)
	case IntegerType:
		return toInteger(value)
	case FloatType:
		return toFloat(value)
	case DoubleType:
		return toDouble(value)
	case BooleanType:
		return toBoolean(value)
	case IPType:
		return toIP(value)
	case MACAddressType:
		return toMACAddress(value)
	case TimestampType:
		return toTimestamp(value, settings)
	default:
		return nil, fmt.Errorf("invalid data type: %v", typ)
	}
}

func toLong(v string) (int64, error) {
	return strconv.ParseInt(v, 0, 64)
}

func toInteger(v string) (int32, error) {
	i, err := strconv.ParseInt(v, 0, 32)
	return int32(i), err
}

func toFloat(v string) (float32, error) {
	f, err := strconv.ParseFloat(v, 32)
	return float32(f), err
}

func toDouble(v string) (float64, error) {
	f, err := strconv.ParseFloat(v, 64)
	return f, err
}

func toBoolean(v string) (bool, error) {
	return strconv.ParseBool(v)
}

func toIP(v string) (string, error) {
	// This is validating that the value is an IP.
	if net.ParseIP(v) != nil {
		return v, nil
	}
	return "", errors.New("value is not a valid IP address")
}

// toMACAddress accepts a MAC addresses as hex characters separated by colon,
// dot, or dash. It returns lowercase hex characters separated by colons.
func toMACAddress(v string) (string, error) {
	// CEF specifies that MAC addresses are colon separated, but this will be a
	// little more liberal.
	hw, err := net.ParseMAC(insertMACSeparators(v))
	if err != nil {
		return "", err
	}
	return hw.String(), nil
}

// insertMACSeparators adds colon separators to EUI-48 and EUI-64 addresses that
// have no separators.
func insertMACSeparators(v string) string {
	const (
		eui48HexLength                 = 48 / 4
		eui64HexLength                 = 64 / 4
		eui64HexWithSeparatorMaxLength = eui64HexLength + eui64HexLength/2 - 1
	)

	// Check that the length is correct for a MAC address without separators.
	// And check that there isn't already a separator in the string.
	if len(v) != eui48HexLength && len(v) != eui64HexLength || v[2] == ':' || v[2] == '-' || v[4] == '.' {
		return v
	}

	var sb strings.Builder
	sb.Grow(eui64HexWithSeparatorMaxLength)

	for i := 0; i < len(v); i++ {
		sb.WriteByte(v[i])
		if i < len(v)-1 && i%2 != 0 {
			sb.WriteByte(':')
		}
	}
	return sb.String()
}

var timeLayouts = []string{
	// MMM dd HH:mm:ss.SSS zzz
	"Jan _2 15:04:05.000 MST",
	"Jan _2 15:04:05.000 Z0700",
	"Jan _2 15:04:05.000 Z07:00",
	"Jan _2 15:04:05.000 GMT-07:00",

	// MMM dd HH:mm:sss.SSS
	"Jan _2 15:04:05.000",

	// MMM dd HH:mm:ss zzz
	"Jan _2 15:04:05 MST",
	"Jan _2 15:04:05 Z0700",
	"Jan _2 15:04:05 Z07:00",
	"Jan _2 15:04:05 GMT-07:00",

	// MMM dd HH:mm:ss
	"Jan _2 15:04:05",

	// MMM dd yyyy HH:mm:ss.SSS zzz
	"Jan _2 2006 15:04:05.000 MST",
	"Jan _2 2006 15:04:05.000 Z0700",
	"Jan _2 2006 15:04:05.000 Z07:00",
	"Jan _2 2006 15:04:05.000 GMT-07:00",

	// MMM dd yyyy HH:mm:ss.SSS
	"Jan _2 2006 15:04:05.000",

	// MMM dd yyyy HH:mm:ss zzz
	"Jan _2 2006 15:04:05 MST",
	"Jan _2 2006 15:04:05 Z0700",
	"Jan _2 2006 15:04:05 Z07:00",
	"Jan _2 2006 15:04:05 GMT-07:00",

	// MMM dd yyyy HH:mm:ss
	"Jan _2 2006 15:04:05",
}

func toTimestamp(v string, settings *Settings) (common.Time, error) {
	if unixMs, err := toLong(v); err == nil {
		return common.Time(time.Unix(0, unixMs*int64(time.Millisecond))), nil
	}

	// Use this timezone when one is not included in the time string.
	defaultLocation := time.UTC
	if settings != nil && settings.timezone != nil {
		defaultLocation = settings.timezone
	}

	for _, layout := range timeLayouts {
		ts, err := time.ParseInLocation(layout, v, defaultLocation)
		if err == nil {
			// Use current year if year is zero.
			if ts.Year() == 0 {
				currentYear := time.Now().In(ts.Location()).Year()
				ts = ts.AddDate(currentYear, 0, 0)
			}

			return common.Time(ts), nil
		}
	}

	return common.Time(time.Time{}), errors.New("value is not a valid timestamp")
}
