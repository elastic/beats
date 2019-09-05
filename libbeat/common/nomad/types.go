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

package nomad

import (
	api "github.com/hashicorp/nomad/api"
	// api "github.com/hashicorp/nomad/nomad/structs"
)

// Resource contains data about a nomad allocation
type Resource = api.Allocation

// Job is the main organization unit in Nomad lingo
type Job = api.Job

// TaskGroup contains a group of tasks that will be allocated in the same node
type TaskGroup = api.TaskGroup

// Client is the interface for executing queries against a Nomad agent
type Client = api.Client
