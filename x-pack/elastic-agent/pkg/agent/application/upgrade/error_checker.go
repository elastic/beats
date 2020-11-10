// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	statusCheckPeriod = 30 * time.Second
)

// ErrAgentStatusFailed is returned when agent reports FAILED status.
var ErrAgentStatusFailed = errors.New("agent in a failed state", errors.TypeApplication)

// ErrorChecker checks agent for status change and sends an error to a channel if found.
type ErrorChecker struct {
	notifyChan  chan error
	log         *logger.Logger
	agentClient client.Client
}

// NewErrorChecker creates a new error checker.
func NewErrorChecker(ch chan error, log *logger.Logger) (*ErrorChecker, error) {
	c := client.New()
	ec := &ErrorChecker{
		notifyChan:  ch,
		agentClient: c,
		log:         log,
	}

	return ec, nil
}

// Run runs the checking loop.
func (ch ErrorChecker) Run(ctx context.Context) {
	ch.log.Debug("Error checker started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(statusCheckPeriod):
			err := ch.agentClient.Connect(ctx)
			if err != nil {
				ch.log.Error(err, "Failed communicating to running daemon", errors.TypeNetwork, errors.M("socket", control.Address()))
				continue
			}

			status, err := ch.agentClient.Status(ctx)
			ch.agentClient.Disconnect()
			if err != nil {
				ch.log.Error("failed retrieving agent status", err)
				// agent is probably not running and this will be detected by pid watcher
				continue
			}

			if status.Status == client.Failed {
				ch.log.Error("error checker notifying failure of agent")
				ch.notifyChan <- ErrAgentStatusFailed
			}

			for _, app := range status.Applications {
				if app.Status == client.Failed {
					err = multierror.Append(err, errors.New(fmt.Sprintf("application %s[%v] failed: %s", app.Name, app.ID, app.Message)))
				}
			}

			if err != nil {
				ch.log.Error("error checker notifying failure of applications")
				ch.notifyChan <- errors.New(err, "applications in a failed state", errors.TypeApplication)
			}
		}
	}
}
