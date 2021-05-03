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
	"net/http"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/docker"
	"github.com/shirou/gopsutil/v3/net"
)

type Diagnostics struct {
	Metrics    Metrics   `json:"metrics"`
	Docker     Docker    `json:"docker"`
	DiagStart  time.Time `json:"started_at"`
	Beat       Beat      `json:"beat"`
	DiagFolder string    `json:"diagnostics_folder"`
	Manifest   Manifest  `json:"manifest"`
	Interval   string
	Duration   string
	Type       string
	HTTP       HTTP
	Context    context.Context
	CancelFunc context.CancelFunc
}

type Manifest struct {
	Version string `json:"version"`
	Command string `json:"command"`
}

type HTTP struct {
	Protocol string
	Host     string
	Client   *http.Client
}

type Docker struct {
	IsContainer bool                      `json:"is_container"`
	Timestamp   time.Time                 `json:"timestamp"`
	Status      []docker.CgroupDockerStat `json:"status"`
	Memory      *docker.CgroupMemStat     `json:"memory"`
	CPUStats    *docker.CgroupCPUStat     `json:"cpu"`
}

type Beat struct {
	Info       beat.Info `json:"info"`
	State      State     `json:"state"`
	ConfigPath string    `json:"config_path"`
	LogPath    string    `json:"log_path"`
	ModulePath string    `json:"module_path"`
}

type Network struct {
	Stats []net.ConntrackStat  `json:"stats"`
	IO    []net.IOCountersStat `json:"io"`
}

type Disk struct {
	Stats map[string]disk.IOCountersStat `json:"stats"`
}

// TODO, this struct might already exist somewhere inside beat, need to double check
type State struct {
	Beat struct {
		Name string `json:"name"`
	} `json:"beat"`
	Host struct {
		Architecture  string `json:"architecture"`
		Containerized string `json:"containerized"`
		Hostname      string `json:"hostname"`
		ID            string `json:"id"`
		Os            struct {
			Codename string `json:"codename"`
			Family   string `json:"family"`
			Kernel   string `json:"kernel"`
			Name     string `json:"name"`
			Platform string `json:"platform"`
			Version  string `json:"version"`
		} `json:"os"`
	} `json:"host"`
	Input struct {
		Count int           `json:"count"`
		Names []interface{} `json:"names"`
	} `json:"input"`
	Management struct {
		Enabled bool `json:"enabled"`
	} `json:"management"`
	Module struct {
		Count int           `json:"count"`
		Names []interface{} `json:"names"`
	} `json:"module"`
	Output struct {
		BatchSize int    `json:"batch_size"`
		Clients   int    `json:"clients"`
		Name      string `json:"name"`
	} `json:"output"`
	Outputs struct {
		Elasticsearch struct {
			ClusterUUID string `json:"cluster_uuid"`
		} `json:"elasticsearch"`
	} `json:"outputs"`
	Queue struct {
		Name string `json:"name"`
	} `json:"queue"`
	Service struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"service"`
}

// TODO, this struct might already exist somewhere inside beat, need to double check
type Metrics struct {
	Timestamp time.Time `json:"timestamp"`
	Beat      struct {
		CPU struct {
			System struct {
				Ticks int `json:"ticks"`
				Time  struct {
					Ms int `json:"ms"`
				} `json:"time"`
			} `json:"system"`
			Total struct {
				Ticks int `json:"ticks"`
				Time  struct {
					Ms int `json:"ms"`
				} `json:"time"`
				Value int `json:"value"`
			} `json:"total"`
			User struct {
				Ticks int `json:"ticks"`
				Time  struct {
					Ms int `json:"ms"`
				} `json:"time"`
			} `json:"user"`
		} `json:"cpu"`
		Handles struct {
			Limit struct {
				Hard int `json:"hard"`
				Soft int `json:"soft"`
			} `json:"limit"`
			Open int `json:"open"`
		} `json:"handles"`
		Info struct {
			EphemeralID string `json:"ephemeral_id"`
			Uptime      struct {
				Ms int `json:"ms"`
			} `json:"uptime"`
		} `json:"info"`
		Memstats struct {
			GcNext      int `json:"gc_next"`
			MemoryAlloc int `json:"memory_alloc"`
			MemorySys   int `json:"memory_sys"`
			MemoryTotal int `json:"memory_total"`
			Rss         int `json:"rss"`
		} `json:"memstats"`
		Runtime struct {
			Goroutines int `json:"goroutines"`
		} `json:"runtime"`
	} `json:"beat"`
	Filebeat struct {
		Events struct {
			Active int `json:"active"`
			Added  int `json:"added"`
			Done   int `json:"done"`
		} `json:"events"`
		Harvester struct {
			Closed    int `json:"closed"`
			OpenFiles int `json:"open_files"`
			Running   int `json:"running"`
			Skipped   int `json:"skipped"`
			Started   int `json:"started"`
		} `json:"harvester"`
		Input struct {
			Log struct {
				Files struct {
					Renamed   int `json:"renamed"`
					Truncated int `json:"truncated"`
				} `json:"files"`
			} `json:"log"`
		} `json:"input"`
	} `json:"filebeat"`
	Libbeat struct {
		Config struct {
			Module struct {
				Running int `json:"running"`
				Starts  int `json:"starts"`
				Stops   int `json:"stops"`
			} `json:"module"`
			Reloads int `json:"reloads"`
			Scans   int `json:"scans"`
		} `json:"config"`
		Output struct {
			Events struct {
				Acked      int `json:"acked"`
				Active     int `json:"active"`
				Batches    int `json:"batches"`
				Dropped    int `json:"dropped"`
				Duplicates int `json:"duplicates"`
				Failed     int `json:"failed"`
				Toomany    int `json:"toomany"`
				Total      int `json:"total"`
			} `json:"events"`
			Read struct {
				Bytes  int `json:"bytes"`
				Errors int `json:"errors"`
			} `json:"read"`
			Type  string `json:"type"`
			Write struct {
				Bytes  int `json:"bytes"`
				Errors int `json:"errors"`
			} `json:"write"`
		} `json:"output"`
		Pipeline struct {
			Clients int `json:"clients"`
			Events  struct {
				Active    int `json:"active"`
				Dropped   int `json:"dropped"`
				Failed    int `json:"failed"`
				Filtered  int `json:"filtered"`
				Published int `json:"published"`
				Retry     int `json:"retry"`
				Total     int `json:"total"`
			} `json:"events"`
			Queue struct {
				Acked     int `json:"acked"`
				MaxEvents int `json:"max_events"`
			} `json:"queue"`
		} `json:"pipeline"`
	} `json:"libbeat"`
	Registrar struct {
		States struct {
			Cleanup int `json:"cleanup"`
			Current int `json:"current"`
			Update  int `json:"update"`
		} `json:"states"`
		Writes struct {
			Fail    int `json:"fail"`
			Success int `json:"success"`
			Total   int `json:"total"`
		} `json:"writes"`
	} `json:"registrar"`
	System struct {
		CPU struct {
			Cores int `json:"cores"`
		} `json:"cpu"`
		Load struct {
			Num1  float64 `json:"1"`
			Num5  float64 `json:"5"`
			Num15 float64 `json:"15"`
			Norm  struct {
				Num1  float64 `json:"1"`
				Num5  float64 `json:"5"`
				Num15 float64 `json:"15"`
			} `json:"norm"`
		} `json:"load"`
	} `json:"system"`
}
