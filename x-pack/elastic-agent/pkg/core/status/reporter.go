// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"sync"

	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
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
	Register(string) Reporter
	Status() AgentStatus
	StatusString() string
	UpdateStateID(string)
}

type controller struct {
	lock      sync.Mutex
	status    AgentStatus
	reporters map[string]*reporter
	log       *logger.Logger
	stateID   string
}

// NewController creates a new reporter.
func NewController(log *logger.Logger) Controller {
	return &controller{
		status:    Healthy,
		reporters: make(map[string]*reporter),
		log:       log,
	}
}

// UpdateStateID cleans health when new configuration is received.
// To prevent reporting failures from previous configuration.
func (r *controller) UpdateStateID(stateID string) {
	if stateID == r.stateID {
		return
	}

	r.lock.Lock()

	r.stateID = stateID
	// cleanup status
	for _, rep := range r.reporters {
		if !rep.isRegistered {
			continue
		}

		rep.lock.Lock()
		rep.status = Healthy
		rep.lock.Unlock()
	}
	r.lock.Unlock()

	r.updateStatus()
}

// Register registers new component for status updates.
func (r *controller) Register(componentIdentifier string) Reporter {
	id := componentIdentifier + "-" + uuid.New().String()[:8]
	rep := &reporter{
		isRegistered: true,
		unregisterFunc: func() {
			r.lock.Lock()
			delete(r.reporters, id)
			r.lock.Unlock()
		},
		notifyChangeFunc: r.updateStatus,
	}

	r.lock.Lock()
	r.reporters[id] = rep
	r.lock.Unlock()

	return rep
}

// Status retrieves current agent status.
func (r *controller) Status() AgentStatus {
	return r.status
}

func (r *controller) updateStatus() {
	status := Healthy

	r.lock.Lock()
	for id, rep := range r.reporters {
		s := rep.status
		if s > status {
			status = s
		}

		r.log.Debugf("'%s' has status '%s'", id, humanReadableStatuses[s])
		if status == Failed {
			break
		}
	}

	if r.status != status {
		r.logStatus(status)
		r.status = status
	}

	r.lock.Unlock()

}

func (r *controller) logStatus(status AgentStatus) {
	logFn := r.log.Infof
	if status == Degraded {
		logFn = r.log.Warnf
	} else if status == Failed {
		logFn = r.log.Errorf
	}

	logFn("Elastic Agent status changed to: '%s'", humanReadableStatuses[status])
}

// StatusString retrieves human readable string of current agent status.
func (r *controller) StatusString() string {
	return humanReadableStatuses[r.Status()]
}

// Reporter reports status of component
type Reporter interface {
	Update(AgentStatus)
	Unregister()
}

type reporter struct {
	lock             sync.Mutex
	isRegistered     bool
	status           AgentStatus
	unregisterFunc   func()
	notifyChangeFunc func()
}

// Update updates the status of a component.
func (r *reporter) Update(s AgentStatus) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.isRegistered {
		return
	}
	if r.status != s {
		r.status = s
		r.notifyChangeFunc()
	}
}

// Unregister unregister status from reporter. Reporter will no longer be taken into consideration
// for overall status computation.
func (r *reporter) Unregister() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.isRegistered = false
	r.unregisterFunc()
	r.notifyChangeFunc()
}
