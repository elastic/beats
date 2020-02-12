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

package windows

import (
	"time"

	windows "github.com/elastic/go-windows"
	"github.com/pkg/errors"
)

func BootTime() (time.Time, error) {
	msSinceBoot, err := windows.GetTickCount64()
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to get boot time")
	}

	// According to GetTickCount64 the resolution is limited to between 10 to 16
	// milliseconds so truncate the time as to not mislead anyone about the
	// resolution.
	bootTime := time.Now().Add(-1 * time.Duration(msSinceBoot) * time.Millisecond)
	bootTime = bootTime.Truncate(10 * time.Millisecond)
	return bootTime, nil
}
