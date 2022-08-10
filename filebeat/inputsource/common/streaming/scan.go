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

package streaming

import (
	"bufio"
	"bytes"
	"strconv"
)

// FactoryDelimiter return a function to split line using a custom delimiter supporting multibytes
// delimiter, the delimiter is stripped from the returned value.
func FactoryDelimiter(delimiter []byte) bufio.SplitFunc {
	return func(data []byte, eof bool) (int, []byte, error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, delimiter); i >= 0 {
			return i + len(delimiter), dropDelimiter(data[0:i], delimiter), nil
		}
		if eof {
			return len(data), dropDelimiter(data, delimiter), nil
		}
		return 0, nil, nil
	}
}

func dropDelimiter(data []byte, delimiter []byte) []byte {
	if len(data) > len(delimiter) &&
		bytes.Equal(data[len(data)-len(delimiter):], delimiter) {
		return data[0 : len(data)-len(delimiter)]
	}
	return data
}

// FactoryRFC6587Framing returns a function that splits based on octet
// counting or non-transparent framing as defined in RFC6587.  Allows
// for custom delimter for non-transparent framing.
func FactoryRFC6587Framing(delimiter []byte) bufio.SplitFunc {
	return func(data []byte, eof bool) (int, []byte, error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}
		// need at least one character to see if octet or
		// non transparent framing
		if len(data) <= 1 {
			return 0, nil, nil
		}
		// It can be assumed that octet-counting framing is
		// used if a syslog frame starts with a digit RFC6587
		if bytes.ContainsAny(data[0:1], "0123456789") {
			if i := bytes.IndexByte(data, ' '); i > 0 {
				length, err := strconv.Atoi(string(data[0:i]))
				if err != nil {
					return 0, nil, err
				}
				end := length + i + 1
				if len(data) >= end {
					return end, data[i+1 : end], nil
				}
			}
			// request more data
			return 0, nil, nil
		}
		if i := bytes.Index(data, delimiter); i >= 0 {
			return i + len(delimiter), dropDelimiter(data[0:i], delimiter), nil
		}
		if eof {
			return len(data), dropDelimiter(data, delimiter), nil
		}
		// request more data
		return 0, nil, nil
	}
}
