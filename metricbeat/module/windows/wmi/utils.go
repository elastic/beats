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

//go:build windows

package wmi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	base "github.com/microsoft/wmi/go/wmi"
	wmi "github.com/microsoft/wmi/pkg/wmiinstance"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Utilities related to Type conversion

// WmiStringConversionFunction defines a function type for converting string values
// into other data types, such as integers or timestamps.
type WmiStringConversionFunction func(string) (interface{}, error)

func ConvertUint64(v string) (interface{}, error) {
	return strconv.ParseUint(v, 10, 64)
}

func ConvertSint64(v string) (interface{}, error) {
	return strconv.ParseInt(v, 10, 64)
}

const WMI_DATETIME_LAYOUT string = "20060102150405.999999"
const TIMEZONE_LAYOUT string = "-07:00"

// The CIMDateFormat is defined as "yyyymmddHHMMSS.mmmmmmsUUU".
// Example: "20231224093045.123456+000"
// More information: https://learn.microsoft.com/en-us/windows/win32/wmisdk/cim-datetime
//
// The "yyyyMMddHHmmSS.mmmmmm" part can be parsed using time.Parse, but Go's time package does not support parsing the "sUUU"
// part (the sign and minute offset from UTC).
//
// Here, "s" represents the sign (+ or -), and "UUU" represents the UTC offset in minutes.
//
// The approach for handling this is:
// 1. Extract the sign ('+' or '-') from the string.
// 2. Normalize the offset from minutes to the standard `hh:mm` format.
// 3. Concatenate the "yyyyMMddHHmmSS.mmmmmm" part with the normalized offset.
// 4. Parse the combined string using time.Parse to return a time.Date object.
func ConvertDatetime(v string) (interface{}, error) {
	if len(v) != 25 {
		return nil, fmt.Errorf("datetime is invalid: the datetime is expected to be exactly 25 characters long, got: %s", v)
	}

	// Extract the sign (either '+' or '-')
	utcOffsetSign := v[21]
	if utcOffsetSign != '+' && utcOffsetSign != '-' {
		return nil, fmt.Errorf("datetime is invalid: the offset sign is expected to be either + or -")
	}

	// Extract UTC offset (last 3 characters)
	utcOffsetStr := v[22:]
	utcOffset, err := strconv.ParseInt(utcOffsetStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("datetime is invalid: error parsing UTC offset: %w", err)
	}
	offsetHours := utcOffset / 60
	offsetMinutes := utcOffset % 60

	// Build the complete date string including the UTC offset in the format yyyyMMddHHmmss.mmmmmm+hh:mm
	// Concatenate the date string with the offset formatted as "+hh:mm"
	dateString := fmt.Sprintf("%s%c%02d:%02d", v[:21], utcOffsetSign, offsetHours, offsetMinutes)

	// Parse the combined datetime string using the defined layout
	date, err := time.Parse(WMI_DATETIME_LAYOUT+TIMEZONE_LAYOUT, dateString)
	if err != nil {
		return nil, fmt.Errorf("datetime is invalid: error parsing the final datetime: %w", err)
	}

	return date, err
}

func ConvertString(v string) (interface{}, error) {
	return v, nil
}

// Function that determines if a given value requires additional conversion
// This holds true for strings that encode uint64, sint64 and datetime format
func RequiresExtraConversion(propertyValue interface{}) bool {
	stringValue, isString := propertyValue.(string)
	if !isString {
		return false
	}
	isEmptyString := len(stringValue) == 0

	// Heuristic to avoid fetching the raw properties for every string property
	//   If the string is empty, no need to convert the string
	//   If the string does not end with a digit, it's not an uint64, sint64, datetime
	return !isEmptyString && stringValue[len(stringValue)-1] >= '0' && stringValue[len(stringValue)-1] <= '9'
}

// Given a Property it returns its CIM Type Qualifier
// https://learn.microsoft.com/en-us/windows/win32/wmisdk/cimtype-qualifier
func getPropertyType(property *ole.IDispatch) (base.WmiType, error) {
	rawType := oleutil.MustGetProperty(property, "CIMType")

	value, err := wmi.GetVariantValue(rawType)
	if err != nil {
		return base.WmiType(0), err
	}

	v, ok := value.(int32)
	if !ok {
		return 0, fmt.Errorf("type assertion to int32 failed")
	}

	return base.WmiType(v), nil
}

