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

	"github.com/cespare/xxhash/v2"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	base "github.com/microsoft/wmi/go/wmi"
	wmi "github.com/microsoft/wmi/pkg/wmiinstance"

	"github.com/elastic/go-freelru"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Utilities related to Type conversion

// Function that convert single strings
type internalWmiConversionFunction[T any] func(string) (T, error)

func internalConvertUint64(v string) (uint64, error) {
	return strconv.ParseUint(v, 10, 64)
}

func internalConvertSint64(v string) (int64, error) {
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
func internalConvertDateTime(v string) (time.Time, error) {

	if len(v) != 25 {
		return time.Time{}, fmt.Errorf("datetime is invalid: the datetime is expected to be exactly 25 characters long, got: %s", v)
	}

	// Extract the sign (either '+' or '-')
	utcOffsetSign := v[21]
	if utcOffsetSign != '+' && utcOffsetSign != '-' {
		return time.Time{}, fmt.Errorf("datetime is invalid: the offset sign is expected to be either + or -")
	}

	// Extract UTC offset (last 3 characters)
	utcOffsetStr := v[22:]
	utcOffset, err := strconv.ParseInt(utcOffsetStr, 10, 16)
	if err != nil {
		return time.Time{}, fmt.Errorf("datetime is invalid: error parsing UTC offset: %w", err)
	}
	offsetHours := utcOffset / 60
	offsetMinutes := utcOffset % 60

	// Build the complete date string including the UTC offset in the format yyyyMMddHHmmss.mmmmmm+hh:mm
	// Concatenate the date string with the offset formatted as "+hh:mm"
	dateString := fmt.Sprintf("%s%c%02d:%02d", v[:21], utcOffsetSign, offsetHours, offsetMinutes)

	// Parse the combined datetime string using the defined layout
	date, err := time.Parse(WMI_DATETIME_LAYOUT+TIMEZONE_LAYOUT, dateString)
	if err != nil {
		return time.Time{}, fmt.Errorf("datetime is invalid: error parsing the final datetime: %w", err)
	}

	return date, err
}

// Type conversion that applies to both arrays and scalars
type WmiConversionFunction func(interface{}) (interface{}, error)

// General-purpose function that invokes the internal WMI conversion on
// both strings and arrays to avoid code duplication.
func GenericWmiConversionFunction[T any](v interface{}, convert internalWmiConversionFunction[T]) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch value := v.(type) {
	case string:
		return convert(value)
	case []interface{}:
		results := make([]T, 0, len(value))
		for i, raw := range value {
			str, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("expected array of strings, got %v at index %d", raw, i)
			}
			result, err := convert(str)
			if err != nil {
				return nil, fmt.Errorf("invalid string at index %d: %w", i, err)
			}
			results = append(results, result)
		}
		return results, nil
	default:
		return nil, fmt.Errorf("expected string or an array of strings, got %T", v)
	}
}

func ConvertUint64(v interface{}) (interface{}, error) {
	return GenericWmiConversionFunction[uint64](v, internalConvertUint64)
}

func ConvertSint64(v interface{}) (interface{}, error) {
	return GenericWmiConversionFunction[int64](v, internalConvertSint64)
}

func ConvertDatetime(v interface{}) (interface{}, error) {
	return GenericWmiConversionFunction[time.Time](v, internalConvertDateTime)
}

func ConvertIdentity(v interface{}) (interface{}, error) {
	return v, nil
}

// Function that returns a WmiConversionFunction that reports an error
// independently of the input value.
func getInvalidConversion(err error) WmiConversionFunction {
	return func(v interface{}) (interface{}, error) {
		return nil, err
	}
}

// Hash Function used for the LRU impementation
func hashStringXXHASH(s string) uint32 {
	// Slight adaptation of https://github.com/elastic/go-freelru/tree/main
	// That truncates the 64-bit hash explicitely to get rid of the security warning:
	// G115: integer overflow conversion uint64 -> uint32 (gosec)
	return uint32(xxhash.Sum64String(s) & 0xFFFFFFFF)
}

type WMISchema struct {
	SubClassSchemas *freelru.LRU[string, map[string]WmiConversionFunction]
}

func (ws WMISchema) Get(class string, property string) (WmiConversionFunction, bool) {
	classSchema, ok := ws.SubClassSchemas.Get(class)
	if !ok {
		// This case is actually unexpected, because we invoke PutClass before and we proceed sequentially
		return getInvalidConversion(fmt.Errorf("could not find class %s in cache", class)), ok
	}
	val, ok := classSchema[property]
	return val, ok
}

