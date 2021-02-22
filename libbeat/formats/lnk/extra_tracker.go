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

package lnk

import (
	"encoding/binary"
	"errors"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

func parseExtraTracker(size uint32, data []byte) (*Tracker, error) {
	if size != 0x00000060 {
		return nil, errors.New("invalid extra tracker block size")
	}
	return &Tracker{
		Version:   binary.LittleEndian.Uint32(data[12:16]),
		MachineID: common.ReadString(data[16:32], 0),
		Droid: []string{
			encodeUUID(data[32:48]),
			encodeUUID(data[48:64]),
		},
		DroidBirth: []string{
			encodeUUID(data[64:80]),
			encodeUUID(data[80:96]),
		},
	}, nil
}
