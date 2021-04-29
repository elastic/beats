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

package diagnostics

import (
	"encoding/json"
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
)

func getHostInfo(diag *Diagnostics) {
	h, err := host.InfoWithContext(diag.Context)
	if err != nil {
		fmt.Println("failed to get host info")
	}
	diag.Host.Info = h
	cm, err := cpu.InfoWithContext(diag.Context)
	if err != nil {
		diag.Logger.Error("Unable to find CPU info")

	}
	diag.Host.CPUInfo = cm
	hjson, err := json.Marshal(diag.Host)
	if err != nil {
		fmt.Println("failed to unmarshal host info")
		fmt.Println(err)
	}
	writeToFile(diag.DiagFolder, "host.json", hjson)
}