// Returns the "raw" SWbemProperty containing type information for a given property.
//
// The microsoft/wmi library does not have a function that given an instance and a property name
// returns the wmi.wmiProperty object. This function mimics the behavior of the `GetSystemProperty`
// method in the wmi.wmiInstance struct and applies it on the Properties_ property
// https://github.com/microsoft/wmi/blob/v0.25.2/pkg/wmiinstance/WmiInstance.go#L87
//
// Note: We are not instantiating a wmi.wmiProperty because of this issue
// https://github.com/microsoft/wmi/issues/150
// Once this issue is resolved, we can instantiate a wmi.WmiProperty and eliminate
// the need for the "getPropertyType" function.
func getProperty(instance *wmi.WmiInstance, propertyName string, logger *logp.Logger) (*ole.IDispatch, error) {
	// Documentation: https://learn.microsoft.com/en-us/windows/win32/wmisdk/swbemobject-properties-
	rawResult, err := oleutil.GetProperty(instance.GetIDispatch(), "Properties_")
	if err != nil {
		return nil, err
	}

	// SWbemObjectEx.Properties_ returns
	// an SWbemPropertySet object that contains the collection
	// of properties for the c class
	sWbemObjectExAsIDispatch := rawResult.ToIDispatch()
	defer func() {
		if cerr := rawResult.Clear(); cerr != nil {
			logger.Debugf("failed to release connection: %w", err)
		}
	}()

	// Get the property
	sWbemProperty, err := oleutil.CallMethod(sWbemObjectExAsIDispatch, "Item", propertyName)
	if err != nil {
		return nil, err
	}

	return sWbemProperty.ToIDispatch(), nil
}

// Given an instance and a property Name, it returns the appropriate conversion function
func GetConvertFunction(instance *wmi.WmiInstance, propertyName string, logger *logp.Logger) (WmiStringConversionFunction, error) {
	rawProperty, err := getProperty(instance, propertyName, logger)
	if err != nil {
		return nil, err
	}
	propType, err := getPropertyType(rawProperty)
	if err != nil {
		return nil, fmt.Errorf("could not fetch CIMType for property '%s' with error %w", propertyName, err)
	}

	var f WmiStringConversionFunction

	switch propType {
	case base.WbemCimtypeDatetime:
		f = ConvertDatetime
	case base.WbemCimtypeUint64:
		f = ConvertUint64
	case base.WbemCimtypeSint64:
		f = ConvertSint64
	default: // For all other types we return the identity function
		f = ConvertString
	}
	return f, err
}

// Utilities related to Warning Threshold

// Define an interface to allow unit-testing long-running queries
// *wmi.wmiSession is an implementation of this interface
type WmiQueryInterface interface {
	QueryInstances(query string) ([]*wmi.WmiInstance, error)
}

// Wrapper of the session.QueryInstances function that execute a query for at most a timeout
// after which we stop actively waiting.
// Note that the underlying query will continue to run, until the query completes or the WMI Arbitrator stops the query
// https://learn.microsoft.com/en-us/troubleshoot/windows-server/system-management-components/new-wmi-arbitrator-behavior-in-windows-server
func ExecuteGuardedQueryInstances(session WmiQueryInterface, query string, timeout time.Duration, logger *logp.Logger) ([]*wmi.WmiInstance, error) {
	var rows []*wmi.WmiInstance
	var err error
	done := make(chan error)
	timedout := false

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		start_time := time.Now()
		rows, err = session.QueryInstances(query)
		if !timedout {
			done <- err
		} else {
			timeSince := time.Since(start_time)
			baseMessage := fmt.Sprintf("The query '%s' that exceeded the warning threshold terminated after %s", query, timeSince)
			var tailMessage string
			// We eventually fetched the documents, let us free them
			if err == nil {
				tailMessage = "successfully. The result will be ignored"
				wmi.CloseAllInstances(rows)
			} else {
				tailMessage = fmt.Sprintf("with an error %v", err)
			}
			logger.Warn("%s %s", baseMessage, tailMessage)
		}
	}()

	select {
	case <-ctx.Done():
		err = fmt.Errorf("the execution of the query '%s' exceeded the warning threshold of %s", query, timeout)
		timedout = true
		close(done)
	case <-done:
		// Query completed in time either successfully or with an error
	}
	return rows, err
}
