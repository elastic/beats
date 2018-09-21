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

package pool

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type phpFpmStatus struct {
	Name                  string        	`json:"pool"`
	ProcessManager        string          	`json:"process manager"`
	SlowRequests          int        		`json:"slow requests"`
	StartTime   	  	  int          		`json:"start time"`
	StartSince            int        		`json:"start since"`
	AcceptedConnection    int        		`json:"accepted conn"`
	ListenQueueLen        int          		`json:"listen queue len"`
	MaxListenQueue        int        		`json:"max listen queue"`
	Queued        		  int          		`json:"listen queue"`  
	ActiveProcesses       int         	    `json:"active processes"`
	IdleProcesses         int   			`json:"idle processes"`
	MaxActiveProcesses    int         	    `json:"max active processes"`
	MaxChildrenReached    int        		`json:"max children reached"`
	TotalProcesses        int         	    `json:"total processes"`
	Processes             []phpFpmProcess   `json:"processes"`
	
}

type phpFpmProcess struct {
	PID           		int         `json:"pid"`
	State         		string   	`json:"state"`
	StartTime       	int         `json:"start time"`
	StartSince        	int        	`json:"start since"`
	Requests           	int         `json:"requests"`
	RequestDuration   	int         `json:"request duration"`
	RequestMethod       string      `json:"request method"`
	RequestURI          string      `json:"request uri"`
	ContentLength       int         `json:"content length"`
	User           		string      `json:"user"`
	Script           	string      `json:"script"`
	LastRequestCPU      float64     `json:"last request cpu"`
	LastRequestMemory	int         `json:"last request memory"`
}

func eventsMapping(content []byte) (common.MapStr, error) {
	var status phpFpmStatus
	err := json.Unmarshal(content, &status)
	if err != nil {
		logp.Err("php-fpm status parsing failed with error: ", err)
		return nil, err
	}
	//remapping process details to match the naming format
	var mapProcesses []common.MapStr
	for _, process := range status.Processes {
		proc := common.MapStr {
			"pid":                 process.PID,
			"state":               process.State,
			"start_time":          process.StartTime,
			"start_since":         process.StartSince,
			"requests":            process.Requests,
			"request_duration":    process.RequestDuration,
			"request_method":      process.RequestMethod,
			"request_uri":         process.RequestURI,
			"content_length":      process.ContentLength,
			"user":                process.User,
			"script":              process.Script,
			"last_request_cpu":    process.LastRequestCPU,
			"last_request_memory": process.LastRequestMemory,
		}
		mapProcesses = append(mapProcesses, proc)
	}
	baseEvent := common.MapStr{
			"name":              status.Name,
			"process_manager":   status.ProcessManager,
			"slow_requests":     status.SlowRequests,
			"start_time":        status.StartTime,
			"start_since":       status.StartSince,
			"connections":       common.MapStr{
				"accepted":          status.AcceptedConnection,
				"listen_queue_len":  status.ListenQueueLen,
				"max_listen_queue":  status.MaxListenQueue,
				"queued":            status.Queued,
			},
			"processes":         common.MapStr { 
				"active":            	status.ActiveProcesses,
				"idle":					status.IdleProcesses,
				"max_active":			status.MaxActiveProcesses,
				"max_children_reached":	status.MaxChildrenReached,
				"total":				status.TotalProcesses,
				"details":				mapProcesses,
			},
	}

	return baseEvent, nil
}

