// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

const LimitOperator table.Operator = 73

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
		fieldString, ok := field.(string); if !ok {
			return false
		}
		return f.Expression == fieldString
	case reflect.Bool:
		expressionBool, ok := ToBool(f.Expression); if !ok {
			return false
		}
		fieldBool, ok := ToBool(field); if !ok {
			return false
		}
		return expressionBool == fieldBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		expressionInt, ok := ToInt64(f.Expression); if !ok {
			return false
		}
		fieldInt, ok := ToInt64(field); if !ok {
			return false
		}
		return expressionInt == fieldInt
	case reflect.Float64, reflect.Float32:
		expressionFloat, ok := ToFloat64(f.Expression); if !ok {
			return false
		}
		fieldFloat, ok := ToFloat64(field); if !ok {
			return false
		}
		return expressionFloat == fieldFloat
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
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		expressionInt, ok := ToInt64(f.Expression); if !ok {
			return false
		}
		// Even though the field is an int, it may not cast to an int64, so we need to convert it
		// to be safe
		fieldInt, ok := ToInt64(field); if !ok {
			return false
		}
		return fieldInt < expressionInt
	case reflect.Float64, reflect.Float32:
		expressionFloat, ok := ToFloat64(f.Expression); if !ok {
			return false
		}
		fieldFloat, ok := ToFloat64(field); if !ok {
			return false
		}
		return fieldFloat < expressionFloat
	case reflect.Bool:
		fieldBool, ok := field.(bool); if !ok {
			return false
		}
		expressionBool, ok := ToBool(f.Expression); if !ok {
			return false
		}
		return !fieldBool && expressionBool
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
	fmt.Println("field", field)
	fmt.Println("kind", kind)
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		expressionInt, ok := ToInt64(f.Expression); if !ok {
			return false
		}
		// Even though the field is an int, it may not cast to an int64, so we need to convert it
		// to be safe
		fieldInt, ok := ToInt64(field); if !ok {
			return false
		}
		return fieldInt > expressionInt
	case reflect.Float64, reflect.Float32:
		expressionFloat, ok := ToFloat64(f.Expression); if !ok {
			return false
		}
		fieldFloat, ok := ToFloat64(field); if !ok {
			return false
		}
		return fieldFloat > expressionFloat
	case reflect.Bool:
		fieldBool, ok := field.(bool); if !ok {
			return false
		}
		expressionBool, ok := ToBool(f.Expression); if !ok {
			return false
		}
		return fieldBool && !expressionBool
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
	pattern := strings.ReplaceAll(f.Expression, "%", ".*")
	pattern = strings.ReplaceAll(pattern, "_", ".")
	pattern = "^" + pattern + "$"
	fieldString, ok := field.(string); if !ok {
		return false
	}
	matched, err := regexp.MatchString(pattern, fieldString)
	if err != nil || !matched {
		return false
	}
	return true
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
	fieldString, ok := field.(string); if !ok {
		return false
	}
	matched, err := regexp.MatchString(pattern, fieldString)
	if err != nil || !matched {
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
	case table.OperatorRegexp:
		// TODO: implement, conversion is not simple
		return false
	case table.OperatorMatch:
		// TODO: implement? possibly N/A, only works with full text search tables
		return false
	case table.OperatorUnique:
		// TODO: implement
		return false // TODO: implement
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
			if constraint.Operator == LimitOperator {
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
			fmt.Println("fieldValue", fieldValue)
			fmt.Println("fieldValue.Kind()", fieldValue.Kind())
			return fieldValue.Interface(), fieldValue.Kind(), nil
		}
	}

	// No field found with that tag
	return nil, reflect.Invalid, fmt.Errorf("no field found with that tag")
}
