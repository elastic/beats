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

package file

import (
	"os"
	"reflect"
	"strconv"
)

type StateOS struct {
	IdxHi uint64 `json:"idxhi,"`
	IdxLo uint64 `json:"idxlo,"`
	Vol   uint64 `json:"vol,"`
}

// GetOSState returns the platform specific StateOS
func GetOSState(info os.FileInfo) StateOS {
	// os.SameFile must be called to populate the id fields. Otherwise in case for example
	// os.Stat(file) is used to get the fileInfo, the ids are empty.
	// https://github.com/elastic/beats/filebeat/pull/53
	os.SameFile(info, info)

	// Gathering fileStat (which is fileInfo) through reflection as otherwise not accessible
	// See https://github.com/golang/go/blob/90c668d1afcb9a17ab9810bce9578eebade4db56/src/os/stat_windows.go#L33
	fileStat := reflect.ValueOf(info).Elem()

	// Get the three fields required to uniquely identify file und windows
	// More details can be found here: https://msdn.microsoft.com/en-us/library/aa363788(v=vs.85).aspx
	// Uint should already return uint64, but making sure this is the case
	// The required fields can be found here: https://github.com/golang/go/blob/master/src/os/types_windows.go#L78
	fileState := StateOS{
		IdxHi: uint64(fileStat.FieldByName("idxhi").Uint()),
		IdxLo: uint64(fileStat.FieldByName("idxlo").Uint()),
		Vol:   uint64(fileStat.FieldByName("vol").Uint()),
	}

	return fileState
}

// IsSame file checks if the files are identical
func (fs StateOS) IsSame(state StateOS) bool {
	return fs.IdxHi == state.IdxHi && fs.IdxLo == state.IdxLo && fs.Vol == state.Vol
}

func (fs StateOS) String() string {
	var buf [92]byte
	current := strconv.AppendUint(buf[:0], fs.IdxHi, 10)
	current = append(current, '-')
	current = strconv.AppendUint(current, fs.IdxLo, 10)
	current = append(current, '-')
	current = strconv.AppendUint(current, fs.Vol, 10)
	return string(current)
}
