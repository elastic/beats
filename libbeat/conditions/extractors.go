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

package conditions

import (
	"fmt"
	"math"
	"strconv"
)

// ExtractFloat extracts a float from an unknown type.
func ExtractFloat(unk interface{}) (float64, error) {
	if v, ok := extractFloat(unk); ok {
		return v, nil
	}
	if s, ok := unk.(string); ok {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return math.NaN(), err
		}
		return f, nil
	}
	return math.NaN(), fmt.Errorf("unknown type %T passed to ExtractFloat", unk)
}

func extractFloat(unk interface{}) (float64, bool) {
	switch i := unk.(type) {
	case float64:
		return float64(i), true
	case float32:
		return float64(i), true
	case int64:
		return float64(i), true
	case int32:
		return float64(i), true
	case int16:
		return float64(i), true
	case int8:
		return float64(i), true
	case uint64:
		return float64(i), true
	case uint32:
		return float64(i), true
	case uint16:
		return float64(i), true
	case uint8:
		return float64(i), true
	case int:
		return float64(i), true
	case uint:
		return float64(i), true
	default:
		return 0, false
	}
}

// ExtractInt extracts an int from an unknown type.
func ExtractInt(unk interface{}) (uint64, error) {
	if v, ok := extractInt(unk); ok {
		return v, nil
	}
	return 0, fmt.Errorf("unknown type %T passed to ExtractInt", unk)
}

func extractInt(unk interface{}) (uint64, bool) {
	switch i := unk.(type) {
	case int64:
		return uint64(i), true
	case int32:
		return uint64(i), true
	case int16:
		return uint64(i), true
	case int8:
		return uint64(i), true
	case uint64:
		return uint64(i), true
	case uint32:
		return uint64(i), true
	case uint16:
		return uint64(i), true
	case uint8:
		return uint64(i), true
	case int:
		return uint64(i), true
	case uint:
		return uint64(i), true
	default:
		return 0, false
	}
}

// ExtractString extracts a string from an unknown type.
func ExtractString(unk interface{}) (string, error) {
	if s, ok := unk.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("unknown type %T passed to ExtractString", unk)
}

// ExtractBool extracts a bool from an unknown type.
func ExtractBool(unk interface{}) (bool, error) {
	if b, ok := unk.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("unknown type %T passed to ExtractBool", unk)
}
