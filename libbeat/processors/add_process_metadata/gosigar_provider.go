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

//go:build ignore
// +build ignore

package add_process_metadata

import (
	"strings"
	"time"

	"github.com/menderesk/gosigar"
)

type gosigarProvider struct{}

func (p gosigarProvider) GetProcessMetadata(pid int) (result *processMetadata, err error) {
	var procExe gosigar.ProcExe
	var procArgs gosigar.ProcArgs
	var procEnv gosigar.ProcEnv
	var procState gosigar.ProcState
	var procTime gosigar.ProcTime

	for _, act := range []struct {
		getter   func(int) error
		required bool
	}{
		{procExe.Get, true},
		{procArgs.Get, false},
		{procEnv.Get, false},
		{procState.Get, false},
		{procTime.Get, false},
	} {
		if err := act.getter(pid); err != nil {
			if act.required {
				return nil, err
			}
		}
	}

	r := processMetadata{
		name:      procExe.Name,
		title:     strings.Join(procArgs.List, " "),
		exe:       procExe.Name,
		args:      procArgs.List,
		env:       procEnv.Vars,
		pid:       pid,
		ppid:      procState.Ppid,
		username:  procState.Username,
		startTime: time.Unix(int64(procTime.StartTime/1000), int64(procTime.StartTime%1000)*1000000),
	}
	r.fields = r.toMap()
	return &r, nil
}
