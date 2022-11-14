// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

// dataType specifies one of AWS VPC flow field data types.
type dataType uint8

// List of DataTypes.
const (
	integerType dataType = iota + 1
	longType
	stringType
	ipType
	timestampType
)

var dataTypeNames = map[dataType]string{
	integerType:   "integer",
	longType:      "long",
	stringType:    "string",
	ipType:        "ip",
	timestampType: "timestamp",
}

func (dt dataType) String() string {
	if dt < integerType || timestampType < dt {
		return fmt.Sprintf("invaild(%d)", dt)
	}
	return dataTypeNames[dt]
}

// toType converts the given value string value to the specified data type.
func toType(value string, typ dataType) (interface{}, error) {
	switch typ {
	case stringType:
		return value, nil
	case longType:
		return toLong(value)
	case integerType:
		return toInteger(value)
	case ipType:
		return toIP(value)
	case timestampType:
		return toTimestamp(value)
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

func toIP(v string) (string, error) {
	// This is validating that the value is an IP.
	if net.ParseIP(v) != nil {
		return v, nil
	}
	return "", errors.New("value is not a valid IP address")
}

func toTimestamp(v string) (time.Time, error) {
	sec, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0).UTC(), nil
}
