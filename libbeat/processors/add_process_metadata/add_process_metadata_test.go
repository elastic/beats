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

package add_process_metadata

import (
	"math"
	"os"
	"testing"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestAddProcessMetadata(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(processorName))
	startTime := time.Now()
	testProcs := testProvider{
		1: {
			name:  "systemd",
			title: "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
			exe:   "/usr/lib/systemd/systemd",
			args:  []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
			env: map[string]string{
				"HOME":       "/",
				"TERM":       "linux",
				"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
				"LANG":       "en_US.UTF-8",
			},
			pid:       1,
			ppid:      0,
			startTime: startTime,
			username:  "root",
			userid:    "0",
		},
		3: {
			name:  "systemd",
			title: "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
			exe:   "/usr/lib/systemd/systemd",
			args:  []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
			env: map[string]string{
				"HOME":       "/",
				"TERM":       "linux",
				"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
				"LANG":       "en_US.UTF-8",
			},
			pid:       1,
			ppid:      0,
			startTime: startTime,
			username:  "user",
			userid:    "1001",
		},
	}

	// mock of the cgroup processCgroupPaths
	processCgroupPaths = func(_ resolve.Resolver, pid int) (cgroup.PathList, error) {
		testMap := map[int]cgroup.PathList{
			1: {
				V1: map[string]cgroup.ControllerPath{

					"cpu":          {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"net_prio":     {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"blkio":        {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"perf_event":   {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"freezer":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"pids":         {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"hugetlb":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"cpuacct":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"cpuset":       {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"net_cls":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"devices":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"memory":       {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"name=systemd": {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				},
			},
			2: {
				V1: map[string]cgroup.ControllerPath{

					"cpu":          {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"net_prio":     {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"blkio":        {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"perf_event":   {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"freezer":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"pids":         {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"hugetlb":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"cpuacct":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"cpuset":       {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"net_cls":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"devices":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"memory":       {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
					"name=systemd": {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				},
			},
		}

		return testMap[pid], nil
	}

	for _, test := range []struct {
		description             string
		config, event, expected mapstr.M
		err, initErr            error
	}{
		{
			description: "default fields",
			config: mapstr.M{
				"match_pids": []string{"system.process.ppid"},
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
				"process": mapstr.M{
					"name":       "systemd",
					"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
					"executable": "/usr/lib/systemd/systemd",
					"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
					"pid":        1,
					"parent": mapstr.M{
						"pid": 0,
					},
					"start_time": startTime,
					"owner": mapstr.M{
						"name": "root",
						"id":   "0",
					},
				},
				"container": mapstr.M{
					"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
				},
			},
		},
		{
			description: "single field",
			config: mapstr.M{
				"match_pids":     []string{"system.process.ppid"},
				"target":         "system.process.parent",
				"include_fields": []string{"process.name"},
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
						"parent": mapstr.M{
							"name": "systemd",
						},
					},
				},
			},
		},
		{
			description: "multiple fields",
			config: mapstr.M{
				"match_pids":     []string{"system.other.pid", "system.process.ppid"},
				"target":         "extra",
				"include_fields": []string{"process.title", "process.start_time"},
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
				"extra": mapstr.M{
					"process": mapstr.M{
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"start_time": startTime,
					},
				},
			},
		},
		{
			description: "complete process info",
			config: mapstr.M{
				"match_pids": []string{"ppid"},
				"target":     "parent",
			},
			event: mapstr.M{
				"ppid": "1",
			},
			expected: mapstr.M{
				"ppid": "1",
				"parent": mapstr.M{
					"process": mapstr.M{
						"name":       "systemd",
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"executable": "/usr/lib/systemd/systemd",
						"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
						"pid":        1,
						"parent": mapstr.M{
							"pid": 0,
						},
						"start_time": startTime,
						"owner": mapstr.M{
							"name": "root",
							"id":   "0",
						},
					},
					"container": mapstr.M{
						"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
					},
				},
			},
		},
		{
			description: "complete process info (restricted fields)",
			config: mapstr.M{
				"match_pids":        []string{"ppid"},
				"restricted_fields": true,
				"target":            "parent",
			},
			event: mapstr.M{
				"ppid": "1",
			},
			expected: mapstr.M{
				"ppid": "1",
				"parent": mapstr.M{
					"process": mapstr.M{
						"name":       "systemd",
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"executable": "/usr/lib/systemd/systemd",
						"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
						"pid":        1,
						"parent": mapstr.M{
							"pid": 0,
						},
						"start_time": startTime,
						"env": map[string]string{
							"HOME":       "/",
							"TERM":       "linux",
							"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
							"LANG":       "en_US.UTF-8",
						},
						"owner": mapstr.M{
							"name": "root",
							"id":   "0",
						},
					},
					"container": mapstr.M{
						"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
					},
				},
			},
		},
		{
			description: "complete process info (restricted fields - alt)",
			config: mapstr.M{
				"match_pids":        []string{"ppid"},
				"restricted_fields": true,
				"target":            "parent",
				"include_fields":    []string{"process"},
			},
			event: mapstr.M{
				"ppid": "1",
			},
			expected: mapstr.M{
				"ppid": "1",
				"parent": mapstr.M{
					"process": mapstr.M{
						"name":       "systemd",
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"executable": "/usr/lib/systemd/systemd",
						"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
						"pid":        1,
						"parent": mapstr.M{
							"pid": 0,
						},
						"start_time": startTime,
						"env": map[string]string{
							"HOME":       "/",
							"TERM":       "linux",
							"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
							"LANG":       "en_US.UTF-8",
						},
						"owner": mapstr.M{
							"name": "root",
							"id":   "0",
						},
					},
				},
			},
		},
		{
			description: "env field (restricted_fields: true)",
			config: mapstr.M{
				"match_pids":        []string{"ppid"},
				"restricted_fields": true,
				"target":            "parent",
				"include_fields":    []string{"process.env"},
			},
			event: mapstr.M{
				"ppid": "1",
			},
			expected: mapstr.M{
				"ppid": "1",
				"parent": mapstr.M{
					"env": map[string]string{
						"HOME":       "/",
						"TERM":       "linux",
						"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
						"LANG":       "en_US.UTF-8",
					},
				},
			},
		},
		{
			description: "env field (restricted_fields: false)",
			config: mapstr.M{
				"match_pids":     []string{"ppid"},
				"target":         "parent",
				"include_fields": []string{"process.env"},
			},
			event: mapstr.M{
				"ppid": "1",
			},
			expected: nil,
			initErr:  errors.New("error unpacking add_process_metadata.target_fields: field 'process.env' not found"),
		},
		{
			description: "fields not found (ignored)",
			config: mapstr.M{
				"match_pids": []string{"ppid"},
			},
			event: mapstr.M{
				"other": "field",
			},
			expected: mapstr.M{
				"other": "field",
			},
		},
		{
			description: "fields not found (reported)",
			config: mapstr.M{
				"match_pids":     []string{"ppid"},
				"ignore_missing": false,
			},
			event: mapstr.M{
				"other": "field",
			},
			expected: mapstr.M{
				"other": "field",
			},
			err: ErrNoMatch,
		},
		{
			description: "overwrite keys",
			config: mapstr.M{
				"overwrite_keys": true,
				"match_pids":     []string{"ppid"},
				"include_fields": []string{"process.name"},
			},
			event: mapstr.M{
				"ppid": 1,
				"process": mapstr.M{
					"name": "other",
				},
			},
			expected: mapstr.M{
				"ppid": 1,
				"process": mapstr.M{
					"name": "systemd",
				},
			},
		},
		{
			description: "overwrite keys error",
			config: mapstr.M{
				"match_pids":     []string{"ppid"},
				"include_fields": []string{"process.name"},
			},
			event: mapstr.M{
				"ppid": 1,
				"process": mapstr.M{
					"name": "other",
				},
			},
			expected: mapstr.M{
				"ppid": 1,
				"process": mapstr.M{
					"name": "other",
				},
			},
			err: errors.New("error applying add_process_metadata processor: target field 'process.name' already exists and overwrite_keys is false"),
		},
		{
			description: "bad PID field cast",
			config: mapstr.M{
				"match_pids": []string{"ppid"},
			},
			event: mapstr.M{
				"ppid": "a",
			},
			expected: mapstr.M{
				"ppid": "a",
			},
			err: errors.New("error applying add_process_metadata processor: cannot parse pid field 'ppid': error converting string to integer: strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		{
			description: "bad PID field type",
			config: mapstr.M{
				"match_pids": []string{"ppid"},
			},
			event: mapstr.M{
				"ppid": false,
			},
			expected: mapstr.M{
				"ppid": false,
			},
			err: errors.New("error applying add_process_metadata processor: cannot parse pid field 'ppid': not an integer or string, but bool"),
		},
		{
			description: "process not found",
			config: mapstr.M{
				"match_pids": []string{"ppid"},
			},
			event: mapstr.M{
				"ppid": 42,
			},
			expected: mapstr.M{
				"ppid": 42,
			},
			err: ErrNoProcess,
		},
		{
			description: "lookup first PID",
			config: mapstr.M{
				"match_pids": []string{"nil", "ppid"},
			},
			event: mapstr.M{
				"nil":  0,
				"ppid": 1,
			},
			expected: mapstr.M{
				"nil":  0,
				"ppid": 1,
			},
			err: ErrNoProcess,
		},
		{
			description: "env field",
			config: mapstr.M{
				"match_pids": []string{"system.process.ppid"},
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
				"process": mapstr.M{
					"name":       "systemd",
					"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
					"executable": "/usr/lib/systemd/systemd",
					"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
					"pid":        1,
					"parent": mapstr.M{
						"pid": 0,
					},
					"start_time": startTime,
					"owner": mapstr.M{
						"name": "root",
						"id":   "0",
					},
				},
				"container": mapstr.M{
					"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
				},
			},
		},
		{
			description: "env field (IncludeContainer id), process not found",
			config: mapstr.M{
				"match_pids": []string{"ppid"},
			},
			event: mapstr.M{
				"ppid": 42,
			},
			expected: mapstr.M{
				"ppid": 42,
			},
			err: ErrNoProcess,
		},
		{
			description: "container.id only",
			config: mapstr.M{
				"match_pids":     []string{"system.process.ppid"},
				"include_fields": []string{"container.id"},
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
				"container": mapstr.M{
					"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
				},
			},
		},
		{
			description: "container.id based on regex in config",
			config: mapstr.M{
				"match_pids":     []string{"system.process.ppid"},
				"include_fields": []string{"container.id"},
				"cgroup_regex":   "\\/.+\\/.+\\/.+\\/([0-9a-f]{64}).*",
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
				"container": mapstr.M{
					"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
				},
			},
		},
		{
			description: "no process metadata available",
			config: mapstr.M{
				"match_pids":   []string{"system.process.ppid"},
				"cgroup_regex": "\\/.+\\/.+\\/.+\\/([0-9a-f]{64}).*",
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "2",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "2",
					},
				},
				"container": mapstr.M{
					"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
				},
			},
		},
		{
			description: "no container id available",
			config: mapstr.M{
				"match_pids":   []string{"system.process.ppid"},
				"cgroup_regex": "\\/.+\\/.+\\/.+\\/([0-9a-f]{64}).*",
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "3",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "3",
					},
				},
				"process": mapstr.M{
					"name":       "systemd",
					"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
					"executable": "/usr/lib/systemd/systemd",
					"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
					"pid":        1,
					"parent": mapstr.M{
						"pid": 0,
					},
					"start_time": startTime,
					"owner": mapstr.M{
						"name": "user",
						"id":   "1001",
					},
				},
			},
		},
		{
			description: "without cgroup cache",
			config: mapstr.M{
				"match_pids":               []string{"system.process.ppid"},
				"include_fields":           []string{"container.id"},
				"cgroup_cache_expire_time": 0,
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
				"container": mapstr.M{
					"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
				},
			},
		},
		{
			description: "custom cache expire time",
			config: mapstr.M{
				"match_pids":               []string{"system.process.ppid"},
				"include_fields":           []string{"container.id"},
				"cgroup_cache_expire_time": 10 * time.Second,
			},
			event: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
			},
			expected: mapstr.M{
				"system": mapstr.M{
					"process": mapstr.M{
						"ppid": "1",
					},
				},
				"container": mapstr.M{
					"id": "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1",
				},
			},
		},
		{
			description: "only user",
			config: mapstr.M{
				"match_pids":     []string{"ppid"},
				"target":         "",
				"include_fields": []string{"process.owner"},
			},
			event: mapstr.M{
				"ppid": "1",
			},
			expected: mapstr.M{
				"ppid": "1",
				"process": mapstr.M{
					"owner": mapstr.M{
						"id":   "0",
						"name": "root",
					},
				},
			},
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			config, err := conf.NewConfigFrom(test.config)
			if err != nil {
				t.Fatal(err)
			}

			proc, err := newProcessMetadataProcessorWithProvider(config, testProcs, true)
			if test.initErr == nil {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assert.EqualError(t, err, test.initErr.Error())
				return
			}
			t.Log(proc.String())
			ev := beat.Event{
				Fields: test.event,
			}
			result, err := proc.Run(&ev)
			if test.err == nil {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assert.EqualError(t, err, test.err.Error())
			}
			if test.expected != nil {
				assert.Equal(t, test.expected, result.Fields)
			} else {
				assert.Nil(t, result)
			}
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		c := mapstr.M{
			"match_pids":     []string{"@metadata.system.ppid"},
			"target":         "@metadata",
			"include_fields": []string{"process.name"},
		}

		config, err := conf.NewConfigFrom(c)
		assert.NoError(t, err)

		proc, err := newProcessMetadataProcessorWithProvider(config, testProcs, true)
		assert.NoError(t, err)

		event := &beat.Event{
			Meta: mapstr.M{
				"system": mapstr.M{
					"ppid": "1",
				},
			},
			Fields: mapstr.M{},
		}
		expMeta := mapstr.M{
			"system": mapstr.M{
				"ppid": "1",
			},
			"process": mapstr.M{
				"name": "systemd",
			},
		}

		result, err := proc.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, result.Meta)
		assert.Equal(t, event.Fields, result.Fields)
	})
}

func TestUsingCache(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(processorName))

	selfPID := os.Getpid()

	// mock of the cgroup processCgroupPaths
	processCgroupPaths = func(_ resolve.Resolver, pid int) (cgroup.PathList, error) {
		testStruct := cgroup.PathList{
			V1: map[string]cgroup.ControllerPath{

				"cpu":          {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"net_prio":     {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"blkio":        {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"perf_event":   {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"freezer":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"pids":         {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"hugetlb":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"cpuacct":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"cpuset":       {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"net_cls":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"devices":      {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"memory":       {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
				"name=systemd": {ControllerPath: "/kubepods/besteffort/pod665fb997-575b-11ea-bfce-080027421ddf/b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1"},
			},
		}
		testMap := map[int]cgroup.PathList{
			selfPID: testStruct,
		}

		// testMap :=
		return testMap[pid], nil
	}

	config, err := conf.NewConfigFrom(mapstr.M{
		"match_pids":     []string{"system.process.ppid"},
		"include_fields": []string{"container.id"},
		"target":         "meta",
	})
	if err != nil {
		t.Fatal(err)
	}
	proc, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ev := beat.Event{
		Fields: mapstr.M{
			"system": mapstr.M{
				"process": mapstr.M{
					"ppid": selfPID,
				},
			},
		},
	}

	// first run
	result, err := proc.Run(&ev)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result.Fields)
	containerID, err := result.Fields.GetValue("meta.container.id")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1", containerID)

	ev = beat.Event{
		Fields: mapstr.M{
			"system": mapstr.M{
				"process": mapstr.M{
					"ppid": selfPID,
				},
			},
		},
	}

	// cached result
	result, err = proc.Run(&ev)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result.Fields)
	containerID, err = result.Fields.GetValue("meta.container.id")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "b5285682fba7449c86452b89a800609440ecc88a7ba5f2d38bedfb85409b30b1", containerID)
}

func TestSelf(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(processorName))
	config, err := conf.NewConfigFrom(mapstr.M{
		"match_pids": []string{"self_pid"},
		"target":     "self",
	})
	if err != nil {
		t.Fatal(err)
	}
	proc, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	selfPID := os.Getpid()
	ev := beat.Event{
		Fields: mapstr.M{
			"self_pid": selfPID,
		},
	}
	result, err := proc.Run(&ev)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result.Fields)
	pidField, err := result.Fields.GetValue("self.process.pid")
	if err != nil {
		t.Fatal(err)
	}
	pid, ok := pidField.(int)
	assert.True(t, ok)
	assert.Equal(t, selfPID, pid)
}

func TestBadProcess(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(processorName))
	config, err := conf.NewConfigFrom(mapstr.M{
		"match_pids": []string{"self_pid"},
		"target":     "self",
	})
	if err != nil {
		t.Fatal(err)
	}
	proc, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	ev := beat.Event{
		Fields: mapstr.M{
			"self_pid": 0,
		},
	}
	result, err := proc.Run(&ev)
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Fields)
	assert.Equal(t, ev.Fields, result.Fields)
}

func TestPIDToInt(t *testing.T) {
	const intIs64bit = unsafe.Sizeof(int(0)) == unsafe.Sizeof(int64(0))
	for _, test := range []struct {
		name string
		pid  interface{}
		fail bool
	}{
		{
			name: "numeric string",
			pid:  "1234",
		},
		{
			name: "numeric string ignore octal",
			pid:  "008",
		},
		{
			name: "numeric string invalid hex",
			pid:  "0x10",
			fail: true,
		},
		{
			name: "non-numeric string",
			pid:  "abcd",
			fail: true,
		},
		{
			name: "int",
			pid:  0,
		},
		{
			name: "int min",
			pid:  math.MaxInt32,
		},
		{
			name: "int max",
			pid:  math.MaxInt32,
		},
		{
			name: "uint min",
			pid:  uint(0),
		},
		{
			name: "uint max",
			pid:  uint(math.MaxUint32),
			fail: !intIs64bit,
		},
		{
			name: "int8",
			pid:  int8(0),
		},
		{
			name: "int8 min",
			pid:  int8(math.MinInt8),
		},
		{
			name: "int8 max",
			pid:  int8(math.MaxInt8),
		},
		{
			name: "uint8 min",
			pid:  uint8(0),
		},
		{
			name: "uint8 max",
			pid:  uint8(math.MaxUint8),
		},
		{
			name: "int16",
			pid:  int16(0),
		},
		{
			name: "int16 min",
			pid:  int16(math.MinInt16),
		},
		{
			name: "int16 max",
			pid:  int16(math.MaxInt16),
		},
		{
			name: "uint16 min",
			pid:  uint16(0),
		},
		{
			name: "uint16 max",
			pid:  uint16(math.MaxUint16),
		},
		{
			name: "int32",
			pid:  int32(0),
		},
		{
			name: "int32 min",
			pid:  int32(math.MinInt32),
		},
		{
			name: "int32 max",
			pid:  int32(math.MaxInt32),
		},
		{
			name: "uint32 min",
			pid:  uint32(0),
		},
		{
			name: "uint32 max",
			pid:  uint32(math.MaxUint32),
			fail: !intIs64bit,
		},
		{
			name: "int64",
			pid:  int64(0),
			fail: false,
		},
		{
			name: "int64 min",
			pid:  int64(math.MinInt64),
			fail: !intIs64bit,
		},
		{
			name: "int64 max",
			pid:  int64(math.MaxInt64),
			fail: !intIs64bit,
		},
		{
			name: "uint64 min",
			pid:  uint64(0),
			fail: false,
		},
		{
			name: "uint64 max",
			pid:  uint64(math.MaxUint64),
			fail: true,
		},
		{
			name: "uintptr",
			pid:  uintptr(0),
			fail: false,
		},
		{
			name: "boolean",
			pid:  false,
			fail: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := pidToInt(test.pid)
			if test.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestV2CID(t *testing.T) {
	processCgroupPaths = func(_ resolve.Resolver, _ int) (cgroup.PathList, error) {
		testMap := cgroup.PathList{
			V1: map[string]cgroup.ControllerPath{
				"cpu": {IsV2: true, ControllerPath: "system.slice/docker-2dcbab615aebfa9313feffc5cfdacd381543cfa04c6be3f39ac656e55ef34805.scope"},
			},
		}
		return testMap, nil
	}
	provider := newCidProvider(resolve.NewTestResolver(""), []string{}, "", processCgroupPaths, nil)
	result, err := provider.GetCid(1)
	assert.NoError(t, err)
	assert.Equal(t, "2dcbab615aebfa9313feffc5cfdacd381543cfa04c6be3f39ac656e55ef34805", result)
}
