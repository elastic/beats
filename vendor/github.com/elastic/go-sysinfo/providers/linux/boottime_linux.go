// Copyright 2018 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package linux

import (
	"sync"
	"time"

	"github.com/prometheus/procfs"
)

var (
	bootTime     time.Time
	bootTimeLock sync.Mutex
)

func BootTime() (time.Time, error) {
	bootTimeLock.Lock()
	defer bootTimeLock.Unlock()

	if !bootTime.IsZero() {
		return bootTime, nil
	}

	stat, err := procfs.NewStat()
	if err != nil {
		return time.Time{}, nil
	}

	bootTime = time.Unix(int64(stat.BootTime), 0)
	return bootTime, nil
}
