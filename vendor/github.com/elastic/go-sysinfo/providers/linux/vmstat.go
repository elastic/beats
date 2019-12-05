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
	"reflect"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

// vmstatTagToFieldIndex contains a mapping of json struct tags to struct field indices.
var vmstatTagToFieldIndex = make(map[string]int)

func init() {
	var vmstat types.VMStatInfo
	val := reflect.ValueOf(vmstat)
	typ := reflect.TypeOf(vmstat)

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if tag := field.Tag.Get("json"); tag != "" {
			vmstatTagToFieldIndex[tag] = i
		}
	}
}

// parseVMStat parses the contents of /proc/vmstat.
func parseVMStat(content []byte) (*types.VMStatInfo, error) {
	var vmStat types.VMStatInfo
	refValues := reflect.ValueOf(&vmStat).Elem()

	err := parseKeyValue(content, " ", func(key, value []byte) error {
		// turn our []byte value into an int
		val, err := parseBytesOrNumber(value)
		if err != nil {
			return errors.Wrapf(err, "failed to parse %v value of %v", string(key), string(value))
		}

		idx, ok := vmstatTagToFieldIndex[string(key)]
		if !ok {
			return nil
		}

		sval := refValues.Field(idx)

		if sval.CanSet() {
			sval.SetUint(val)
		}
		return nil
	})

	return &vmStat, err
}
