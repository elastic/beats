// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rollback

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/hashicorp/go-multierror"
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
	err := c.Connect(context.Background())
	if err != nil {
		return nil, errors.New(err, "Failed communicating to running daemon", errors.TypeNetwork, errors.M("socket", control.Address()))
	}

	ec := &ErrorChecker{
		notifyChan:  ch,
		agentClient: c,
		log:         log,
	}

	return ec, nil
}

// Run runs the checking loop.
func (ch ErrorChecker) Run(ctx context.Context) {
	defer ch.agentClient.Disconnect()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(statusCheckPeriod):
			status, err := ch.agentClient.Status(ctx)
			if err != nil {
				ch.log.Error("failed retrieving agent status", err)
				// agent is probably not running and this will be detected by pid watcher
				continue
			}

			if status.Status == client.Failed {
				ch.notifyChan <- ErrAgentStatusFailed
			}

			for _, app := range status.Applications {
				if app.Status == client.Failed {
					err = multierror.Append(err, errors.New(fmt.Sprintf("application %s[%v] failed: %s", app.Name, app.ID, app.Message)))
				}
			}

			if err != nil {
				ch.notifyChan <- errors.New(err, "applications in a failed state", errors.TypeApplication)
			}
		}
	}
}
