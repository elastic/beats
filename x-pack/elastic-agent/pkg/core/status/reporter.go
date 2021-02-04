// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// AgentStatusCode is the status code for the Elastic Agent overall.
type AgentStatusCode int

const (
	// Healthy status means everything is fine.
	Healthy AgentStatusCode = iota
	// Degraded status means something minor is preventing agent to work properly.
	Degraded
	// Failed status means agent is unable to work properly.
	Failed
)

var (
	humanReadableStatuses = map[AgentStatusCode]string{
		Healthy:  "online",
		Degraded: "degraded",
		Failed:   "error",
	}
)

// AgentApplicationStatus returns the status of specific application.
type AgentApplicationStatus struct {
	ID      string
	Name    string
	Status  state.Status
	Message string
}

// AgentStatus returns the overall status of the Elastic Agent.
type AgentStatus struct {
	Status       AgentStatusCode
	Message      string
	Applications []AgentApplicationStatus
}

// Controller takes track of component statuses.
type Controller interface {
	RegisterComponent(string) Reporter
	RegisterApp(id string, name string) Reporter
	Status() AgentStatus
	StatusCode() AgentStatusCode
	StatusString() string
	UpdateStateID(string)
}

type controller struct {
	lock         sync.Mutex
	status       AgentStatusCode
	reporters    map[string]*reporter
	appReporters map[string]*reporter
	log          *logger.Logger
	stateID      string
}

// NewController creates a new reporter.
func NewController(log *logger.Logger) Controller {
	return &controller{
		status:       Healthy,
		reporters:    make(map[string]*reporter),
		appReporters: make(map[string]*reporter),
		log:          log,
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
	// cleanup status for component reporters
	// the status of app reports remain the same
	for _, rep := range r.reporters {
		if !rep.isRegistered {
			continue
		}

		rep.lock.Lock()
		rep.status = state.Configuring
		rep.message = ""
		rep.lock.Unlock()
	}
	r.lock.Unlock()

	r.updateStatus()
}

// Register registers new component for status updates.
func (r *controller) RegisterComponent(componentIdentifier string) Reporter {
	id := componentIdentifier + "-" + uuid.New().String()[:8]
	rep := &reporter{
		name:         componentIdentifier,
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

// RegisterApp registers new component for status updates.
func (r *controller) RegisterApp(componentIdentifier string, name string) Reporter {
	id := componentIdentifier + "-" + uuid.New().String()[:8]
	rep := &reporter{
		name:         name,
		status:       state.Stopped,
		isRegistered: true,
		unregisterFunc: func() {
			r.lock.Lock()
			delete(r.appReporters, id)
			r.lock.Unlock()
		},
		notifyChangeFunc: r.updateStatus,
	}

	r.lock.Lock()
	r.appReporters[id] = rep
	r.lock.Unlock()

	return rep
}

// Status retrieves current agent status.
func (r *controller) Status() AgentStatus {
	r.lock.Lock()
	defer r.lock.Unlock()
	apps := make([]AgentApplicationStatus, len(r.appReporters))
	i := 0
	for key, rep := range r.appReporters {
		rep.lock.Lock()
		apps[i] = AgentApplicationStatus{
			ID:      key,
			Name:    rep.name,
			Status:  rep.status,
			Message: rep.message,
		}
		rep.lock.Unlock()
		i++
	}
	return AgentStatus{
		Status:       r.status,
		Message:      "",
		Applications: apps,
	}
}

// StatusCode retrieves current agent status code.
func (r *controller) StatusCode() AgentStatusCode {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.status
}

func (r *controller) updateStatus() {
	status := Healthy

	r.lock.Lock()
	for id, rep := range r.reporters {
		s := statusToAgentStatus(rep.status)
		if s > status {
			status = s
		}

		r.log.Debugf("'%s' has status '%s'", id, humanReadableStatuses[s])
		if status == Failed {
			break
		}
	}
	if status != Failed {
		for id, rep := range r.appReporters {
			s := statusToAgentStatus(rep.status)
			if s > status {
				status = s
			}

			r.log.Debugf("'%s' has status '%s'", id, humanReadableStatuses[s])
			if status == Failed {
				break
			}
		}
	}

	if r.status != status {
		r.logStatus(status)
		r.status = status
	}

	r.lock.Unlock()

}

func (r *controller) logStatus(status AgentStatusCode) {
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
	return humanReadableStatuses[r.StatusCode()]
}

// Reporter reports status of component
type Reporter interface {
	Update(state.Status, string)
	Unregister()
}

type reporter struct {
	name             string
	lock             sync.Mutex
	isRegistered     bool
	status           state.Status
	message          string
	unregisterFunc   func()
	notifyChangeFunc func()
}

// Update updates the status of a component.
func (r *reporter) Update(s state.Status, message string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.isRegistered {
		return
	}
	r.message = message
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

func statusToAgentStatus(status state.Status) AgentStatusCode {
	s := status.ToProto()
	if s == proto.StateObserved_DEGRADED {
		return Degraded
	}
	if s == proto.StateObserved_FAILED {
		return Failed
	}
	return Healthy
}
