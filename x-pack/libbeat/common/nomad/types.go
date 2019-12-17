// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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

// Desired status for a given allocation
const (
	AllocDesiredStatusRun   = api.AllocDesiredStatusRun   // Allocation should run
	AllocDesiredStatusStop  = api.AllocDesiredStatusStop  // Allocation should stop
	AllocDesiredStatusEvict = api.AllocDesiredStatusEvict // Allocation should stop, and was evicted
)
