// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/o365audit/poll"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// contentBlob is a poll.Transaction that processes "content blobs":
// aggregations of audit event objects returned by the API.
type contentBlob struct {
	env     apiEnvironment
	id, url string
	// cursor is used to ACK the resulting events.
	cursor checkpoint
	// skipLines is used when resuming from a saved cursor so that already
	// acknowledged objects are not duplicated.
	skipLines int
}

// String returns a printable representation of this transaction.
func (c contentBlob) String() string {
	return fmt.Sprintf("content blob url:%s id:%s", c.url, c.id)
}

// RequestDecorators returns the decorators used to perform a request.
func (c contentBlob) RequestDecorators() []autorest.PrepareDecorator {
	return []autorest.PrepareDecorator{
		autorest.WithBaseURL(c.url),
	}
}

// Delay returns the delay to perform this request.
func (c contentBlob) Delay() time.Duration {
	return 0
}

// OnResponse parses the response for a content blob.
func (c contentBlob) OnResponse(response *http.Response) (actions []poll.Action) {
	if response.StatusCode != 200 {
		return c.handleError(response)
	}
	var raws []json.RawMessage
	if err := readJSONBody(response, &raws); err != nil {
		return append(actions, poll.Terminate(errors.Wrap(err, "reading body failed")))
	}
	entries := make([]mapstr.M, len(raws))
	for idx, raw := range raws {
		var entry mapstr.M
		if err := json.Unmarshal(raw, &entry); err != nil {
			return append(actions, poll.Terminate(errors.Wrap(err, "decoding json failed")))
		}
		entries[idx] = entry
		id, _ := getString(entry, "Id")
		ts, _ := getString(entry, "CreationTime")
		c.env.Logger.Debugf(" > event %d: created:%s id:%s for %s", idx+1, ts, id, c.cursor)
	}
	if len(entries) > c.skipLines {
		for _, entry := range entries[:c.skipLines] {
			id, _ := getString(entry, "Id")
			c.env.Logger.Debugf("Skipping event %s [%s] for %s", c.cursor, id, c.id)
		}
		for idx, entry := range entries[c.skipLines:] {
			c.cursor = c.cursor.ForNextLine()
			c.env.Logger.Debugf("Reporting event %s for %s", c.cursor, c.id)
			actions = append(actions, c.env.Report(raws[idx], entry, c.cursor))
		}
		c.skipLines = 0
	} else {
		for _, entry := range entries {
			id, _ := getString(entry, "Id")
			c.env.Logger.Debugf("Skipping event all %s [%s] for %s", c.cursor, id, c.id)
		}

		c.skipLines -= len(entries)
	}
	// The API only documents the use of NextPageUri header for list requests
	// but one can't be too careful.
	if url, found := getNextPage(response); found {
		return append(actions, poll.Fetch(newPager(url, c)))
	}

	return actions
}

func (c contentBlob) handleError(response *http.Response) (actions []poll.Action) {
	var msg apiError
	readJSONBody(response, &msg)
	c.env.Logger.Warnf("Got error %s: %+v", response.Status, msg)

	if _, found := fatalErrors[msg.Error.Code]; found {
		return []poll.Action{
			c.env.ReportAPIError(msg),
			poll.Terminate(errors.New(msg.Error.Message)),
		}
	}

	switch response.StatusCode {
	case 401: // Authentication error. Renew oauth token and repeat this op.
		return []poll.Action{
			poll.RenewToken(),
			poll.Fetch(withDelay{contentBlob: c, delay: c.env.Config.PollInterval}),
		}
	case 404:
		return nil
	}
	if msg.Error.Code != "" {
		actions = append(actions, c.env.ReportAPIError(msg))
	}
	return append(actions, poll.Fetch(withDelay{contentBlob: c, delay: c.env.Config.ErrorRetryInterval}))
}

// ContentBlob creates a new contentBlob.
func ContentBlob(url string, cursor checkpoint, env apiEnvironment) contentBlob {
	return contentBlob{
		url:    url,
		env:    env,
		cursor: cursor,
	}
}

// WithID configures a content blob with the given origin ID.
func (c contentBlob) WithID(id string) contentBlob {
	c.id = id
	return c
}

// WithSkipLines configures a content blob with the number of objects to skip.
func (c contentBlob) WithSkipLines(nlines int) contentBlob {
	c.skipLines = nlines
	return c
}

type withDelay struct {
	contentBlob
	delay time.Duration
}

// Delay overrides the contentBlob's delay.
func (w withDelay) Delay() time.Duration {
	return w.delay
}
