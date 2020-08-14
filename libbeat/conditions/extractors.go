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
	switch i := unk.(type) {
	case float64:
		return float64(i), nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int16:
		return float64(i), nil
	case int8:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint16:
		return float64(i), nil
	case uint8:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint:
		return float64(i), nil
	case string:
		f, err := strconv.ParseFloat(i, 64)
		if err != nil {
			return math.NaN(), err
		}
		return f, err
	default:
		return math.NaN(), fmt.Errorf("unknown type %T passed to ExtractFloat", unk)
	}
}

// ExtractInt extracts an int from an unknown type.
func ExtractInt(unk interface{}) (uint64, error) {
	switch i := unk.(type) {
	case int64:
		return uint64(i), nil
	case int32:
		return uint64(i), nil
	case int16:
		return uint64(i), nil
	case int8:
		return uint64(i), nil
	case uint64:
		return uint64(i), nil
	case uint32:
		return uint64(i), nil
	case uint16:
		return uint64(i), nil
	case uint8:
		return uint64(i), nil
	case int:
		return uint64(i), nil
	case uint:
		return uint64(i), nil
	default:
		return 0, fmt.Errorf("unknown type %T passed to ExtractInt", unk)
	}
}

// ExtractString extracts a string from an unknown type.
func ExtractString(unk interface{}) (string, error) {
	switch s := unk.(type) {
	case string:
		return s, nil
	default:
		return "", fmt.Errorf("unknown type %T passed to ExtractString", unk)
	}
}

// ExtractBool extracts a bool from an unknown type.
func ExtractBool(unk interface{}) (bool, error) {
	switch b := unk.(type) {
	case bool:
		return b, nil
	default:
		return false, fmt.Errorf("unknown type %T passed to ExtractBool", unk)
	}
}
