// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/o365audit/poll"
)

// Subscribe is a poll.Transaction that subscribes to an event stream.
type subscribe struct {
	apiEnvironment
}

// String returns the printable representation of a subscribe transaction.
func (s subscribe) String() string {
	return fmt.Sprintf("subscribe tenant:%s contentType:%s", s.TenantID, s.ContentType)
}

// RequestDecorators returns the decorators used to perform a request.
func (s subscribe) RequestDecorators() []autorest.PrepareDecorator {
	return []autorest.PrepareDecorator{
		autorest.AsPost(),
		autorest.WithBaseURL(s.Config.Resource),
		autorest.WithPath("api/v1.0"),
		autorest.WithPath(s.TenantID),
		autorest.WithPath("activity/feed/subscriptions/start"),
		autorest.WithQueryParameters(
			map[string]interface{}{
				"contentType": s.ContentType,
			}),
	}
}

// OnResponse handles the output of a list content request.
func (s subscribe) OnResponse(response *http.Response) []poll.Action {
	if response.StatusCode != 200 {
		return s.handleError(response)
	}
	var js subscribeResponse
	if err := readJSONBody(response, &js); err != nil {
		return []poll.Action{
			poll.Terminate(err),
		}
	}
	if js.Status != "enabled" {
		return []poll.Action{
			poll.Terminate(fmt.Errorf("unable to subscribe. Got status: %s", js.Status)),
		}
	}
	return nil
}

func (s subscribe) handleError(response *http.Response) []poll.Action {
	var msg apiError
	if err := readJSONBody(response, &msg); err != nil {
		return []poll.Action{poll.Terminate(err)}
	}
	return []poll.Action{
		poll.Terminate(fmt.Errorf("got an error when subscribing: %s body: %+v", response.Status, msg)),
	}
}

// Delay returns the delay before executing a transaction.
func (s subscribe) Delay() time.Duration {
	return time.Second * 5
}

// Subscribe returns an action to subscribe to a stream.
func Subscribe(env apiEnvironment) subscribe {
	return subscribe{
		apiEnvironment: env,
	}
}
