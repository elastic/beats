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

package ecs

import (
	"time"
)

// These fields contain information about a process.
// These fields can help you correlate metrics information with a process
// id/name from a log message.  The `process.pid` often stays in the metric
// itself and is copied to the global field for correlation.
type Process struct {
	// Parent process.
	Parent *Process `ecs:"parent"`

	// Process id.
	PID int64 `ecs:"pid"`

	// Unique identifier for the process.
	// The implementation of this is specified by the data source, but some
	// examples of what could be used here are a process-generated UUID, Sysmon
	// Process GUIDs, or a hash of some uniquely identifying components of a
	// process.
	// Constructing a globally unique identifier is a common practice to
	// mitigate PID reuse as well as to identify a specific process over time,
	// across multiple monitored hosts.
	EntityID string `ecs:"entity_id"`

	// Process name.
	// Sometimes called program name or similar.
	Name string `ecs:"name"`

	// Identifier of the group of processes the process belongs to.
	PGID int64 `ecs:"pgid"`

	// Full command line that started the process, including the absolute path
	// to the executable, and all arguments.
	// Some arguments may be filtered to protect sensitive information.
	CommandLine string `ecs:"command_line"`

	// Array of process arguments, starting with the absolute path to the
	// executable.
	// May be filtered to protect sensitive information.
	Args []string `ecs:"args"`

	// Length of the process.args array.
	// This field can be useful for querying or performing bucket analysis on
	// how many arguments were provided to start a process. More arguments may
	// be an indication of suspicious activity.
	ArgsCount int64 `ecs:"args_count"`

	// Absolute path to the process executable.
	Executable string `ecs:"executable"`

	// Process title.
	// The proctitle, some times the same as process name. Can also be
	// different: for example a browser setting its title to the web page
	// currently opened.
	Title string `ecs:"title"`

	// Thread ID.
	ThreadID int64 `ecs:"thread.id"`

	// Thread name.
	ThreadName string `ecs:"thread.name"`

	// The time the process started.
	Start time.Time `ecs:"start"`

	// Seconds the process has been up.
	Uptime int64 `ecs:"uptime"`

	// The working directory of the process.
	WorkingDirectory string `ecs:"working_directory"`

	// The exit code of the process, if this is a termination event.
	// The field should be absent if there is no exit code for the event (e.g.
	// process start).
	ExitCode int64 `ecs:"exit_code"`

	// The time the process ended.
	End time.Time `ecs:"end"`
}
