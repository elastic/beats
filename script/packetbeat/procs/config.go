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

package procs

import "time"

type ProcsConfig struct {
	Enabled         bool          `config:"enabled"`
	MaxProcReadFreq time.Duration `config:"max_proc_read_freq"`
	Monitored       []ProcConfig  `config:"monitored"`
	RefreshPidsFreq time.Duration `config:"refresh_pids_freq"`
}

type ProcConfig struct {
	Process     string `config:"process"`
	CmdlineGrep string `config:"cmdline_grep"`
}
