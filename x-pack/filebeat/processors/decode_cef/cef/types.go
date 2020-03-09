// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

import (
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"

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

// ToType converts the given value string value to the specified data type.
func ToType(value string, typ DataType) (interface{}, error) {
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
		return toTimestamp(value)
	default:
		return nil, errors.Errorf("invalid data type: %v", typ)
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
	hw, err := net.ParseMAC(v)
	if err != nil {
		return "", err
	}
	return hw.String(), nil
}

var timeLayouts = []string{
	// MMM dd HH:mm:ss.SSS zzz
	"Jan _2 15:04:05.000 MST",
	// MMM dd HH:mm:sss.SSS
	"Jan _2 15:04:05.000",
	// MMM dd HH:mm:ss zzz
	"Jan _2 15:04:05 MST",
	// MMM dd HH:mm:ss
	"Jan _2 15:04:05",
	// MMM dd yyyy HH:mm:ss.SSS zzz
	"Jan _2 2006 15:04:05.000 MST",
	// MMM dd yyyy HH:mm:ss.SSS
	"Jan _2 2006 15:04:05.000",
	// MMM dd yyyy HH:mm:ss zzz
	"Jan _2 2006 15:04:05 MST",
	// MMM dd yyyy HH:mm:ss
	"Jan _2 2006 15:04:05",
}

func toTimestamp(v string) (common.Time, error) {
	if unixMs, err := toLong(v); err == nil {
		return common.Time(time.Unix(0, unixMs*int64(time.Millisecond))), nil
	}

	for _, layout := range timeLayouts {
		ts, err := time.ParseInLocation(layout, v, time.UTC)
		if err == nil {
			// Use current year if no year is zero.
			if ts.Year() == 0 {
				currentYear := time.Now().In(ts.Location()).Year()
				ts = ts.AddDate(currentYear, 0, 0)
			}

			return common.Time(ts), nil
		}
	}

	return common.Time(time.Time{}), errors.New("value is not a valid timestamp")
}
