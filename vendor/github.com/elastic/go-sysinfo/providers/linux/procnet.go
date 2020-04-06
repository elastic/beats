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

package linux

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

// fillStruct is some reflection work that can dynamically fill one of our tagged `netstat` structs with netstat data
func fillStruct(str interface{}, data map[string]map[string]uint64) {
	val := reflect.ValueOf(str).Elem()
	typ := reflect.TypeOf(str).Elem()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag := field.Tag.Get("netstat"); tag != "" {
			if values, ok := data[tag]; ok {
				val.Field(i).Set(reflect.ValueOf(values))
			}
		}
	}
}

// parseEntry parses two lines from the net files, the first line being keys, the second being values
func parseEntry(line1, line2 string) (map[string]uint64, error) {
	keyArr := strings.Split(strings.TrimSpace(line1), " ")
	valueArr := strings.Split(strings.TrimSpace(line2), " ")

	if len(keyArr) != len(valueArr) {
		return nil, errors.New("key and value lines are mismatched")
	}

	counters := make(map[string]uint64, len(valueArr))
	for iter, value := range valueArr {

		// This if-else block is to deal with the MaxConn value in SNMP,
		// which is a signed value according to RFC2012.
		// This library emulates the behavior of the kernel: store all values as a uint, then cast to a signed value for printing
		// Users of this library need to be aware that this value should be printed as a signed int or hex value to make it useful.
		var parsed uint64
		var err error
		if strings.Contains(value, "-") {
			signedParsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing string to int in line: %#v", valueArr)
			}
			parsed = uint64(signedParsed)
		} else {
			parsed, err = strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing string to int in line: %#v", valueArr)
			}
		}

		counters[keyArr[iter]] = parsed
	}
	return counters, nil
}

// parseNetFile parses an entire file, and returns a 2D map, representing how files are sorted by protocol
func parseNetFile(body string) (map[string]map[string]uint64, error) {
	fileMetrics := make(map[string]map[string]uint64)
	bodySplit := strings.Split(strings.TrimSpace(body), "\n")
	// There should be an even number of lines. If not, something is wrong.
	if len(bodySplit)%2 != 0 {
		return nil, fmt.Errorf("badly parsed body: %s", body)
	}
	// in the network counters, data is divided into two-line sections: a line of keys, and a line of values
	// With each line
	for index := 0; index < len(bodySplit); index += 2 {
		keysSplit := strings.Split(bodySplit[index], ":")
		valuesSplit := strings.Split(bodySplit[index+1], ":")
		if len(keysSplit) != 2 || len(valuesSplit) != 2 {
			return nil, fmt.Errorf("wrong number of keys: %#v", keysSplit)
		}
		valMap, err := parseEntry(keysSplit[1], valuesSplit[1])
		if err != nil {
			return nil, errors.Wrap(err, "error parsing lines")
		}
		fileMetrics[valuesSplit[0]] = valMap
	}
	return fileMetrics, nil
}

// getNetSnmpStats pulls snmp stats from /proc/net
func getNetSnmpStats(raw []byte) (types.SNMP, error) {
	snmpData, err := parseNetFile(string(raw))
	if err != nil {
		return types.SNMP{}, errors.Wrap(err, "error parsing SNMP")
	}
	output := types.SNMP{}
	fillStruct(&output, snmpData)

	return output, nil
}

// getNetstatStats pulls netstat stats from /proc/net
func getNetstatStats(raw []byte) (types.Netstat, error) {
	netstatData, err := parseNetFile(string(raw))
	if err != nil {
		return types.Netstat{}, errors.Wrap(err, "error parsing netstat")
	}
	output := types.Netstat{}
	fillStruct(&output, netstatData)
	return output, nil
}
