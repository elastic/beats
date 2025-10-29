// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"fmt"
	"reflect"
	"strings"
	"regexp"
	"github.com/spf13/cast"
	"github.com/osquery/osquery-go/plugin/table"
)


// Filter represents an osquery constraint
type Filter struct {
	// ColumnName is the name of the column to filter on
	ColumnName string
	// Operator is the operator to use for the filter
	Operator table.Operator
	// Expression is the expression to use for the filter
	Expression string
}

// equals checks if the value of the field is equal to the expression
func (f Filter) equals(entry any) bool {
	field, kind, err := GetValueByOsqueryTag(entry, f.ColumnName)
	if err != nil {
		return false
	}
	switch kind {
	case reflect.String:
		return f.Expression == field.(string)
	case reflect.Bool:
		return cast.ToBool(f.Expression) == cast.ToBool(field)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cast.ToInt64(f.Expression) == cast.ToInt64(field)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cast.ToUint64(f.Expression) == cast.ToUint64(field)
	case reflect.Float64, reflect.Float32:
		return cast.ToFloat64(f.Expression) == cast.ToFloat64(field)
	default:
		return false
	}
}

// lessThan checks if the value of the field is less than the expression
func (f Filter) lessThan(entry any) bool {
	field, kind, err := GetValueByOsqueryTag(entry, f.ColumnName)
	if err != nil {
		return false
	}
	switch kind {
	case reflect.String:
		return false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cast.ToInt64(field) < cast.ToInt64(f.Expression)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cast.ToUint64(field) < cast.ToUint64(f.Expression)
	case reflect.Float64, reflect.Float32:
		return cast.ToFloat64(field) < cast.ToFloat64(f.Expression)
	case reflect.Bool:
		return cast.ToInt8(field.(bool)) < cast.ToInt8(cast.ToBool(f.Expression))
	default:
		return false
	}
}

// greaterThan checks if the value of the field is greater than the expression
func (f Filter) greaterThan(entry any) bool {
	field, kind, err := GetValueByOsqueryTag(entry, f.ColumnName)
	if err != nil {
		return false
	}
	switch kind {
	case reflect.String:
		castedField, err := cast.ToInt64E(field.(string))
		if err != nil {
			return false
		}
		castedExpression, err := cast.ToInt64E(f.Expression)
		if err != nil {
			return false
		}
		return castedField > castedExpression
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cast.ToInt64(field) > cast.ToInt64(f.Expression)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cast.ToUint64(field) > cast.ToUint64(f.Expression)
	case reflect.Float64, reflect.Float32:
		return cast.ToFloat64(field) > cast.ToFloat64(f.Expression)
	case reflect.Bool:
		return cast.ToInt8(field.(bool)) > cast.ToInt8(cast.ToBool(f.Expression))
	default:
		return false
	}
}

// like checks if the value of the field is like the expression
func (f Filter) like(entry any) bool {
	field, kind, err := GetValueByOsqueryTag(entry, f.ColumnName)
	if err != nil {
		return false
	}
	if kind != reflect.String {
		return false
	}
	pattern := strings.ReplaceAll(f.Expression, "%", "*")
	pattern = strings.ReplaceAll(pattern, "_", ".")
	pattern = "^" + pattern + "$"
	matched, err := regexp.MatchString(pattern, field.(string))
	if err != nil {
		return false
	}
	return matched
}

// glob checks if the value of the field is a glob pattern
func (f Filter) glob(entry any) bool {
	field, kind, err := GetValueByOsqueryTag(entry, f.ColumnName)
	if err != nil {
		return false
	}
	if kind != reflect.String {
		return false
	}
	pattern := strings.ReplaceAll(f.Expression, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "?", ".")
	pattern = "^" + pattern + "$"
	matched, err := regexp.MatchString(pattern, field.(string))
	if err != nil {
		return false
	}
	return matched
}

// Matches checks if the value of the field matches the expression
func (f Filter) Matches(entry any) bool {
	switch f.Operator {
	case table.OperatorEquals:
		return f.equals(entry)
	case table.OperatorGreaterThan:
		return f.greaterThan(entry)
	case table.OperatorLessThan:
		return f.lessThan(entry)
	case table.OperatorGreaterThanOrEquals:
		return f.equals(entry) || f.greaterThan(entry)
	case table.OperatorLessThanOrEquals:
		return f.equals(entry) || f.lessThan(entry)
	case table.OperatorLike:
		return f.like(entry)
	case table.OperatorGlob:
		return f.glob(entry)
	default:
		return false
	}
}

// GetConstraintFilters gets the constraints from the query context
func GetConstraintFilters(queryContext table.QueryContext) []Filter {
	var results []Filter
    for columnName, clist := range queryContext.Constraints {
		if len(clist.Constraints) == 0 {
			continue
		}
		for _, constraint := range clist.Constraints {
			if constraint.Operator == 73 {
				// ignore limit constraint for now, since it is not documented as a valid operator 
				// in either osquery documentation or the osquery-go library, but is is passed in the query context.
				// without knowledge of which records to limit, we should not apply it to the results and allow
				// osquery to handle it.
				continue
			}
			results = append(results, Filter{ColumnName: columnName, Operator: constraint.Operator, Expression: constraint.Expression})
		}
	}
	return results
}

// GetValueByTag gets the value of a field by its tag
func GetValueByOsqueryTag(s any, tagValue string) (any, reflect.Kind, error) {
	// Get the Value of the struct
	v := reflect.ValueOf(s)

	// If s is a pointer, we need to get the element it points to
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Make sure we're dealing with a struct
	if v.Kind() != reflect.Struct {
		return nil, reflect.Invalid, fmt.Errorf("not a struct")
	}

	// Get the Type of the struct
	t := v.Type()

	// Iterate over all fields in the struct
	for i := 0; i < t.NumField(); i++ {
		// Get the StructField, which contains tag info
		field := t.Field(i)

		// Get the tag value for the given key
		tag := field.Tag.Get("osquery")

		// Check if the tag value matches our target
		if tag == tagValue {
			// Found it! Get the Value of this field from the struct instance
			fieldValue := v.Field(i)
			// Return the value as a generic interface{}
			return fieldValue.Interface(), fieldValue.Kind(), nil
		}
	}

	// No field found with that tag
	return nil, reflect.Invalid, fmt.Errorf("no field found with that tag")
}
