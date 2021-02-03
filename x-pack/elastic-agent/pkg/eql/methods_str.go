// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// concat concatenates the arguments into a string
func concat(args []interface{}) (interface{}, error) {
	var sb strings.Builder
	for _, arg := range args {
		sb.WriteString(toString(arg))
	}
	return sb.String(), nil
}

// endsWith returns true if the string ends with given suffix
func endsWith(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("endsWith: accepts exactly 2 arguments; recieved %d", len(args))
	}
	input, iOk := args[0].(string)
	suffix, sOk := args[1].(string)
	if !iOk || !sOk {
		return nil, fmt.Errorf("endsWith: accepts exactly 2 string arguments; recieved %T and %T", args[0], args[1])
	}
	return strings.HasSuffix(input, suffix), nil
}

// indexOf returns the starting index of substring
func indexOf(args []interface{}) (interface{}, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("indexOf: accepts 2-3 arguments; recieved %d", len(args))
	}
	input, iOk := args[0].(string)
	substring, sOk := args[1].(string)
	if !iOk || !sOk {
		return nil, fmt.Errorf("indexOf: argument 0 and 1 must be a string; recieved %T and %T", args[0], args[1])
	}
	start := 0
	if len(args) > 2 {
		s, sOk := args[2].(int)
		if !sOk {
			return nil, fmt.Errorf("indexOf: argument 2 must be a integer; recieved %T", args[2])
		}
		start = s
	}
	return start + strings.Index(input[start:], substring), nil
}

// match returns true if the string matches any of the provided regular expressions
func match(args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("match: accepts minimum of 2 arguments; recieved %d", len(args))
	}
	input, iOk := args[0].(string)
	if !iOk {
		return nil, fmt.Errorf("match: argument 0 must be a string; recieved %T", args[0])
	}
	for i, reg := range args[1:] {
		switch r := reg.(type) {
		case string:
			exp, err := regexp.Compile(r)
			if err != nil {
				return nil, fmt.Errorf("match: failed to compile regexp: %s", err)
			}
			if exp.Match([]byte(input)) {
				return true, nil
			}
		default:
			return nil, fmt.Errorf("match: argument %d must be a string; recieved %T", i+1, reg)
		}
	}
	return false, nil
}

// number converts the string into a integer
func number(args []interface{}) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("number: accepts between 1-2 arguments; recieved %d", len(args))
	}
	input, iOk := args[0].(string)
	if !iOk {
		return nil, fmt.Errorf("number: argument 0 must be a string; recieved %T", args[0])
	}
	base := 10
	if len(args) > 1 {
		switch a := args[1].(type) {
		case int:
			base = a
		default:
			return nil, fmt.Errorf("number: argument 1 must be an integer; recieved %T", args[1])
		}
	}
	if strings.HasPrefix(input, "0x") {
		input = input[2:]
	}
	n, err := strconv.ParseInt(input, base, 64)
	if err != nil {
		return nil, fmt.Errorf("number: failed to convert '%s' to integer", input)
	}
	return int(n), nil
}

// startsWith returns true if the string starts with given prefix
func startsWith(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("startsWith: accepts exactly 2 arguments; recieved %d", len(args))
	}
	input, iOk := args[0].(string)
	prefix, pOk := args[1].(string)
	if !iOk || !pOk {
		return nil, fmt.Errorf("startsWith: accepts exactly 2 string arguments; recieved %T and %T", args[0], args[1])
	}
	return strings.HasPrefix(input, prefix), nil
}

// str converts the argument into a string
func str(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string: accepts exactly 1 argument; recieved %d", len(args))
	}
	return toString(args[0]), nil
}

// stringContains returns true if the string contains substring
func stringContains(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("stringContains: accepts exactly 2 arguments; recieved %d", len(args))
	}
	input, iOk := args[0].(string)
	substr, sOk := args[1].(string)
	if !iOk || !sOk {
		return nil, fmt.Errorf("stringContains: accepts exactly 2 string arguments; recieved %T and %T", args[0], args[1])
	}
	return strings.Contains(input, substr), nil
}

func toString(arg interface{}) string {
	switch a := arg.(type) {
	case *null:
		return "null"
	case string:
		return a
	case int:
		return strconv.Itoa(a)
	case float64:
		return strconv.FormatFloat(a, 'E', -1, 64)
	case bool:
		return strconv.FormatBool(a)
	case []interface{}:
		var sb strings.Builder
		sb.WriteString("[")
		for idx, item := range a {
			sb.WriteString(toString(item))
			if idx < len(a)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString("]")
		return sb.String()
	case map[string]interface{}:
		var sb strings.Builder
		sb.WriteString("{")
		idx := 0
		for k, v := range a {
			sb.WriteString(k)
			sb.WriteString(":")
			sb.WriteString(toString(v))
			if idx < len(a)-1 {
				sb.WriteString(",")
			}
			idx++
		}
		sb.WriteString("}")
		return sb.String()
	default:
		return fmt.Sprintf("%s", a)
	}
}
