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

package wmi

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"unicode"

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

func ConvertDatetime(v string) (interface{}, error) {
	layout := "20060102150405.999999-0700"
	return time.Parse(layout, v+"0")
}

func ConvertString(v string) (interface{}, error) {
	return v, nil
}

// Function that determines if a given value requires additional conversion
// This holds true for strings that encode uint64, sint64 and datetime format
func RequiresExtraConversion(fieldValue interface{}) bool {
	stringValue, isString := fieldValue.(string)
	if !isString {
		return false
	}
	isEmptyString := len(stringValue) == 0

	// Heuristic to avoid fetching the raw properties for every string property
	//   If the string is empty, no need to convert the string
	//   If the string does not end with a digit, it's not an uint64, sint64, datetime
	return !isEmptyString && unicode.IsDigit(rune(stringValue[len(stringValue)-1]))
}

// Given a Property it returns its CIM Type Qualifier
// https://learn.microsoft.com/en-us/windows/win32/wmisdk/cimtype-qualifier
// We assume that it is **always** defined for every property to simiplifying
// The error handling
func getPropertyType(property *ole.IDispatch) base.WmiType {
	rawType := oleutil.MustGetProperty(property, "CIMType")

	value, err := wmi.GetVariantValue(rawType)
	if err != nil {
		panic("Error retrieving the wmi property type")
	}

	return base.WmiType(value.(int32))
}

// Returns the "raw" SWbemProperty containing type information for a given field.
//
// The microsoft/wmi library does not have a function that given an instance and a property name
// returns the wmi.wmiProperty object. This function mimics the behavior of the `GetSystemProperty`
// method in the wmi.wmiInstance struct and applies it on the Properties_ field
// https://github.com/microsoft/wmi/blob/v0.25.2/pkg/wmiinstance/WmiInstance.go#L87
//
// Note: We are not instantiating a wmi.wmiProperty because of this issue
// https://github.com/microsoft/wmi/issues/150
// Once this issue is resolved, we can instantiate a wmi.WmiProperty and eliminate
// the need for the "getPropertyType" function.
func getProperty(instance *wmi.WmiInstance, propertyName string) (*ole.IDispatch, error) {
	// Documentation: https://learn.microsoft.com/en-us/windows/win32/wmisdk/swbemobject-properties-
	rawResult, err := oleutil.GetProperty(instance.GetIDispatch(), "Properties_")
	if err != nil {
		return nil, err
	}

	// SWbemObjectEx.Properties_ returns
	// an SWbemPropertySet object that contains the collection
	// of properties for the c class
	sWbemObjectExAsIDispatch := rawResult.ToIDispatch()
	defer rawResult.Clear()

	// Get the property
	sWbemProperty, err := oleutil.CallMethod(sWbemObjectExAsIDispatch, "Item", propertyName)
	if err != nil {
		return nil, err
	}

	return sWbemProperty.ToIDispatch(), nil
}

// Given an instance and a property Name, it returns the appropriate conversion function
func GetConvertFunction(instance *wmi.WmiInstance, propertyName string) (WmiStringConversionFunction, error) {
	rawProperty, err := getProperty(instance, propertyName)
	if err != nil {
		return nil, err
	}
	propType := getPropertyType(rawProperty)

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
func ExecuteGuardedQueryInstances(session WmiQueryInterface, query string, timeout time.Duration) ([]*wmi.WmiInstance, error) {
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
			baseMessage := fmt.Sprintf("The timed out query '%s' terminated after %s", query, timeSince)
			// We eventually fetched the documents, let us free them
			if err == nil {
				logp.Warn("%s successfully. The result will be ignored", baseMessage)
				wmi.CloseAllInstances(rows)
			} else {
				logp.Warn("%s with an error %v", baseMessage, err)
			}
		}
	}()

	select {
	case <-ctx.Done():
		err = fmt.Errorf("the execution of the query'%s' exceeded the threshold of %s", query, timeout)
		timedout = true
		close(done)
	case <-done:
		// Query completed in time either successfully or with an error
	}
	return rows, err
}
