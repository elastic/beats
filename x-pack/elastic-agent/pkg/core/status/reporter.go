// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"sync"

	"github.com/google/uuid"
)

// AgentStatus represents a status of agent.
type AgentStatus int

// UpdateFunc is used by components to notify reporter about status changes.
type UpdateFunc func(AgentStatus)

const (
	// Healthy status means everything is fine.
	Healthy AgentStatus = iota
	// Degraded status means something minor is preventing agent to work properly.
	Degraded
	// Failed status means agent is unable to work properly.
	Failed
)

var (
	humanReadableStatuses = map[AgentStatus]string{
		Healthy:  "online",
		Degraded: "degraded",
		Failed:   "error",
	}
)

// Controller takes track of component statuses.
type Controller interface {
	Register() Reporter
	Status() AgentStatus
	StatusString() string
}

type controller struct {
	lock      sync.Mutex
	status    AgentStatus
	reporters map[string]*reporter
}

// Reporter reports status of component
type Reporter interface {
	Update(AgentStatus)
	Unregister()
}

type reporter struct {
	lock           sync.Mutex
	isRegistered   bool
	status         AgentStatus
	unregisterFunc func()
}

// Update updates the status of a component.
func (r *reporter) Update(s AgentStatus) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.isRegistered {
		return
	}
	r.status = s
}

// Unregister unregister status from reporter. Reporter will no longer be taken into consideration
// for overall status computation.
func (r *reporter) Unregister() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.isRegistered = false
	r.unregisterFunc()
}

// NewController creates a new reporter.
func NewController() Controller {
	return &controller{
		reporters: make(map[string]*reporter),
		status:    Healthy,
	}
}

// Register registers new component for status updates.
func (r *controller) Register() Reporter {
	id := uuid.New().String()

	rep := &reporter{
		isRegistered: true,
		unregisterFunc: func() {
			r.lock.Lock()
			delete(r.reporters, id)
			r.lock.Unlock()
		},
	}

	r.lock.Lock()
	r.reporters[id] = rep
	r.lock.Unlock()

	return rep
}

// Status retrieves current agent status.
func (r *controller) Status() AgentStatus {
	status := Healthy
	r.lock.Lock()
	for _, r := range r.reporters {
		s := r.status
		if s > status {
			status = s
		}

		if status == Failed {
			break
		}
	}
	r.lock.Unlock()

	return status
}

// StatusString retrieves human readable string of current agent status.
func (r *controller) StatusString() string {
	return humanReadableStatuses[r.Status()]
}
