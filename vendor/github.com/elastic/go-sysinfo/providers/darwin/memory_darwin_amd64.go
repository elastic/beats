// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build darwin,amd64,cgo

package darwin

import (
	"github.com/pkg/errors"
)

const hwMemsizeMIB = "hw.memsize"

func MemTotal() (uint64, error) {
	var size uint64
	if err := sysctlByName(hwMemsizeMIB, &size); err != nil {
		return 0, errors.Wrap(err, "failed to get mem total")
	}

	return size, nil
}
