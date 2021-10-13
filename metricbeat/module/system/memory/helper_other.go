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

//go:build darwin || freebsd || aix || openbsd || windows
// +build darwin freebsd aix openbsd windows

package memory

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/common"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// These whole helper files are a shim until we can make breaking changes and remove these
// data enrichers from the metricset, as they're linux-only.
// DEPRECATE: 8.0
func fetchLinuxMemStats(baseMap common.MapStr) error {
	return errors.New("MemStats is only available on Linux")
}

func getVMStat() (*sysinfotypes.VMStatInfo, error) {
	return nil, errors.New("VMStat is only available on Linux")
}