func (ws *WMISchema) PutClass(class string) map[string]WmiConversionFunction {
	v, ok := ws.SubClassSchemas.Get(class)
	if !ok {
		v = make(map[string]WmiConversionFunction)
		ws.SubClassSchemas.Add(class, v)
	}
	return v
}

func (ws *WMISchema) Put(class string, key string, wcf WmiConversionFunction) {
	classSchema := ws.PutClass(class)
	classSchema[key] = wcf
}

func NewWMISchema(size uint32) (*WMISchema, error) {
	flu, err := freelru.New[string, map[string]WmiConversionFunction](size, hashStringXXHASH)
	if err != nil {
		return nil, err
	}

	return &WMISchema{
		SubClassSchemas: flu,
	}, nil
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
func GetConvertFunction(instance *wmi.WmiInstance, propertyName string, logger *logp.Logger) (WmiConversionFunction, error) {
	rawProperty, err := getProperty(instance, propertyName, logger)
	if err != nil {
		return nil, err
	}
	propType, err := getPropertyType(rawProperty)
	if err != nil {
		return nil, fmt.Errorf("could not fetch CIMType for property '%s' with error %w", propertyName, err)
	}

	var f WmiConversionFunction

	switch propType {
	case base.WbemCimtypeDatetime:
		f = ConvertDatetime
	case base.WbemCimtypeUint64:
		f = ConvertUint64
	case base.WbemCimtypeSint64:
		f = ConvertSint64
	case base.WbemCimtypeObject:
		// NOTE: The WMI CIM type 'object' is intentionally not supported here.
		//
		// Supporting embedded object types would require complex COM marshaling,
		// and without further processing can cause go panic during (JSON) marshalling
		//
		// If you have a real-world need for this, please open a GitHub issue to discuss.
		f = getInvalidConversion(fmt.Errorf("the Type %s is unsupported. Consider flattening your class", "CIM Type Object"))

	default: // For all other types we return the identity function
		f = ConvertIdentity
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
			logger.Warnf("%s %s", baseMessage, tailMessage)
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

func errorOnClassDoesNotExist(rows []*wmi.WmiInstance, class string, namespace string) error {
	switch len(rows) {
	case 0:
		return fmt.Errorf("class '%s' not found in namespace '%s'", class, namespace)
	case 1:
		return nil
	default:
		return fmt.Errorf("unexpected case: Metaclass should return only a single entry for the class %s", class)
	}
}

// Given an instance a list of desired properties, it returns two arrays:
//
// Valid Properties: the list of properties that it's both desired and in the class
// Invalid properties the list of properties that are desired but are not part of the class
func filterValidProperties(instance_properties []string, properties []string) ([]string, []string) {
	if len(properties) == 0 {
		return instance_properties, []string{}
	}

	// Create the map for membership checks
	set := make(map[string]struct{})
	for _, item := range instance_properties {
		set[item] = struct{}{} // struct{} takes 0 bytes
	}

	validProperties := []string{}
	invalidProperties := []string{}
	for _, p := range properties {
		if _, exists := set[p]; exists {
			validProperties = append(validProperties, p)
		} else {
			invalidProperties = append(invalidProperties, p)
		}
	}
	return validProperties, invalidProperties
}

func validateQueryFields(instance *wmi.WmiInstance, queryConfig *QueryConfig, logger *logp.Logger) error {

	// We are using '*', so we don't modify the queryConfig
	if len(queryConfig.Properties) == 0 {
		return nil
	}

	// Valid Properties contains the properties that are both contained in the
	// user-provided lists and in the properties of the class
	validProperties, invalidProperties := filterValidProperties(instance.GetClass().GetPropertiesNames(), queryConfig.Properties)

	if len(validProperties) == 0 {
		return fmt.Errorf("all the properties listed are invalid %v. We are skipping the query", invalidProperties)
	}

	if len(invalidProperties) > 0 {
		logger.Warnf("We are going to ignore the properties '%v' because '%s' class in namespace '%s' does not contain those properties. Please amend your configuration or check why it's the case", invalidProperties, queryConfig.Class, queryConfig.Namespace)
	}

	queryConfig.Properties = validProperties

	return nil
}
