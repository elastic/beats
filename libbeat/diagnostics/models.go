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
	"context"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/docker"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Diagnostics struct {
	Metrics    Metrics   `json:"metrics"`
	Host       Host      `json:"host"`
	Docker     Docker    `json:"docker"`
	DiagStart  time.Time `json:"started_at"`
	Beat       Beat      `json:"beat"`
	DiagFolder string    `json:"diagnostics_folder"`
	Logger     *logp.Logger
	Context    context.Context
}

type Docker struct {
	IsContainer bool                      `json:"is_container"`
	Timestamp   time.Time                 `json:"timestamp"`
	Status      []docker.CgroupDockerStat `json:"status"`
	Memory      *docker.CgroupMemStat     `json:"memory"`
	CPUStats    *docker.CgroupCPUStat     `json:"cpu"`
}

type Host struct {
	Info    *host.InfoStat `json:"info"`
	CPUInfo []cpu.InfoStat `json:"cpu_info"`
}

type Beat struct {
	Info       beat.Info `json:"info"`
	ConfigPath string    `json:"config_path"`
	LogPath    string    `json:"log_path"`
	ModulePath string    `json:"module_path"`
}

type Metrics struct {
	Timestamp    time.Time              `json:"timestamp"`
	Swap         *mem.SwapMemoryStat    `json:"swap"`
	Memory       *mem.VirtualMemoryStat `json:"memory"`
	NumGoroutine int                    `json:"go_routines"`
	CPUStats     []cpu.TimesStat        `json:"cpu"`

	AvgLoad *load.AvgStat `json:"cpu_avg_load"`
	Network Network       `json:"network"`
	Disk    Disk          `json:"disk"`
}

type Network struct {
	Stats []net.ConntrackStat  `json:"stats"`
	IO    []net.IOCountersStat `json:"io"`
}

type Disk struct {
	Stats map[string]disk.IOCountersStat `json:"stats"`
}
