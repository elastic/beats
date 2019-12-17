// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	"github.com/elastic/beats/x-pack/agent/pkg/reporter"
)

const (
	defaultThreshold = 1000
)

type event struct {
	EventType string                 `json:"type"`
	Ts        fleetapi.Time          `json:"timestamp"`
	SubType   string                 `json:"subtype"`
	Msg       string                 `json:"message"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Data      string                 `json:"data,omitempty"`
}

func (e *event) Type() string {
	return e.EventType
}

func (e *event) Timestamp() time.Time {
	return time.Time(e.Ts)
}

func (e *event) Message() string {
	return e.Msg
}

type checkinExecutor interface {
	Execute(r *fleetapi.CheckinRequest) (*fleetapi.CheckinResponse, error)
}

type remoteClient interface {
	Send(
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)
}

// Reporter is a reporter without any effects, serves just as a showcase for further implementations.
type Reporter struct {
	logger         *logger.Logger
	queue          []reporter.Event
	qlock          sync.Mutex
	ticker         *time.Ticker
	threshold      int
	droppedCounter int
	checkingCmd    checkinExecutor
	closeChan      chan struct{}
	closeOnce      sync.Once
}

type agentInfo interface {
	AgentID() string
}

// NewReporter creates a new fleet reporter.
func NewReporter(agentInfo agentInfo, l *logger.Logger, c *ManagementConfig, client remoteClient) (*Reporter, error) {
	checkinClient := fleetapi.NewCheckinCmd(agentInfo, client)

	frequency := time.Duration(c.ReportingCheckFrequency) * time.Second
	r := &Reporter{
		queue:       make([]reporter.Event, 0),
		ticker:      time.NewTicker(frequency),
		logger:      l,
		checkingCmd: checkinClient,
		threshold:   c.Threshold,
		closeChan:   make(chan struct{}),
	}

	go r.reportLoop()
	return r, nil
}

// Report in noop reporter does nothing.
func (r *Reporter) Report(e reporter.Event) error {
	r.qlock.Lock()
	defer r.qlock.Unlock()

	r.queue = append(r.queue, e)
	if r.threshold > 0 && len(r.queue) > r.threshold {
		r.dropEvent()
	}
	return nil
}

// Close stops all the background jobs reporter is running.
// Guards agains panic of closing channel multiple times.
func (r *Reporter) Close() error {
	r.closeOnce.Do(func() { close(r.closeChan) })
	return nil
}

func (r *Reporter) reportLoop() {
	for {
		select {
		case <-r.ticker.C:
		case <-r.closeChan:
			r.logger.Info("stop received, cancelling the fleet report loop")
			return
		}

		// report all events up to this point
		r.qlock.Lock()
		batch := r.queueCopy()
		r.droppedCounter = 0
		r.qlock.Unlock()

		if err := r.reportBatch(batch); err != nil {
			r.logger.Errorf("failed to report event batch: %v", err)
			continue
		}

		// shrink
		r.qlock.Lock()

		// in case some event are dropped decrease size to avoid event-loss
		if size := len(batch) - r.droppedCounter; size > 0 {
			r.queue = r.queue[size:]
		}
		r.qlock.Unlock()
	}
}

func (r *Reporter) queueCopy() []reporter.Event {
	size := len(r.queue)
	batch := make([]reporter.Event, size)

	copy(batch, r.queue)
	return batch
}

func (r *Reporter) reportBatch(ee []reporter.Event) error {
	req := &fleetapi.CheckinRequest{
		Events: make([]fleetapi.SerializableEvent, 0, len(ee)),
	}

	for _, e := range ee {
		req.Events = append(req.Events,
			&event{
				EventType: e.Type(),
				Ts:        fleetapi.Time(e.Time()),
				SubType:   e.SubType(),
				Msg:       e.Message(),
				Payload:   e.Payload(),
				Data:      e.Data(),
			})
	}

	_, err := r.checkingCmd.Execute(req)
	return err
}

func (r *Reporter) dropEvent() {
	r.droppedCounter++
	if dropped := r.tryDropInfo(); !dropped {
		r.dropFirst()
	}
}

// tryDropInfo returns true if info was found and dropped.
func (r *Reporter) tryDropInfo() bool {
	for i, e := range r.queue {
		if e.Type() != reporter.EventTypeError {
			r.queue = append(r.queue[:i], r.queue[i+1:]...)
			r.logger.Infof("fleet reporter dropped event because threshold[%d] was reached: %v", r.threshold, e)
			return true
		}
	}

	return false
}

func (r *Reporter) dropFirst() {
	r.queue = r.queue[1:]
}

// Check it is reporter.Backend.
var _ reporter.Backend = &Reporter{}
