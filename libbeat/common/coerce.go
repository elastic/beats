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

package common

import "strconv"

// TryToInt tries to coerce the given interface to an int. On success it returns
// the int value and true.
func TryToInt(number interface{}) (int, bool) {
	var rtn int
	switch v := number.(type) {
	case int:
		rtn = int(v)
	case int8:
		rtn = int(v)
	case int16:
		rtn = int(v)
	case int32:
		rtn = int(v)
	case int64:
		rtn = int(v)
	case uint:
		rtn = int(v)
	case uint8:
		rtn = int(v)
	case uint16:
		rtn = int(v)
	case uint32:
		rtn = int(v)
	case uint64:
		rtn = int(v)
	case string:
		var err error
		rtn, err = strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return rtn, true
}

// TryToFloat64 tries to coerce the given interface to an float64. It accepts
// a float32, float64, or string. On success it returns the float64 value and
// true.
func TryToFloat64(number interface{}) (float64, bool) {
	var rtn float64
	switch v := number.(type) {
	case float32:
		rtn = float64(v)
	case float64:
		rtn = v
	case string:
		var err error
		rtn, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return rtn, true
}
